package questions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("question not found")
	ErrInvalidReference    = errors.New("invalid subject/grade year/difficulty/status reference")
	ErrAlternativesInvalid = errors.New("alternatives must be exactly five, with exactly one correct")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type ListFilter struct {
	DisciplineID *int64
	SubjectID    *int64
	GradeYearID  *int64
	DifficultyID *int64
	StatusID     *int64
	AuthorID     *int64
	Search       string
	Page         int
	PageSize     int
	SortBy       string
	SortDir      string
}

type QuestionSummary struct {
	ID             int64
	DisciplineName string
	SubjectName    string
	GradeYearName  string
	DifficultyName string
	StatusCode     string
	StatusName     string
	AuthorName     string
	UpdatedAt      time.Time
	RevisionNumber int
}

// sortColumns é uma lista branca de colunas ordenáveis (seção 38) — nunca
// interpola a entrada do usuário diretamente na query.
var sortColumns = map[string]string{
	"updatedAt": "q.updated_at",
	"createdAt": "q.created_at",
	"subject":   "sub.name",
	"status":    "st.sort_order",
}

// List retorna uma página de questões com os filtros da seção 38, já com os
// nomes resolvidos via JOIN (evita N+1 chamadas do frontend) e o total via
// count(*) OVER() (evita uma segunda query só para paginação).
func (r *Repository) List(ctx context.Context, f ListFilter) ([]QuestionSummary, int, error) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	sortColumn, ok := sortColumns[f.SortBy]
	if !ok {
		sortColumn = sortColumns["updatedAt"]
	}
	sortDir := "DESC"
	if f.SortDir == "asc" {
		sortDir = "ASC"
	}

	var searchPattern *string
	if s := strings.TrimSpace(f.Search); s != "" {
		p := "%" + s + "%"
		searchPattern = &p
	}

	query := fmt.Sprintf(`
		SELECT
			q.id, d.name, sub.name, gy.name, df.name, st.code, st.name, u.name,
			q.updated_at, q.revision_number,
			count(*) OVER() AS total
		FROM questions q
		JOIN disciplines d ON d.id = q.discipline_id
		JOIN subjects sub ON sub.id = q.subject_id
		JOIN grade_years gy ON gy.id = q.grade_year_id
		JOIN difficulties df ON df.id = q.difficulty_id
		JOIN question_statuses st ON st.id = q.status_id
		JOIN users u ON u.id = q.author_id
		WHERE ($1::bigint IS NULL OR q.discipline_id = $1)
		  AND ($2::bigint IS NULL OR q.subject_id = $2)
		  AND ($3::bigint IS NULL OR q.grade_year_id = $3)
		  AND ($4::bigint IS NULL OR q.difficulty_id = $4)
		  AND ($5::bigint IS NULL OR q.status_id = $5)
		  AND ($6::bigint IS NULL OR q.author_id = $6)
		  AND ($7::text IS NULL OR q.statement_json::text ILIKE $7 OR q.command_json::text ILIKE $7)
		ORDER BY %s %s
		LIMIT $8 OFFSET $9
	`, sortColumn, sortDir)

	rows, err := r.pool.Query(ctx, query,
		f.DisciplineID, f.SubjectID, f.GradeYearID, f.DifficultyID, f.StatusID, f.AuthorID,
		searchPattern, pageSize, (page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []QuestionSummary
	var total int
	for rows.Next() {
		var s QuestionSummary
		if err := rows.Scan(&s.ID, &s.DisciplineName, &s.SubjectName, &s.GradeYearName, &s.DifficultyName,
			&s.StatusCode, &s.StatusName, &s.AuthorName, &s.UpdatedAt, &s.RevisionNumber, &total); err != nil {
			return nil, 0, err
		}
		result = append(result, s)
	}
	return result, total, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id int64) (Question, []Alternative, error) {
	var q Question
	err := r.pool.QueryRow(ctx, `
		SELECT id, discipline_id, subject_id, grade_year_id, difficulty_id, status_id, author_id,
		       statement_json, command_json, revision_number, created_at, updated_at
		FROM questions WHERE id = $1
	`, id).Scan(&q.ID, &q.DisciplineID, &q.SubjectID, &q.GradeYearID, &q.DifficultyID, &q.StatusID, &q.AuthorID,
		&q.StatementJSON, &q.CommandJSON, &q.RevisionNumber, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Question{}, nil, ErrNotFound
	}
	if err != nil {
		return Question{}, nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT position, content_json, is_correct FROM question_alternatives
		WHERE question_id = $1 ORDER BY position
	`, id)
	if err != nil {
		return Question{}, nil, err
	}
	defer rows.Close()

	var alts []Alternative
	for rows.Next() {
		var a Alternative
		if err := rows.Scan(&a.Position, &a.ContentJSON, &a.IsCorrect); err != nil {
			return Question{}, nil, err
		}
		alts = append(alts, a)
	}
	return q, alts, rows.Err()
}

// Create grava a questão e suas cinco alternativas em uma única transação.
// A trigger de constraint (seção 7) valida no COMMIT que existem exatamente
// 5 alternativas com exatamente 1 correta — aqui isso já deve ter sido
// pré-validado pelo handler; a checagem do banco é uma segunda camada de
// segurança, não a primeira.
func (r *Repository) Create(ctx context.Context, q Question, alts []Alternative) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO questions
			(discipline_id, subject_id, grade_year_id, difficulty_id, status_id, author_id, statement_json, command_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`, q.DisciplineID, q.SubjectID, q.GradeYearID, q.DifficultyID, q.StatusID, q.AuthorID, q.StatementJSON, q.CommandJSON,
	).Scan(&id)
	if isForeignKeyViolation(err) {
		return 0, ErrInvalidReference
	}
	if err != nil {
		return 0, err
	}

	for _, a := range alts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO question_alternatives (question_id, position, content_json, is_correct)
			VALUES ($1,$2,$3,$4)
		`, id, a.Position, a.ContentJSON, a.IsCorrect); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		if isRaiseException(err) {
			return 0, ErrAlternativesInvalid
		}
		return 0, err
	}
	return id, nil
}

// Update altera metadados editáveis (assunto, ano, dificuldade, status),
// conteúdo (enunciado/comando) e as cinco alternativas. A disciplina não é
// editável após a criação. revision_number só é incrementado quando o
// conteúdo de fato mudou (seção 5.4) — não a cada chamada de autosave que
// reenvia o mesmo conteúdo.
func (r *Repository) Update(ctx context.Context, q Question, alts []Alternative) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var currentStatement, currentCommand json.RawMessage
	err = tx.QueryRow(ctx, `
		SELECT statement_json, command_json FROM questions WHERE id = $1 FOR UPDATE
	`, q.ID).Scan(&currentStatement, &currentCommand)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT position, content_json, is_correct FROM question_alternatives WHERE question_id = $1
	`, q.ID)
	if err != nil {
		return err
	}
	currentAlts := make(map[int16]Alternative, 5)
	for rows.Next() {
		var a Alternative
		if err := rows.Scan(&a.Position, &a.ContentJSON, &a.IsCorrect); err != nil {
			rows.Close()
			return err
		}
		currentAlts[a.Position] = a
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	contentChanged := !bytes.Equal(currentStatement, q.StatementJSON) || !bytes.Equal(currentCommand, q.CommandJSON)
	for _, a := range alts {
		prev, ok := currentAlts[a.Position]
		if !ok || !bytes.Equal(prev.ContentJSON, a.ContentJSON) || prev.IsCorrect != a.IsCorrect {
			contentChanged = true
		}
	}

	revisionExpr := "revision_number"
	if contentChanged {
		revisionExpr = "revision_number + 1"
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		UPDATE questions
		SET subject_id = $1, grade_year_id = $2, difficulty_id = $3, status_id = $4,
		    statement_json = $5, command_json = $6, revision_number = %s, updated_at = now()
		WHERE id = $7
	`, revisionExpr), q.SubjectID, q.GradeYearID, q.DifficultyID, q.StatusID, q.StatementJSON, q.CommandJSON, q.ID)
	if isForeignKeyViolation(err) {
		return ErrInvalidReference
	}
	if err != nil {
		return err
	}

	for _, a := range alts {
		if _, err := tx.Exec(ctx, `
			UPDATE question_alternatives SET content_json = $1, is_correct = $2
			WHERE question_id = $3 AND position = $4
		`, a.ContentJSON, a.IsCorrect, q.ID, a.Position); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		if isRaiseException(err) {
			return ErrAlternativesInvalid
		}
		return err
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM questions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// isRaiseException identifica o erro levantado pela trigger de constraint
// da seção 7 (RAISE EXCEPTION sem SQLSTATE explícito usa "P0001").
func isRaiseException(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "P0001"
}
