package library_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// createCBZWithComicInfo creates a CBZ file containing a ComicInfo.xml and a dummy page.
func createCBZWithComicInfo(t *testing.T, dir, name, comicInfoXML string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create CBZ file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Add a dummy page so the archive is valid.
	w, err := zw.Create("page01.jpg")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	w.Write([]byte("fake-image-data"))

	if comicInfoXML != "" {
		w, err = zw.Create("ComicInfo.xml")
		if err != nil {
			t.Fatalf("Failed to create ComicInfo.xml entry: %v", err)
		}
		w.Write([]byte(comicInfoXML))
	}

	return path
}

func TestParseChapterMetadata_PopulatesMetadataForChaptersWithoutRows(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	cms := store.NewChapterMetadataStore(app.DB())

	// Create a folder and chapter in the DB with a real CBZ on disk.
	seriesDir := filepath.Join(app.Config().Library.Path, "Series A")
	os.Mkdir(seriesDir, 0755)

	comicXML := `<?xml version="1.0" encoding="utf-8"?>
<ComicInfo>
  <Title>The Beginning</Title>
  <Number>1</Number>
  <Volume>1</Volume>
  <Summary>First chapter</Summary>
  <Writer>Author A</Writer>
  <Genre>Action, Adventure</Genre>
</ComicInfo>`

	chPath := createCBZWithComicInfo(t, seriesDir, "Ch.001 - The Beginning.cbz", comicXML)

	folder, err := st.CreateFolder(seriesDir, "Series A", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	ch, err := st.CreateChapter(folder.ID, chPath, "hash_backfill_1", 1, "")
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	// Verify no metadata exists yet.
	got, err := cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata before backfill: %v", err)
	}
	if got != nil {
		t.Fatal("Expected no metadata before backfill")
	}

	// Run the backfill job.
	ctx := &testutil.MockJobContext{App: app}
	library.ParseChapterMetadata(ctx)

	// Verify metadata was created with ComicInfo values (higher priority).
	got, err = cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata after backfill: %v", err)
	}
	if got == nil {
		t.Fatal("Expected metadata after backfill, got nil")
	}
	if got.Title == nil || *got.Title != "The Beginning" {
		t.Errorf("Title: got %v, want 'The Beginning'", got.Title)
	}
	if got.Number == nil || *got.Number != "1" {
		t.Errorf("Number: got %v, want '1'", got.Number)
	}
	if got.Volume == nil || *got.Volume != "1" {
		t.Errorf("Volume: got %v, want '1'", got.Volume)
	}
	if got.Summary == nil || *got.Summary != "First chapter" {
		t.Errorf("Summary: got %v, want 'First chapter'", got.Summary)
	}
	if len(got.Authors) != 1 || got.Authors[0].Name != "Author A" {
		t.Errorf("Authors: got %v, want [{Author A WRITER}]", got.Authors)
	}
	if len(got.Genres) != 2 {
		t.Errorf("Genres: got %v, want [Action Adventure]", got.Genres)
	}
}

func TestParseChapterMetadata_SkipsChaptersWithExistingMetadata(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	cms := store.NewChapterMetadataStore(app.DB())

	seriesDir := filepath.Join(app.Config().Library.Path, "Series B")
	os.Mkdir(seriesDir, 0755)

	comicXML := `<?xml version="1.0" encoding="utf-8"?>
<ComicInfo><Title>Overwrite Me</Title></ComicInfo>`

	chPath := createCBZWithComicInfo(t, seriesDir, "Ch.005.cbz", comicXML)

	folder, err := st.CreateFolder(seriesDir, "Series B", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	ch, err := st.CreateChapter(folder.ID, chPath, "hash_backfill_2", 1, "")
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	// Pre-populate metadata with a custom title.
	originalTitle := "Original Title"
	err = cms.UpsertChapterMetadata(ch.ID, &metadata.ChapterMetadata{
		Title: &originalTitle,
	})
	if err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}

	// Run the backfill job.
	ctx := &testutil.MockJobContext{App: app}
	library.ParseChapterMetadata(ctx)

	// Verify the existing metadata was NOT overwritten.
	got, err := cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if got == nil {
		t.Fatal("Expected metadata to still exist")
	}
	if got.Title == nil || *got.Title != "Original Title" {
		t.Errorf("Title should remain 'Original Title', got %v", got.Title)
	}
}

func TestParseChapterMetadata_Idempotent(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	cms := store.NewChapterMetadataStore(app.DB())

	seriesDir := filepath.Join(app.Config().Library.Path, "Series C")
	os.Mkdir(seriesDir, 0755)

	chPath := createCBZWithComicInfo(t, seriesDir, "Ch.010.cbz", "")

	folder, err := st.CreateFolder(seriesDir, "Series C", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	ch, err := st.CreateChapter(folder.ID, chPath, "hash_backfill_3", 1, "")
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	ctx := &testutil.MockJobContext{App: app}

	// First run — should create metadata from filename.
	library.ParseChapterMetadata(ctx)

	got, err := cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata after first run: %v", err)
	}
	if got == nil {
		t.Fatal("Expected metadata after first run")
	}
	if got.Number == nil || *got.Number != "010" {
		t.Errorf("Number: got %v, want '010'", got.Number)
	}

	// Second run — should process zero chapters (idempotent).
	library.ParseChapterMetadata(ctx)

	got2, err := cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata after second run: %v", err)
	}
	if got2 == nil {
		t.Fatal("Expected metadata after second run")
	}
	// Metadata should be unchanged.
	if got2.Number == nil || *got2.Number != "010" {
		t.Errorf("Number after second run: got %v, want '010'", got2.Number)
	}
}

func TestParseChapterMetadata_HandlesExtractionErrorsGracefully(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	cms := store.NewChapterMetadataStore(app.DB())

	seriesDir := filepath.Join(app.Config().Library.Path, "Series D")
	os.Mkdir(seriesDir, 0755)

	// Create a chapter in the DB pointing to a non-existent file.
	badPath := filepath.Join(seriesDir, "Ch.042 - Missing.cbz")

	folder, err := st.CreateFolder(seriesDir, "Series D", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	ch, err := st.CreateChapter(folder.ID, badPath, "hash_backfill_4", 1, "")
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	// Run the backfill job — should not panic or abort.
	ctx := &testutil.MockJobContext{App: app}
	library.ParseChapterMetadata(ctx)

	// Metadata should still be created (from filename parsing at minimum).
	got, err := cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if got == nil {
		t.Fatal("Expected metadata to be created even with extraction error")
	}
	// Filename parsing should have extracted the number.
	if got.Number == nil || *got.Number != "042" {
		t.Errorf("Number: got %v, want '042'", got.Number)
	}
}

func TestParseChapterMetadata_FilenameOnlyWhenNoComicInfo(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	cms := store.NewChapterMetadataStore(app.DB())

	seriesDir := filepath.Join(app.Config().Library.Path, "Series E")
	os.Mkdir(seriesDir, 0755)

	// CBZ without ComicInfo.xml
	chPath := createCBZWithComicInfo(t, seriesDir, "Vol.02 Ch.015 - The Journey.cbz", "")

	folder, err := st.CreateFolder(seriesDir, "Series E", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	ch, err := st.CreateChapter(folder.ID, chPath, "hash_backfill_5", 1, "")
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	ctx := &testutil.MockJobContext{App: app}
	library.ParseChapterMetadata(ctx)

	got, err := cms.GetChapterMetadata(ch.ID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if got == nil {
		t.Fatal("Expected metadata from filename parsing")
	}
	if got.Number == nil || *got.Number != "015" {
		t.Errorf("Number: got %v, want '015'", got.Number)
	}
	if got.Volume == nil || *got.Volume != "2" {
		t.Errorf("Volume: got %v, want '2'", got.Volume)
	}
	if got.Title == nil || *got.Title != "The Journey" {
		t.Errorf("Title: got %v, want 'The Journey'", got.Title)
	}
}
