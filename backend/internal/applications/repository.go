package applications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("application not found")
	ErrDuplicateName = errors.New("application with this name already exists")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context) ([]Application, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, status, creator_id, created_at, updated_at
		FROM applications
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Status, &a.CreatorID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id int64) (Application, error) {
	var a Application
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, status, creator_id, created_at, updated_at
		FROM applications WHERE id = $1
	`, id).Scan(&a.ID, &a.Name, &a.Description, &a.Status, &a.CreatorID, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Application{}, ErrNotFound
	}
	return a, err
}

func (r *Repository) Create(ctx context.Context, name, description string, creatorID int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO applications (name, description, creator_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, name, description, creatorID).Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrDuplicateName
	}
	return id, err
}

// Update altera nome, descrição e status. Congelamento (seção 27) é por
// caderno, não por aplicação — o status da aplicação é só informativo
// (RASCUNHO/ATIVA/ENCERRADA), não trava a edição dos cadernos dela.
func (r *Repository) Update(ctx context.Context, id int64, name, description, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE applications SET name = $1, description = $2, status = $3, updated_at = now()
		WHERE id = $4
	`, name, description, status, id)
	if isUniqueViolation(err) {
		return ErrDuplicateName
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
