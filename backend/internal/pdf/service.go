package pdf

import (
	"context"
	"errors"
	"fmt"

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

// Generate só cria o registro PENDING e devolve — o trabalho pesado roda
// depois, em background, via Process (seção 30: nunca bloquear a
// requisição HTTP que pediu a geração).
func (s *Service) Generate(ctx context.Context, bookletID, requestedBy int64) (int64, error) {
	return s.Repo.CreateDocument(ctx, bookletID, requestedBy)
}

// Process é o trabalho de fato: garante o snapshot (selecionando e
// congelando na primeira vez — seções 25/26/27), monta o HTML, aciona o
// Chromium e grava o resultado. Recebe um contexto criado pelo chamador
// especificamente para isso (não o da requisição HTTP, que já teria sido
// cancelado quando o handler retornou) e nunca deixa um registro preso em
// PROCESSING: todo caminho de erro marca FAILED com uma mensagem.
func (s *Service) Process(ctx context.Context, documentID, bookletID int64) {
	if err := s.Repo.MarkProcessing(ctx, documentID); err != nil {
		return
	}

	if err := s.ensureSnapshot(ctx, bookletID); err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, err.Error())
		return
	}

	snapshots, err := s.Repo.LoadSnapshots(ctx, bookletID)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not load snapshots: %v", err))
		return
	}
	if len(snapshots) == 0 {
		_ = s.Repo.MarkFailed(ctx, documentID, "nenhuma questão selecionada para este caderno")
		return
	}

	booklet, err := s.Booklets.FindByID(ctx, bookletID)
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
	documentHTML, err := BuildDocument(title, snapshots)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not build document: %v", err))
		return
	}

	pdfBytes, err := RenderPDF(ctx, s.ChromePath, documentHTML, title)
	if err != nil {
		_ = s.Repo.MarkFailed(ctx, documentID, fmt.Sprintf("could not render pdf: %v", err))
		return
	}

	relativePath, err := s.Storage.Save(application.ID, booklet.ID, documentID, pdfBytes)
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
		return errors.New("questões elegíveis insuficientes para uma ou mais cotas no momento da geração")
	default:
		return err
	}
}
