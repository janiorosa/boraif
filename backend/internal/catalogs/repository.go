package catalogs

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStatusNotFound = errors.New("question status not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListGradeYears(ctx context.Context) ([]GradeYear, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, name FROM grade_years ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GradeYear
	for rows.Next() {
		var g GradeYear
		if err := rows.Scan(&g.ID, &g.Code, &g.Name); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (r *Repository) ListDifficulties(ctx context.Context) ([]Difficulty, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, name FROM difficulties ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Difficulty
	for rows.Next() {
		var d Difficulty
		if err := rows.Scan(&d.ID, &d.Code, &d.Name); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *Repository) ListQuestionStatuses(ctx context.Context) ([]QuestionStatus, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, eligible_for_exam FROM question_statuses ORDER BY sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []QuestionStatus
	for rows.Next() {
		var s QuestionStatus
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.EligibleForExam); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// StatusIDByCode resolve o id de um status a partir do código estável (ex.:
// "RASCUNHO"), usado para definir o status inicial de uma questão nova
// (seção 37) sem espalhar o id numérico pelo código do domínio de questões.
func (r *Repository) StatusIDByCode(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `SELECT id FROM question_statuses WHERE code = $1`, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrStatusNotFound
	}
	return id, err
}
