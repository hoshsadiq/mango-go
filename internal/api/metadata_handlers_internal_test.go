package api

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

func TestNewServer_DefaultMetadataProviderWired(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		Library: struct {
			Path string `mapstructure:"path"`
		}{Path: t.TempDir()},
	}
	hub := websocket.NewHub()
	go hub.Run()

	app := &core.App{Version: "test"}
	app.SetConfig(cfg)
	app.SetDB(db)
	app.SetWsHub(hub)

	srv := NewServer(app)
	if srv.metadataProvider == nil {
		t.Fatal("NewServer did not wire a default metadataProvider")
	}
}
