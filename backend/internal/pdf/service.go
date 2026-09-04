package pdf

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"boraif/internal/applications"
	"boraif/internal/booklets"
)

// Service orquestra a geração de PDF (seção 28/29/30), combinando o
// repositório deste pacote (snapshots + generated_documents) com os
// repositórios de aplicações e cadernos.
type Service struct {
	Repo         *Repository
	Booklets     *booklets.Repository
	Applications *applications.Repository
	Storage      *Storage
	ChromePath   string
}

// Generate garante o snapshot e os tipos de prova do caderno (seções
// 25/26/27 + variantes — rápido, só banco, por isso roda antes de
// responder) e cria um par de documentos PENDING — prova (KindExam) e
// gabarito (KindAnswerKey) — para cada tipo configurado. A etapa lenta
// (Chromium) fica para Process, chamado depois em background para cada
// documento criado (seção 30: nunca bloquear a requisição HTTP com isso).
func (s *Service) Generate(ctx context.Context, bookletID, requestedBy int64) ([]int64, error) {
	if err := s.ensureSnapshot(ctx, bookletID); err != nil {
		return nil, err
	}

	variants, err := s.Repo.ListVariants(ctx, bookletID)
	if err != nil {
		return nil, err
	}

	documentIDs := make([]int64, 0, len(variants)*2)
	for _, v := range variants {
		examID, err := s.Repo.CreateDocument(ctx, bookletID, v.ID, requestedBy, KindExam)
		if err != nil {
			return nil, err
		}
		documentIDs = append(documentIDs, examID)

		keyID, err := s.Repo.CreateDocument(ctx, bookletID, v.ID, requestedBy, KindAnswerKey)
		if err != nil {
			return nil, err
		}
		documentIDs = append(documentIDs, keyID)
	}
	return documentIDs, nil
}

// ProcessAll roda Process para cada documento de uma geração, um de cada
// vez — evita várias instâncias do Chromium disputando recursos ao mesmo
// tempo (um caderno com 4 tipos já gera 8 documentos numa única geração).
func (s *Service) ProcessAll(ctx context.Context, documentIDs []int64) {
	for _, id := range documentIDs {
		s.Process(ctx, id)
	}
}

// Process é o trabalho de fato de UM documento (prova ou gabarito de um
// tipo específico): monta o HTML, aciona o Chromium e grava o resultado.
// Recebe um contexto criado pelo chamador especificamente para isso (não o
// da requisição HTTP, que já teria sido cancelado quando o handler
// retornou) e nunca deixa um registro preso em PROCESSING: todo caminho de
// erro marca FAILED com uma mensagem.
func (s *Service) Process(ctx context.Context, documentID int64) {
	doc, err := s.Repo.FindDocumentByID(ctx, documentID)
	if err != nil {
		return
	}
	if err := s.Repo.MarkProcessing(ctx, documentID); err != nil {
		return
	}
	if doc.VariantID == nil {
		_ = s.Repo.MarkFailed(ctx, documentID, "documento sem tipo de prova associado")
		return
	}

	variant, err := s.Repo.FindVariantByID(ctx, *doc.VariantID)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not load variant: %v", err))
		return
	}

	questions, err := s.Repo.LoadVariantQuestions(ctx, variant.ID)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not load variant questions: %v", err))
		return
	}
	if len(questions) == 0 {
		_ = s.Repo.MarkFailed(ctx, documentID, "nenhuma questão selecionada para este caderno")
		return
	}

	booklet, err := s.Booklets.FindByID(ctx, doc.BookletID)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not load booklet: %v", err))
		return
	}
	application, err := s.Applications.FindByID(ctx, booklet.ApplicationID)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not load application: %v", err))
		return
	}

	title := application.Name + " — " + booklet.Name

	var documentHTML string
	switch doc.Kind {
	case KindExam:
		documentHTML, err = BuildExamDocument(title, variant.VariantNumber, questions)
	case KindAnswerKey:
		documentHTML, err = BuildAnswerKeyDocument(title, variant.VariantNumber, questions)
	default:
		err = fmt.Errorf("tipo de documento desconhecido: %s", doc.Kind)
	}
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not build document: %v", err))
		return
	}

	headerTitle := title + " — Tipo " + strconv.Itoa(variant.VariantNumber)
	pdfBytes, err := RenderPDF(ctx, s.ChromePath, documentHTML, headerTitle)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not render pdf: %v", err))
		return
	}

	relativePath, err := s.Storage.Save(application.ID, booklet.ID, variant.VariantNumber, doc.Kind, documentID, pdfBytes)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not save pdf: %v", err))
		return
	}

	_ = s.Repo.MarkCompleted(ctx, documentID, relativePath)
}

func (s *Service) ensureSnapshot(ctx context.Context, bookletID int64) error {
	isFrozen, _, err := s.Repo.IsFrozen(ctx, bookletID)
	if err != nil {
		return err
	}
	if isFrozen {
		return nil
	}

	err = s.Repo.SelectAndSnapshot(ctx, bookletID)
	switch {
	case errors.Is(err, ErrAlreadyFrozen):
		// corrida: outra geração já congelou entre o IsFrozen acima e agora.
		return nil
	case errors.Is(err, ErrInsufficient):
		return fmt.Errorf("%w: questões elegíveis insuficientes para uma ou mais cotas no momento da geração", ErrInsufficient)
	default:
		return err
	}
}
