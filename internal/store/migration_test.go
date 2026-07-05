package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func findProjectRoot() (string, error) {
	_, b, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(b)
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir, nil
		}
		currentDir = filepath.Dir(currentDir)
	}
	return "", fmt.Errorf("could not find project root containing go.mod")
}

func TestMetadataMigrationUp(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign key support: %v", err)
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	migrationsPath := filepath.Join(projectRoot, "internal", "assets", "migrations")

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", migrationsPath), "sqlite3", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Check if all 7 new tables exist
	tables := []string{
		"series_metadata",
		"series_provider_link",
		"series_metadata_genres",
		"series_metadata_tags",
		"series_metadata_authors",
		"series_metadata_links",
		"series_metadata_titles",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("Table %s not found", table)
		}
	}

	// Check if all 7 indexes exist
	indexes := []string{
		"idx_series_metadata_folder",
		"idx_series_provider_link_folder",
		"idx_series_metadata_genres_mid",
		"idx_series_metadata_tags_mid",
		"idx_series_metadata_authors_mid",
		"idx_series_metadata_links_mid",
		"idx_series_metadata_titles_mid",
	}

	for _, idx := range indexes {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("Index %s not found", idx)
		}
	}

	// Verify existing tables are untouched
	existingTables := []string{"folders", "chapters", "tags", "folder_tags"}
	for _, table := range existingTables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("Existing table %s was affected", table)
		}
	}
}

func TestMetadataMigrationDown(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign key support: %v", err)
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	migrationsPath := filepath.Join(projectRoot, "internal", "assets", "migrations")

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", migrationsPath), "sqlite3", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='series_metadata'").Scan(&count)
	if err != nil || count == 0 {
		t.Fatal("series_metadata table not created")
	}

	// Step down past 000012 (chapter_metadata), 000011 (folder_tags_source), and 000010 (metadata tables)
	if err := m.Steps(-3); err != nil {
		t.Fatalf("Failed to step down migration: %v", err)
	}

	// Verify new tables are gone
	newTables := []string{
		"series_metadata",
		"series_provider_link",
		"series_metadata_genres",
		"series_metadata_tags",
		"series_metadata_authors",
		"series_metadata_links",
		"series_metadata_titles",
	}

	for _, table := range newTables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err == nil && count > 0 {
			t.Errorf("Table %s still exists after down migration", table)
		}
	}

	// Verify existing tables still exist
	existingTables := []string{"folders", "chapters", "tags", "folder_tags"}
	for _, table := range existingTables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("Existing table %s was affected by down migration", table)
		}
	}
}
