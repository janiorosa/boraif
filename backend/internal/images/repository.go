package images

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, img Image) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO images (discipline_id, filename, path, mime_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, img.DisciplineID, img.Filename, img.Path, img.MimeType, img.SizeBytes, img.UploadedBy).Scan(&id)
	return id, err
}

// List retorna a biblioteca de imagens de uma disciplina (seção 13/36),
// mais recentes primeiro, com busca opcional pelo nome original do arquivo
// e paginação simples (mesma técnica de count(*) OVER() usada em questions).
func (r *Repository) List(ctx context.Context, disciplineID int64, search string, page, pageSize int) ([]Image, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 24
	}

	var searchPattern *string
	if s := strings.TrimSpace(search); s != "" {
		p := "%" + s + "%"
		searchPattern = &p
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, discipline_id, filename, path, mime_type, size_bytes, uploaded_by, created_at,
		       count(*) OVER() AS total
		FROM images
		WHERE discipline_id = $1
		  AND ($2::text IS NULL OR filename ILIKE $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, disciplineID, searchPattern, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []Image
	var total int
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.DisciplineID, &img.Filename, &img.Path, &img.MimeType,
			&img.SizeBytes, &img.UploadedBy, &img.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		result = append(result, img)
	}
	return result, total, rows.Err()
}
