package store_test

import (
	"errors"
	"testing"

	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func ptrTo[T any](v T) *T {
	return &v
}

func createTestFolder(t *testing.T, s *store.Store, name string) int64 {
	t.Helper()
	folder, err := s.CreateFolder("/library/"+name, name, nil)
	if err != nil {
		t.Fatalf("CreateFolder(%s): %v", name, err)
	}
	return folder.ID
}

func TestMetadataCreateRead(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "Naruto")

	status := metadata.SeriesStatusOngoing
	rd := metadata.ReadingDirectionRightToLeft
	meta := &metadata.SeriesMetadata{
		Status:           &status,
		Title:            ptrTo("Naruto"),
		Summary:          ptrTo("A ninja story"),
		Publisher:        ptrTo("Shueisha"),
		ReadingDirection: &rd,
		AgeRating:        ptrTo(13),
		Language:         ptrTo("ja"),
		TotalBookCount:   ptrTo(72),
		CommunityScore:   ptrTo(8.5),
		ReleaseYear:      ptrTo(1999),
		ReleaseMonth:     ptrTo(9),
		ReleaseDay:       ptrTo(21),
		ThumbnailURL:     ptrTo("https://example.com/naruto.jpg"),
		Genres:           []string{"Action", "Adventure"},
		Tags:             []string{"Ninja", "Shonen"},
		Authors:          []metadata.Author{{Name: "Masashi Kishimoto", Role: metadata.AuthorRoleWriter}},
		Links:            []metadata.WebLink{{Label: "AniList", URL: "https://anilist.co/manga/11"}},
		Titles: []metadata.SeriesTitle{
			{Title: "NARUTO", Type: "ROMAJI", Language: "ja"},
			{Title: "Naruto", Type: "LOCALIZED", Language: "en"},
		},
	}

	if err := s.UpsertSeriesMetadata(folderID, meta); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if !row.Status.Valid || row.Status.String != "ONGOING" {
		t.Errorf("Status: got %v, want ONGOING", row.Status)
	}
	if !row.Title.Valid || row.Title.String != "Naruto" {
		t.Errorf("Title: got %v, want Naruto", row.Title)
	}
	if !row.Summary.Valid || row.Summary.String != "A ninja story" {
		t.Errorf("Summary: got %v, want 'A ninja story'", row.Summary)
	}
	if !row.Publisher.Valid || row.Publisher.String != "Shueisha" {
		t.Errorf("Publisher: got %v, want Shueisha", row.Publisher)
	}
	if !row.ReadingDirection.Valid || row.ReadingDirection.String != "RIGHT_TO_LEFT" {
		t.Errorf("ReadingDirection: got %v, want RIGHT_TO_LEFT", row.ReadingDirection)
	}
	if !row.AgeRating.Valid || row.AgeRating.Int64 != 13 {
		t.Errorf("AgeRating: got %v, want 13", row.AgeRating)
	}
	if !row.Language.Valid || row.Language.String != "ja" {
		t.Errorf("Language: got %v, want ja", row.Language)
	}
	if !row.TotalBookCount.Valid || row.TotalBookCount.Int64 != 72 {
		t.Errorf("TotalBookCount: got %v, want 72", row.TotalBookCount)
	}
	if !row.CommunityScore.Valid || row.CommunityScore.Float64 != 8.5 {
		t.Errorf("CommunityScore: got %v, want 8.5", row.CommunityScore)
	}
	if !row.ReleaseYear.Valid || row.ReleaseYear.Int64 != 1999 {
		t.Errorf("ReleaseYear: got %v, want 1999", row.ReleaseYear)
	}
	if !row.ReleaseMonth.Valid || row.ReleaseMonth.Int64 != 9 {
		t.Errorf("ReleaseMonth: got %v, want 9", row.ReleaseMonth)
	}
	if !row.ReleaseDay.Valid || row.ReleaseDay.Int64 != 21 {
		t.Errorf("ReleaseDay: got %v, want 21", row.ReleaseDay)
	}
	if !row.ThumbnailURL.Valid || row.ThumbnailURL.String != "https://example.com/naruto.jpg" {
		t.Errorf("ThumbnailURL: got %v", row.ThumbnailURL)
	}

	// Child tables
	if len(row.Genres) != 2 || row.Genres[0] != "Action" || row.Genres[1] != "Adventure" {
		t.Errorf("Genres: got %v", row.Genres)
	}
	if len(row.Tags) != 2 || row.Tags[0] != "Ninja" || row.Tags[1] != "Shonen" {
		t.Errorf("Tags: got %v", row.Tags)
	}
	if len(row.Authors) != 1 || row.Authors[0].Name != "Masashi Kishimoto" || row.Authors[0].Role != metadata.AuthorRoleWriter {
		t.Errorf("Authors: got %v", row.Authors)
	}
	if len(row.Links) != 1 || row.Links[0].Label != "AniList" {
		t.Errorf("Links: got %v", row.Links)
	}
	if len(row.Titles) != 2 || row.Titles[0].Title != "NARUTO" {
		t.Errorf("Titles: got %v", row.Titles)
	}
}

func TestMetadataNilSkip(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "NilSkipTest")

	// First upsert: set both Title and Summary.
	status := metadata.SeriesStatusOngoing
	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Status:  &status,
		Title:   ptrTo("Naruto"),
		Summary: ptrTo("A ninja story"),
	}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert: update Title only, Summary is nil (should be preserved).
	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title: ptrTo("Naruto Shippuden"),
	}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if !row.Title.Valid || row.Title.String != "Naruto Shippuden" {
		t.Errorf("Title: got %q, want 'Naruto Shippuden'", row.Title.String)
	}
	if !row.Summary.Valid || row.Summary.String != "A ninja story" {
		t.Errorf("Summary should be preserved: got %v", row.Summary)
	}
	if !row.Status.Valid || row.Status.String != "ONGOING" {
		t.Errorf("Status should be preserved: got %v", row.Status)
	}
}

func TestMetadataLockEnforcement(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "LockTest")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title: ptrTo("Original Title"),
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	if err := s.UpdateMetadataLocks(folderID, map[string]bool{"title_lock": true}); err != nil {
		t.Fatalf("UpdateMetadataLocks: %v", err)
	}

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title: ptrTo("New Title"),
	}); err != nil {
		t.Fatalf("upsert with lock: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if row.Title.String != "Original Title" {
		t.Errorf("locked title was overwritten: got %q, want 'Original Title'", row.Title.String)
	}
	if !row.TitleLock {
		t.Error("TitleLock should be true")
	}
}

func TestMetadataLockPreventsCollectionReplace(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "CollLockTest")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Genres: []string{"Action", "Adventure"},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	if err := s.UpdateMetadataLocks(folderID, map[string]bool{"genres_lock": true}); err != nil {
		t.Fatalf("UpdateMetadataLocks: %v", err)
	}

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Genres: []string{"Romance"},
	}); err != nil {
		t.Fatalf("upsert with genre lock: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if len(row.Genres) != 2 || row.Genres[0] != "Action" {
		t.Errorf("locked genres were replaced: got %v, want [Action Adventure]", row.Genres)
	}
}

func TestMetadataEmptySliceClearsCollection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "EmptySliceTest")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Genres: []string{"Action", "Adventure"},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Genres: []string{},
	}); err != nil {
		t.Fatalf("clear upsert: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if len(row.Genres) != 0 {
		t.Errorf("genres should be empty: got %v", row.Genres)
	}
}

func TestMetadataGetNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "NotFoundTest")

	_, err := s.GetSeriesMetadata(folderID)
	if !errors.Is(err, store.ErrMetadataNotFound) {
		t.Errorf("expected ErrMetadataNotFound, got %v", err)
	}
}

func TestProviderLinkNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "PLNotFound")

	_, err := s.GetProviderLink(folderID)
	if !errors.Is(err, store.ErrProviderLinkNotFound) {
		t.Errorf("expected ErrProviderLinkNotFound, got %v", err)
	}
}

func TestProviderLinkCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "PLCrud")

	if err := s.UpsertProviderLink(folderID, "anilist", "12345"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	link, err := s.GetProviderLink(folderID)
	if err != nil {
		t.Fatalf("GetProviderLink: %v", err)
	}
	if link.ProviderName != "anilist" || link.ProviderID != "12345" || link.FolderID != folderID {
		t.Errorf("link mismatch: got %+v", link)
	}

	// Upsert (update provider_id)
	if err := s.UpsertProviderLink(folderID, "anilist", "67890"); err != nil {
		t.Fatalf("UpsertProviderLink update: %v", err)
	}
	link, err = s.GetProviderLink(folderID)
	if err != nil {
		t.Fatalf("GetProviderLink after update: %v", err)
	}
	if link.ProviderID != "67890" {
		t.Errorf("provider_id should be updated: got %q", link.ProviderID)
	}

	if err := s.DeleteProviderLink(folderID); err != nil {
		t.Fatalf("DeleteProviderLink: %v", err)
	}
	_, err = s.GetProviderLink(folderID)
	if !errors.Is(err, store.ErrProviderLinkNotFound) {
		t.Errorf("after delete: expected ErrProviderLinkNotFound, got %v", err)
	}
}

func TestUpdateMetadataFieldAutoLocks(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "AutoLock")

	// UpdateMetadataField should create the metadata row and auto-lock.
	if err := s.UpdateMetadataField(folderID, "title", "My Custom Title"); err != nil {
		t.Fatalf("UpdateMetadataField: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if !row.Title.Valid || row.Title.String != "My Custom Title" {
		t.Errorf("title: got %v, want 'My Custom Title'", row.Title)
	}
	if !row.TitleLock {
		t.Error("title_lock should be true after UpdateMetadataField")
	}

	// Now UpsertSeriesMetadata should not overwrite the locked field.
	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title: ptrTo("Provider Title"),
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	row, err = s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata after upsert: %v", err)
	}
	if row.Title.String != "My Custom Title" {
		t.Errorf("auto-locked title was overwritten: got %q", row.Title.String)
	}
}

func TestUpdateMetadataFieldRejectsUnknown(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "UnknownField")

	err := s.UpdateMetadataField(folderID, "nonexistent_field", "value")
	if err == nil {
		t.Error("expected error for unknown field, got nil")
	}
}

func TestUpdateMetadataLocksRejectsUnknown(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "UnknownLock")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title: ptrTo("Test"),
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	err := s.UpdateMetadataLocks(folderID, map[string]bool{"bogus_lock": true})
	if err == nil {
		t.Error("expected error for unknown lock field, got nil")
	}
}

func TestMetadataReset(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "ResetTest")

	status := metadata.SeriesStatusCompleted
	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Status:       &status,
		Title:        ptrTo("Naruto"),
		Genres:       []string{"Action"},
		Tags:         []string{"Ninja"},
		Authors:      []metadata.Author{{Name: "MK", Role: metadata.AuthorRoleWriter}},
		Links:        []metadata.WebLink{{Label: "AL", URL: "https://al.co"}},
		Titles:       []metadata.SeriesTitle{{Title: "Naruto", Type: "LOCALIZED"}},
		ThumbnailURL: ptrTo("https://example.com/thumb.jpg"),
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := s.UpdateMetadataLocks(folderID, map[string]bool{"title_lock": true}); err != nil {
		t.Fatalf("UpdateMetadataLocks: %v", err)
	}
	if err := s.UpsertProviderLink(folderID, "anilist", "999"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	if err := s.ResetSeriesMetadata(folderID); err != nil {
		t.Fatalf("ResetSeriesMetadata: %v", err)
	}

	// Metadata should be cleared.
	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata after reset: %v", err)
	}
	if row.Status.Valid {
		t.Errorf("Status should be NULL after reset, got %v", row.Status)
	}
	if row.Title.Valid {
		t.Errorf("Title should be NULL after reset, got %v", row.Title)
	}
	if row.ThumbnailURL.Valid {
		t.Errorf("ThumbnailURL should be NULL after reset, got %v", row.ThumbnailURL)
	}
	if row.TitleLock {
		t.Error("TitleLock should be false after reset")
	}
	if len(row.Genres) != 0 {
		t.Errorf("Genres should be empty after reset, got %v", row.Genres)
	}
	if len(row.Tags) != 0 {
		t.Errorf("Tags should be empty after reset, got %v", row.Tags)
	}
	if len(row.Authors) != 0 {
		t.Errorf("Authors should be empty after reset, got %v", row.Authors)
	}
	if len(row.Links) != 0 {
		t.Errorf("Links should be empty after reset, got %v", row.Links)
	}
	if len(row.Titles) != 0 {
		t.Errorf("Titles should be empty after reset, got %v", row.Titles)
	}

	// Provider link should be preserved.
	link, err := s.GetProviderLink(folderID)
	if err != nil {
		t.Fatalf("GetProviderLink after reset: expected link preserved, got %v", err)
	}
	if link.ProviderID != "999" {
		t.Errorf("ProviderID: got %q, want '999'", link.ProviderID)
	}
}

func TestMetadataUnlink(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "UnlinkTest")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title:  ptrTo("One Piece"),
		Genres: []string{"Action"},
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := s.UpsertProviderLink(folderID, "anilist", "30013"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	if err := s.UnlinkSeriesMetadata(folderID); err != nil {
		t.Fatalf("UnlinkSeriesMetadata: %v", err)
	}

	// Metadata should be cleared (same as reset).
	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata after unlink: %v", err)
	}
	if row.Title.Valid {
		t.Errorf("Title should be NULL after unlink, got %v", row.Title)
	}
	if len(row.Genres) != 0 {
		t.Errorf("Genres should be empty after unlink, got %v", row.Genres)
	}

	// Provider link should be GONE.
	_, err = s.GetProviderLink(folderID)
	if !errors.Is(err, store.ErrProviderLinkNotFound) {
		t.Errorf("after unlink: expected ErrProviderLinkNotFound, got %v", err)
	}
}

func TestUpdateMetadataFieldCreatesRow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "FieldCreateRow")

	// No metadata row exists yet. UpdateMetadataField should create one.
	if err := s.UpdateMetadataField(folderID, "summary", "Brand new summary"); err != nil {
		t.Fatalf("UpdateMetadataField: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}
	if !row.Summary.Valid || row.Summary.String != "Brand new summary" {
		t.Errorf("Summary: got %v, want 'Brand new summary'", row.Summary)
	}
	if !row.SummaryLock {
		t.Error("SummaryLock should be true")
	}
}

func TestMetadataReleaseDateLockCoversAllThreeFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "DateLock")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		ReleaseYear:  ptrTo(2000),
		ReleaseMonth: ptrTo(1),
		ReleaseDay:   ptrTo(15),
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Lock release_date (covers year+month+day).
	if err := s.UpdateMetadataLocks(folderID, map[string]bool{"release_date_lock": true}); err != nil {
		t.Fatalf("UpdateMetadataLocks: %v", err)
	}

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		ReleaseYear:  ptrTo(2025),
		ReleaseMonth: ptrTo(12),
		ReleaseDay:   ptrTo(31),
	}); err != nil {
		t.Fatalf("upsert with date lock: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if row.ReleaseYear.Int64 != 2000 {
		t.Errorf("ReleaseYear: got %d, want 2000 (locked)", row.ReleaseYear.Int64)
	}
	if row.ReleaseMonth.Int64 != 1 {
		t.Errorf("ReleaseMonth: got %d, want 1 (locked)", row.ReleaseMonth.Int64)
	}
	if row.ReleaseDay.Int64 != 15 {
		t.Errorf("ReleaseDay: got %d, want 15 (locked)", row.ReleaseDay.Int64)
	}
}

func TestMetadataNilSliceKeepsExistingCollection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folderID := createTestFolder(t, s, "NilSliceKeep")

	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Genres: []string{"Action", "Comedy"},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Upsert with nil Genres (should keep existing).
	if err := s.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{
		Title: ptrTo("Updated Title"),
		// Genres is nil, existing genres should be preserved
	}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	row, err := s.GetSeriesMetadata(folderID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata: %v", err)
	}

	if len(row.Genres) != 2 {
		t.Errorf("nil Genres should preserve existing: got %v, want [Action Comedy]", row.Genres)
	}
}
