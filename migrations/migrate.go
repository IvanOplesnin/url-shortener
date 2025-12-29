package migrate

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// Вшиваем миграции в бинарь.
//
//go:embed schema/*.sql
var migrationsFS embed.FS

// Up применяет все миграции вверх.
func Up(db *sql.DB) error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrationsFS)

	if err := goose.Up(db, "schema"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// Down откатывает одну миграцию (опционально).
func Down(db *sql.DB) error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(migrationsFS)

	if err := goose.Down(db, "schema"); err != nil {
		return fmt.Errorf("goose down: %w", err)
	}
	return nil
}
