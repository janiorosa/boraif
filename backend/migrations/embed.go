// Package migrations embute os arquivos SQL de migration no binário, para que
// o backend não dependa de arquivos soltos no filesystem em produção.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
