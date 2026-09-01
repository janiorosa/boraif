// Package db centraliza a conexão com o PostgreSQL e a execução das migrations.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	"boraif/migrations"

	_ "github.com/jackc/pgx/v5/stdlib" // driver database/sql usado só pelo goose
)

// migrationFilePattern é o único formato que o goose sabe interpretar
// (prefixo numérico + nome). Qualquer outra coisa na pasta migrations/ —
// notadamente arquivos "._nome.sql" que o macOS cria sozinho ao copiar
// arquivos em certos fluxos de transferência — quebra o parser do goose
// com um erro só de leitura difícil de associar à causa real. Já
// aconteceu duas vezes em produção mesmo depois de um .dockerignore
// filtrando isso no build; em vez de depender só de higiene de arquivo
// externa ao código, o filtro abaixo garante que o próprio binário nunca
// enxerga esse tipo de lixo, não importa como ele chegue até aqui.
var migrationFilePattern = regexp.MustCompile(`^[0-9]+.*\.sql$`)

// filteredMigrationsFS embrulha o embed.FS das migrations escondendo
// qualquer entrada que não pareça uma migration de verdade.
type filteredMigrationsFS struct {
	fs.FS
}

func (f filteredMigrationsFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	if err != nil {
		return nil, err
	}
	filtered := entries[:0]
	for _, entry := range entries {
		if entry.IsDir() || migrationFilePattern.MatchString(entry.Name()) {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

// Connect abre um pool de conexões pgx para uso geral pela aplicação.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return pool, nil
}

// Migrate aplica as migrations pendentes. Usa uma conexão database/sql
// separada porque é o que a biblioteca goose espera.
func Migrate(databaseURL string) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("opening migration connection: %w", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(filteredMigrationsFS{migrations.FS})
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}
