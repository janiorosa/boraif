package pdf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyFrozen = errors.New("booklet configuration already frozen")
	ErrInsufficient  = errors.New("not enough eligible questions for one or more quotas")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// IsFrozen indica se a configuração do caderno já foi congelada (seção 27),
// ou seja, se as questões dele já foram selecionadas e o snapshot já existe.
func (r *Repository) IsFrozen(ctx context.Context, bookletID int64) (bool, int64, error) {
	var configID int64
	var isFrozen bool
	err := r.pool.QueryRow(ctx, `
		SELECT id, is_frozen FROM booklet_configurations WHERE booklet_id = $1
	`, bookletID).Scan(&configID, &isFrozen)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, ErrNotFound
	}
	return isFrozen, configID, err
}


type quotaRow struct {
	DisciplineID int64
	SubjectID    *int64
	DifficultyID *int64
	Quantity     int
}

// SelectAndSnapshot escolhe aleatoriamente as questões elegíveis de cada
// cota (seção 25: só status com eligible_for_exam), congela a representação
// delas (seção 26), gera os "tipos de prova" configurados (variant_count —
// mesmas questões, ordem diferente) e marca a configuração como congelada
// (seção 27) — tudo numa única transação. Se qualquer cota não tiver
// questões suficientes *no momento exato da geração* (podem ter mudado
// desde a última checagem de disponibilidade), a transação inteira é
// desfeita e ErrInsufficient é devolvido: nunca fica um snapshot parcial.
//
// Diferente da versão anterior (que embaralhava todas as questões juntas),
// aqui as questões de uma mesma disciplina ficam sempre num bloco contíguo
// de posições — a ordem dos blocos segue a primeira vez que cada disciplina
// aparece entre as cotas. Isso é o que permite aos tipos de prova
// reordenarem só *dentro* de cada bloco, preservando "questões 1-10 são de
// Matemática em todos os tipos", como pedido.
func (r *Repository) SelectAndSnapshot(ctx context.Context, bookletID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var configID int64
	var isFrozen bool
	var variantCount int
	err = tx.QueryRow(ctx, `
		SELECT id, is_frozen, variant_count FROM booklet_configurations WHERE booklet_id = $1 FOR UPDATE
	`, bookletID).Scan(&configID, &isFrozen, &variantCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isFrozen {
		return ErrAlreadyFrozen
	}

	years, err := idsFor(ctx, tx, `SELECT grade_year_id FROM booklet_configuration_grade_years WHERE configuration_id = $1`, configID)
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT discipline_id, subject_id, difficulty_id, quantity FROM booklet_quota_rules WHERE configuration_id = $1 ORDER BY id
	`, configID)
	if err != nil {
		return err
	}
	var quotas []quotaRow
	for rows.Next() {
		var q quotaRow
		if err := rows.Scan(&q.DisciplineID, &q.SubjectID, &q.DifficultyID, &q.Quantity); err != nil {
			rows.Close()
			return err
		}
		quotas = append(quotas, q)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Agrupa os IDs selecionados por disciplina, preservando a ordem de
	// primeira aparição entre as cotas — garante blocos contíguos por
	// disciplina mesmo que o admin tenha cadastrado cotas da mesma
	// disciplina de forma não-consecutiva (ex.: Matemática/Fácil, Física,
	// Matemática/Difícil).
	type disciplineBlock struct {
		disciplineID int64
		questionIDs  []int64
	}
	var blocks []*disciplineBlock
	blockByDiscipline := map[int64]*disciplineBlock{}

	for _, q := range quotas {
		qrows, err := tx.Query(ctx, `
			SELECT q.id
			FROM questions q
			JOIN question_statuses st ON st.id = q.status_id
			WHERE q.discipline_id = $1
			  AND ($2::bigint IS NULL OR q.subject_id = $2)
			  AND ($3::bigint IS NULL OR q.difficulty_id = $3)
			  AND q.grade_year_id = ANY($4)
			  AND st.eligible_for_exam
			ORDER BY random()
			LIMIT $5
		`, q.DisciplineID, q.SubjectID, q.DifficultyID, years, q.Quantity)
		if err != nil {
			return err
		}
		var picked []int64
		for qrows.Next() {
			var id int64
			if err := qrows.Scan(&id); err != nil {
				qrows.Close()
				return err
			}
			picked = append(picked, id)
		}
		qrows.Close()
		if err := qrows.Err(); err != nil {
			return err
		}
		if len(picked) < q.Quantity {
			return ErrInsufficient
		}

		blk, ok := blockByDiscipline[q.DisciplineID]
		if !ok {
			blk = &disciplineBlock{disciplineID: q.DisciplineID}
			blockByDiscipline[q.DisciplineID] = blk
			blocks = append(blocks, blk)
		}
		blk.questionIDs = append(blk.questionIDs, picked...)
	}

	type blockRange struct{ start, end int } // posições 1-based, inclusive, dentro do caderno
	var selectedIDs []int64
	var blockRanges []blockRange
	pos := 1
	for _, blk := range blocks {
		start := pos
		selectedIDs = append(selectedIDs, blk.questionIDs...)
		pos += len(blk.questionIDs)
		blockRanges = append(blockRanges, blockRange{start: start, end: pos - 1})
	}

	snapshotIDs := make([]int64, len(selectedIDs))
	altsBySnapshot := make(map[int64][]SnapshotAlternative, len(selectedIDs))

	for i, questionID := range selectedIDs {
		var stmt, cmd json.RawMessage
		var disciplineName, subjectName, gradeYearName, difficultyName string
		err := tx.QueryRow(ctx, `
			SELECT q.statement_json, q.command_json, d.name, sub.name, gy.name, df.name
			FROM questions q
			JOIN disciplines d ON d.id = q.discipline_id
			JOIN subjects sub ON sub.id = q.subject_id
			JOIN grade_years gy ON gy.id = q.grade_year_id
			JOIN difficulties df ON df.id = q.difficulty_id
			WHERE q.id = $1
		`, questionID).Scan(&stmt, &cmd, &disciplineName, &subjectName, &gradeYearName, &difficultyName)
		if err != nil {
			return err
		}

		altRows, err := tx.Query(ctx, `
			SELECT position, content_json, is_correct FROM question_alternatives WHERE question_id = $1 ORDER BY position
		`, questionID)
		if err != nil {
			return err
		}
		var alternatives []SnapshotAlternative
		for altRows.Next() {
			var pos int16
			var content json.RawMessage
			var isCorrect bool
			if err := altRows.Scan(&pos, &content, &isCorrect); err != nil {
				altRows.Close()
				return err
			}
			alternatives = append(alternatives, SnapshotAlternative{Position: positionLetter(pos), Content: content, IsCorrect: isCorrect})
		}
		altRows.Close()
		if err := altRows.Err(); err != nil {
			return err
		}
		alternativesJSON, err := json.Marshal(alternatives)
		if err != nil {
			return err
		}

		var snapshotID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO booklet_question_snapshots
				(booklet_id, question_id, position_in_booklet, discipline_name, subject_name, grade_year_name, difficulty_name,
				 statement_json, command_json, alternatives_json)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, bookletID, questionID, i+1, disciplineName, subjectName, gradeYearName, difficultyName, stmt, cmd, alternativesJSON).
			Scan(&snapshotID); err != nil {
			return err
		}
		snapshotIDs[i] = snapshotID
		altsBySnapshot[snapshotID] = alternatives
	}

	// Um tipo de prova por variant_count: mesmo conjunto de questões, mesmos
	// blocos por disciplina nas mesmas posições — só a ordem dentro de cada
	// bloco e a ordem das 5 alternativas de cada questão mudam.
	for v := 1; v <= variantCount; v++ {
		var variantID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO booklet_variants (booklet_id, variant_number) VALUES ($1, $2) RETURNING id
		`, bookletID, v).Scan(&variantID); err != nil {
			return err
		}

		variantOrder := make([]int64, len(snapshotIDs))
		for _, br := range blockRanges {
			sub := append([]int64(nil), snapshotIDs[br.start-1:br.end]...)
			rand.Shuffle(len(sub), func(i, j int) { sub[i], sub[j] = sub[j], sub[i] })
			copy(variantOrder[br.start-1:br.end], sub)
		}

		for i, snapID := range variantOrder {
			alts := altsBySnapshot[snapID]
			order := []int{0, 1, 2, 3, 4}
			rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

			altOrderLetters := make([]string, 0, len(order))
			var correctLetter string
			for newPos, origIdx := range order {
				altOrderLetters = append(altOrderLetters, alts[origIdx].Position)
				if alts[origIdx].IsCorrect {
					correctLetter = positionLetter(int16(newPos + 1))
				}
			}
			altOrderJSON, err := json.Marshal(altOrderLetters)
			if err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO booklet_variant_questions (variant_id, snapshot_id, position_in_variant, alternative_order, correct_letter)
				VALUES ($1, $2, $3, $4, $5)
			`, variantID, snapID, i+1, altOrderJSON, correctLetter); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE booklet_configurations SET is_frozen = true, updated_at = now() WHERE id = $1`, configID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListVariants devolve os tipos de prova já gerados para um caderno (vazio
// se a configuração ainda não foi congelada), em ordem de tipo.
func (r *Repository) ListVariants(ctx context.Context, bookletID int64) ([]Variant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, booklet_id, variant_number FROM booklet_variants WHERE booklet_id = $1 ORDER BY variant_number
	`, bookletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Variant
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.BookletID, &v.VariantNumber); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *Repository) FindVariantByID(ctx context.Context, variantID int64) (Variant, error) {
	var v Variant
	err := r.pool.QueryRow(ctx, `
		SELECT id, booklet_id, variant_number FROM booklet_variants WHERE id = $1
	`, variantID).Scan(&v.ID, &v.BookletID, &v.VariantNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return Variant{}, ErrNotFound
	}
	return v, err
}

// LoadVariantQuestions devolve as questões de um tipo de prova já resolvidas
// para exibição: na ordem impressa daquele tipo, com as alternativas na
// ordem de exibição daquele tipo (IsCorrect recalculado) e a letra correta —
// que junto da posição impressa é exatamente o gabarito daquele tipo.
func (r *Repository) LoadVariantQuestions(ctx context.Context, variantID int64) ([]VariantQuestionDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT vq.snapshot_id, vq.position_in_variant, vq.alternative_order, vq.correct_letter,
		       s.discipline_name, s.subject_name, s.grade_year_name, s.difficulty_name,
		       s.statement_json, s.command_json, s.alternatives_json
		FROM booklet_variant_questions vq
		JOIN booklet_question_snapshots s ON s.id = vq.snapshot_id
		WHERE vq.variant_id = $1
		ORDER BY vq.position_in_variant
	`, variantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []VariantQuestionDetail
	for rows.Next() {
		var d VariantQuestionDetail
		var altOrderJSON, originalAltsJSON json.RawMessage
		if err := rows.Scan(&d.SnapshotID, &d.PositionInVariant, &altOrderJSON, &d.CorrectLetter,
			&d.DisciplineName, &d.SubjectName, &d.GradeYearName, &d.DifficultyName,
			&d.StatementJSON, &d.CommandJSON, &originalAltsJSON); err != nil {
			return nil, err
		}

		var altOrder []string
		if err := json.Unmarshal(altOrderJSON, &altOrder); err != nil {
			return nil, fmt.Errorf("decoding alternative_order for snapshot %d: %w", d.SnapshotID, err)
		}
		var originalAlts []SnapshotAlternative
		if err := json.Unmarshal(originalAltsJSON, &originalAlts); err != nil {
			return nil, fmt.Errorf("decoding alternatives_json for snapshot %d: %w", d.SnapshotID, err)
		}
		byLetter := make(map[string]SnapshotAlternative, len(originalAlts))
		for _, a := range originalAlts {
			byLetter[a.Position] = a
		}

		d.Alternatives = make([]SnapshotAlternative, 0, len(altOrder))
		for i, origLetter := range altOrder {
			newLetter := positionLetter(int16(i + 1))
			orig := byLetter[origLetter]
			d.Alternatives = append(d.Alternatives, SnapshotAlternative{
				Position:  newLetter,
				Content:   orig.Content,
				IsCorrect: newLetter == d.CorrectLetter,
			})
		}

		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *Repository) CreateDocument(ctx context.Context, bookletID, variantID, requestedBy int64, kind string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO generated_documents (booklet_id, variant_id, kind, requested_by) VALUES ($1, $2, $3, $4) RETURNING id
	`, bookletID, variantID, kind, requestedBy).Scan(&id)
	return id, err
}

func (r *Repository) MarkProcessing(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE generated_documents SET status = $1 WHERE id = $2`, StatusProcessing, id)
	return err
}

func (r *Repository) MarkCompleted(ctx context.Context, id int64, filePath string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE generated_documents SET status = $1, file_path = $2, completed_at = now() WHERE id = $3
	`, StatusCompleted, filePath, id)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id int64, message string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE generated_documents SET status = $1, error_message = $2, completed_at = now() WHERE id = $3
	`, StatusFailed, message, id)
	return err
}

func (r *Repository) FindDocumentByID(ctx context.Context, id int64) (GeneratedDocument, error) {
	var d GeneratedDocument
	err := r.pool.QueryRow(ctx, `
		SELECT gd.id, gd.booklet_id, gd.variant_id, bv.variant_number, gd.kind, gd.status, gd.file_path,
		       gd.error_message, gd.requested_by, gd.created_at, gd.completed_at
		FROM generated_documents gd
		LEFT JOIN booklet_variants bv ON bv.id = gd.variant_id
		WHERE gd.id = $1
	`, id).Scan(&d.ID, &d.BookletID, &d.VariantID, &d.VariantNumber, &d.Kind, &d.Status, &d.FilePath, &d.ErrorMessage,
		&d.RequestedBy, &d.CreatedAt, &d.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return GeneratedDocument{}, ErrNotFound
	}
	return d, err
}

// ListDocumentsForBooklet devolve todos os documentos gerados de um caderno,
// de todos os tipos de prova, agrupáveis pelo VariantNumber/Kind de cada um.
func (r *Repository) ListDocumentsForBooklet(ctx context.Context, bookletID int64) ([]GeneratedDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT gd.id, gd.booklet_id, gd.variant_id, bv.variant_number, gd.kind, gd.status, gd.file_path, gd.error_message,
		       gd.requested_by, gd.created_at, gd.completed_at
		FROM generated_documents gd
		LEFT JOIN booklet_variants bv ON bv.id = gd.variant_id
		WHERE gd.booklet_id = $1
		ORDER BY bv.variant_number NULLS LAST, gd.kind, gd.created_at DESC
	`, bookletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GeneratedDocument
	for rows.Next() {
		var d GeneratedDocument
		if err := rows.Scan(&d.ID, &d.BookletID, &d.VariantID, &d.VariantNumber, &d.Kind, &d.Status, &d.FilePath, &d.ErrorMessage,
			&d.RequestedBy, &d.CreatedAt, &d.CompletedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func positionLetter(pos int16) string {
	const letters = "ABCDE"
	if pos < 1 || int(pos) > len(letters) {
		return ""
	}
	return string(letters[pos-1])
}

func idsFor(ctx context.Context, tx pgx.Tx, query string, arg int64) ([]int64, error) {
	rows, err := tx.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
