package disciplines

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("discipline not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// FindByID é usado, por exemplo, pelo upload de imagens (Fase 5) para
// resolver o código da disciplina e montar o caminho de armazenamento
// (seção 13: banco de imagens separado por disciplina).
func (r *Repository) FindByID(ctx context.Context, id int64) (Discipline, error) {
	var d Discipline
	err := r.pool.QueryRow(ctx, `SELECT id, code, name FROM disciplines WHERE id = $1`, id).
		Scan(&d.ID, &d.Code, &d.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Discipline{}, ErrNotFound
	}
	return d, err
}

// List retorna as disciplinas cadastradas. Não há CRUD de disciplinas nesta
// fase: as 13 disciplinas do Ensino Médio já vêm fixas pelo seed (seção 32).
func (r *Repository) List(ctx context.Context) ([]Discipline, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, name FROM disciplines ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Discipline
	for rows.Next() {
		var d Discipline
		if err := rows.Scan(&d.ID, &d.Code, &d.Name); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
