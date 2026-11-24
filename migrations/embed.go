// Package migrations встраивает SQL-миграции goose в бинарник.
package migrations

import "embed"

// FS содержит файлы миграций.
//
//go:embed *.sql
var FS embed.FS
