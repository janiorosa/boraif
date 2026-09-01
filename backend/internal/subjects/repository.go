package subjects

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("subject not found")
	ErrDuplicateName    = errors.New("subject with this name already exists in the discipline")
	ErrInvalidReference = errors.New("invalid discipline reference")
	ErrInUse            = errors.New("subject is in use by existing questions")
)

// similarityThreshold define a partir de que ponto dois nomes são
// considerados "parecidos" o bastante para alertar o usuário antes de criar
// um assunto novo (seção 14). 0.35 é um valor conservador: pega variações
// como acentuação/plural/typo sem soar como grafias completamente distintas.
const similarityThreshold = 0.35

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List retorna os assuntos, opcionalmente filtrados por disciplina.
func (r *Repository) List(ctx context.Context, disciplineID *int64) ([]Subject, error) {
	var rows pgx.Rows
	var err error
	if disciplineID != nil {
		rows, err = r.pool.Query(ctx, `
			SELECT id, discipline_id, name, created_by
			FROM subjects WHERE discipline_id = $1 ORDER BY name
		`, *disciplineID)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, discipline_id, name, created_by
			FROM subjects ORDER BY discipline_id, name
		`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Subject
	for rows.Next() {
		var s Subject
		if err := rows.Scan(&s.ID, &s.DisciplineID, &s.Name, &s.CreatedBy); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// FindSimilar procura assuntos já existentes na mesma disciplina com nome
// igual ou parecido, usando a extensão pg_trgm. Usado para alertar o
// professor antes de criar uma duplicata acidental (seção 14).
func (r *Repository) FindSimilar(ctx context.Context, disciplineID int64, name string) ([]SimilarSubject, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, similarity(name, $2) AS score
		FROM subjects
		WHERE discipline_id = $1 AND similarity(name, $2) > $3
		ORDER BY score DESC
		LIMIT 5
	`, disciplineID, name, similarityThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SimilarSubject
	for rows.Next() {
		var s SimilarSubject
		if err := rows.Scan(&s.ID, &s.Name, &s.Score); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *Repository) Create(ctx context.Context, s Subject) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO subjects (discipline_id, name, created_by)
		VALUES ($1, $2, $3)
		RETURNING id
	`, s.DisciplineID, s.Name, s.CreatedBy).Scan(&id)
	switch {
	case isUniqueViolation(err):
		return 0, ErrDuplicateName
	case isForeignKeyViolation(err):
		return 0, ErrInvalidReference
	default:
		return id, err
	}
}

func (r *Repository) Update(ctx context.Context, id int64, name string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE subjects SET name = $1 WHERE id = $2`, name, id)
	switch {
	case isUniqueViolation(err):
		return ErrDuplicateName
	case err != nil:
		return err
	case tag.RowsAffected() == 0:
		return ErrNotFound
	default:
		return nil
	}
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM subjects WHERE id = $1`, id)
	switch {
	case isForeignKeyViolation(err):
		return ErrInUse
	case err != nil:
		return err
	case tag.RowsAffected() == 0:
		return ErrNotFound
	default:
		return nil
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
