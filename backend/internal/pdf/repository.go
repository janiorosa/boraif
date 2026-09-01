package pdf

import (
	"context"
	"encoding/json"
	"errors"
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
// delas (seção 26) e marca a configuração como congelada (seção 27) — tudo
// numa única transação. Se qualquer cota não tiver questões suficientes
// *no momento exato da geração* (podem ter mudado desde a última checagem
// de disponibilidade), a transação inteira é desfeita e ErrInsufficient é
// devolvido: nunca fica um snapshot parcial.
func (r *Repository) SelectAndSnapshot(ctx context.Context, bookletID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var configID int64
	var isFrozen bool
	err = tx.QueryRow(ctx, `
		SELECT id, is_frozen FROM booklet_configurations WHERE booklet_id = $1 FOR UPDATE
	`, bookletID).Scan(&configID, &isFrozen)
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
		SELECT discipline_id, subject_id, difficulty_id, quantity FROM booklet_quota_rules WHERE configuration_id = $1
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

	var selectedIDs []int64
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
		selectedIDs = append(selectedIDs, picked...)
	}

	rand.Shuffle(len(selectedIDs), func(i, j int) { selectedIDs[i], selectedIDs[j] = selectedIDs[j], selectedIDs[i] })

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

		if _, err := tx.Exec(ctx, `
			INSERT INTO booklet_question_snapshots
				(booklet_id, question_id, position_in_booklet, discipline_name, subject_name, grade_year_name, difficulty_name,
				 statement_json, command_json, alternatives_json)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, bookletID, questionID, i+1, disciplineName, subjectName, gradeYearName, difficultyName, stmt, cmd, alternativesJSON); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE booklet_configurations SET is_frozen = true, updated_at = now() WHERE id = $1`, configID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LoadSnapshots devolve as questões já congeladas de um caderno, na ordem
// impressa (seção 26).
func (r *Repository) LoadSnapshots(ctx context.Context, bookletID int64) ([]Snapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, booklet_id, question_id, position_in_booklet, discipline_name, subject_name, grade_year_name, difficulty_name,
		       statement_json, command_json, alternatives_json
		FROM booklet_question_snapshots
		WHERE booklet_id = $1
		ORDER BY position_in_booklet
	`, bookletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Snapshot
	for rows.Next() {
		var s Snapshot
		var altJSON json.RawMessage
		if err := rows.Scan(&s.ID, &s.BookletID, &s.QuestionID, &s.PositionInBooklet, &s.DisciplineName, &s.SubjectName,
			&s.GradeYearName, &s.DifficultyName, &s.StatementJSON, &s.CommandJSON, &altJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(altJSON, &s.Alternatives); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *Repository) CreateDocument(ctx context.Context, bookletID, requestedBy int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO generated_documents (booklet_id, requested_by) VALUES ($1, $2) RETURNING id
	`, bookletID, requestedBy).Scan(&id)
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
		SELECT id, booklet_id, status, file_path, error_message, requested_by, created_at, completed_at
		FROM generated_documents WHERE id = $1
	`, id).Scan(&d.ID, &d.BookletID, &d.Status, &d.FilePath, &d.ErrorMessage, &d.RequestedBy, &d.CreatedAt, &d.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return GeneratedDocument{}, ErrNotFound
	}
	return d, err
}

func (r *Repository) ListDocumentsForBooklet(ctx context.Context, bookletID int64) ([]GeneratedDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, booklet_id, status, file_path, error_message, requested_by, created_at, completed_at
		FROM generated_documents WHERE booklet_id = $1 ORDER BY created_at DESC
	`, bookletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GeneratedDocument
	for rows.Next() {
		var d GeneratedDocument
		if err := rows.Scan(&d.ID, &d.BookletID, &d.Status, &d.FilePath, &d.ErrorMessage, &d.RequestedBy, &d.CreatedAt, &d.CompletedAt); err != nil {
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
