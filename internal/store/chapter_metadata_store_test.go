package store_test

import (
	"slices"
	"testing"

	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func createTestChapter(t *testing.T, s *store.Store, folderID int64, path, hash string) int64 {
	t.Helper()
	chapter, err := s.CreateChapter(folderID, path, hash, 10, "thumb")
	if err != nil {
		t.Fatalf("CreateChapter(%s): %v", path, err)
	}
	return chapter.ID
}

func TestChapterMetadataCreateRead(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "Series1")
	chapterID := createTestChapter(t, s, folderID, "/lib/Series1/ch1.cbz", "hash1")

	meta := &metadata.ChapterMetadata{
		Title:           ptrTo("Chapter 1"),
		Number:          ptrTo("1"),
		SortNumber:      ptrTo(1.0),
		Volume:          ptrTo("Vol. 1"),
		Summary:         ptrTo("First chapter summary"),
		Notes:           ptrTo("Some notes"),
		ReleaseDate:     ptrTo("2024-01-15"),
		Language:        ptrTo("en"),
		ChapterType:     ptrTo(metadata.ChapterTypeRegular),
		AgeRating:       ptrTo("Teen"),
		Web:             ptrTo("https://example.com/ch1"),
		Characters:      ptrTo("Naruto, Sasuke"),
		Teams:           ptrTo("Team 7"),
		Locations:       ptrTo("Konoha"),
		ScanlationGroup: ptrTo("GroupA"),
		StoryArc:        ptrTo("Land of Waves"),
		StoryArcNumber:  ptrTo("1"),
		Authors:         []metadata.ChapterMetadataAuthor{{Name: "Kishimoto", Role: "Writer"}},
		Genres:          []string{"Action", "Adventure"},
		Tags:            []string{"Ninja", "Shonen"},
	}

	if err := cms.UpsertChapterMetadata(chapterID, meta); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}

	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if got == nil {
		t.Fatal("GetChapterMetadata returned nil")
	}

	// Scalar fields
	if got.Title == nil || *got.Title != "Chapter 1" {
		t.Errorf("Title: got %v, want 'Chapter 1'", got.Title)
	}
	if got.Number == nil || *got.Number != "1" {
		t.Errorf("Number: got %v, want '1'", got.Number)
	}
	if got.SortNumber == nil || *got.SortNumber != 1.0 {
		t.Errorf("SortNumber: got %v, want 1.0", got.SortNumber)
	}
	if got.Volume == nil || *got.Volume != "Vol. 1" {
		t.Errorf("Volume: got %v, want 'Vol. 1'", got.Volume)
	}
	if got.Summary == nil || *got.Summary != "First chapter summary" {
		t.Errorf("Summary: got %v, want 'First chapter summary'", got.Summary)
	}
	if got.Notes == nil || *got.Notes != "Some notes" {
		t.Errorf("Notes: got %v, want 'Some notes'", got.Notes)
	}
	if got.ReleaseDate == nil || *got.ReleaseDate != "2024-01-15" {
		t.Errorf("ReleaseDate: got %v, want '2024-01-15'", got.ReleaseDate)
	}
	if got.Language == nil || *got.Language != "en" {
		t.Errorf("Language: got %v, want 'en'", got.Language)
	}
	if got.ChapterType == nil || *got.ChapterType != metadata.ChapterTypeRegular {
		t.Errorf("ChapterType: got %v, want '%s'", got.ChapterType, metadata.ChapterTypeRegular)
	}
	if got.AgeRating == nil || *got.AgeRating != "Teen" {
		t.Errorf("AgeRating: got %v, want 'Teen'", got.AgeRating)
	}
	if got.Web == nil || *got.Web != "https://example.com/ch1" {
		t.Errorf("Web: got %v", got.Web)
	}
	if got.Characters == nil || *got.Characters != "Naruto, Sasuke" {
		t.Errorf("Characters: got %v", got.Characters)
	}
	if got.Teams == nil || *got.Teams != "Team 7" {
		t.Errorf("Teams: got %v", got.Teams)
	}
	if got.Locations == nil || *got.Locations != "Konoha" {
		t.Errorf("Locations: got %v", got.Locations)
	}
	if got.ScanlationGroup == nil || *got.ScanlationGroup != "GroupA" {
		t.Errorf("ScanlationGroup: got %v", got.ScanlationGroup)
	}
	if got.StoryArc == nil || *got.StoryArc != "Land of Waves" {
		t.Errorf("StoryArc: got %v", got.StoryArc)
	}
	if got.StoryArcNumber == nil || *got.StoryArcNumber != "1" {
		t.Errorf("StoryArcNumber: got %v", got.StoryArcNumber)
	}
	if got.ChapterID != chapterID {
		t.Errorf("ChapterID: got %d, want %d", got.ChapterID, chapterID)
	}

	// Child tables
	if len(got.Authors) != 1 || got.Authors[0].Name != "Kishimoto" || got.Authors[0].Role != "Writer" {
		t.Errorf("Authors: got %v", got.Authors)
	}
	if len(got.Genres) != 2 || got.Genres[0] != "Action" || got.Genres[1] != "Adventure" {
		t.Errorf("Genres: got %v", got.Genres)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "Ninja" || got.Tags[1] != "Shonen" {
		t.Errorf("Tags: got %v", got.Tags)
	}
}

func TestChapterMetadataGetNilForMissing(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "MissingSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/Missing/ch1.cbz", "hashmissing")

	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil metadata for chapter without metadata row, got %+v", got)
	}
}

func TestChapterMetadataUpsertNilSkip(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "NilSkipSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/NilSkip/ch1.cbz", "hashnilskip")

	// First upsert: set Title and Summary.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title:   ptrTo("Original Title"),
		Summary: ptrTo("Original Summary"),
	}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert: update Title only, Summary nil → should be preserved.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title: ptrTo("Updated Title"),
	}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}

	if got.Title == nil || *got.Title != "Updated Title" {
		t.Errorf("Title: got %v, want 'Updated Title'", got.Title)
	}
	if got.Summary == nil || *got.Summary != "Original Summary" {
		t.Errorf("Summary should be preserved: got %v", got.Summary)
	}
}

func TestChapterMetadataUpsertNilMeta(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cms := store.NewChapterMetadataStore(db)

	// Upserting nil metadata should be a no-op.
	if err := cms.UpsertChapterMetadata(999, nil); err != nil {
		t.Errorf("UpsertChapterMetadata(nil) should return nil, got %v", err)
	}
}

func TestChapterMetadataLockEnforcement(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "LockSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/Lock/ch1.cbz", "hashlock")

	// Set initial title.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title: ptrTo("Original Title"),
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Lock the title field.
	if err := cms.UpdateChapterMetadataLocks(chapterID, &metadata.ChapterMetadataLocks{
		TitleLock: true,
	}); err != nil {
		t.Fatalf("UpdateChapterMetadataLocks: %v", err)
	}

	// Try to update the locked title.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title: ptrTo("New Title"),
	}); err != nil {
		t.Fatalf("upsert with lock: %v", err)
	}

	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}

	if got.Title == nil || *got.Title != "Original Title" {
		t.Errorf("locked title was overwritten: got %v, want 'Original Title'", got.Title)
	}
	if !got.TitleLock {
		t.Error("TitleLock should be true")
	}
}

func TestChapterMetadataLockMultipleFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "MultiLockSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/MultiLock/ch1.cbz", "hashmultilock")

	// Set initial values.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title:   ptrTo("Title"),
		Summary: ptrTo("Summary"),
		Volume:  ptrTo("Vol 1"),
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Lock title and summary, but NOT volume.
	if err := cms.UpdateChapterMetadataLocks(chapterID, &metadata.ChapterMetadataLocks{
		TitleLock:   true,
		SummaryLock: true,
	}); err != nil {
		t.Fatalf("UpdateChapterMetadataLocks: %v", err)
	}

	// Try to update all three.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title:   ptrTo("New Title"),
		Summary: ptrTo("New Summary"),
		Volume:  ptrTo("Vol 2"),
	}); err != nil {
		t.Fatalf("upsert with partial locks: %v", err)
	}

	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}

	if *got.Title != "Title" {
		t.Errorf("locked Title was overwritten: got %q", *got.Title)
	}
	if *got.Summary != "Summary" {
		t.Errorf("locked Summary was overwritten: got %q", *got.Summary)
	}
	if *got.Volume != "Vol 2" {
		t.Errorf("unlocked Volume should be updated: got %q, want 'Vol 2'", *got.Volume)
	}
}

func TestChapterMetadataChildTables(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "ChildSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/Child/ch1.cbz", "hashchild")

	// Set initial child data.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Authors: []metadata.ChapterMetadataAuthor{{Name: "Author1", Role: "Writer"}},
		Genres:  []string{"Action", "Adventure"},
		Tags:    []string{"Tag1"},
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if len(got.Authors) != 1 || got.Authors[0].Name != "Author1" {
		t.Errorf("Authors: got %v", got.Authors)
	}
	if len(got.Genres) != 2 {
		t.Errorf("Genres: got %v, want [Action Adventure]", got.Genres)
	}

	// Replace genres with empty slice → clears them.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Genres: []string{},
	}); err != nil {
		t.Fatalf("clear genres upsert: %v", err)
	}

	got, err = cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata after clear: %v", err)
	}
	if len(got.Genres) != 0 {
		t.Errorf("Genres should be empty after clear: got %v", got.Genres)
	}
	// Authors should be preserved (nil in second upsert = keep existing).
	if len(got.Authors) != 1 {
		t.Errorf("Authors should be preserved: got %v", got.Authors)
	}
}

func TestChapterMetadataGetByFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "FolderSeries")
	ch1 := createTestChapter(t, s, folderID, "/lib/Folder/ch1.cbz", "hashfolder1")
	ch2 := createTestChapter(t, s, folderID, "/lib/Folder/ch2.cbz", "hashfolder2")
	// ch3 has no metadata — should not appear in results.
	_ = createTestChapter(t, s, folderID, "/lib/Folder/ch3.cbz", "hashfolder3")

	if err := cms.UpsertChapterMetadata(ch1, &metadata.ChapterMetadata{
		Title: ptrTo("Chapter 1"),
	}); err != nil {
		t.Fatalf("upsert ch1: %v", err)
	}
	if err := cms.UpsertChapterMetadata(ch2, &metadata.ChapterMetadata{
		Title: ptrTo("Chapter 2"),
	}); err != nil {
		t.Fatalf("upsert ch2: %v", err)
	}

	result, err := cms.GetChapterMetadataByFolder(folderID)
	if err != nil {
		t.Fatalf("GetChapterMetadataByFolder: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 metadata records, got %d", len(result))
	}
	// Results are ordered by chapter_id.
	if *result[0].Title != "Chapter 1" {
		t.Errorf("result[0].Title: got %q, want 'Chapter 1'", *result[0].Title)
	}
	if *result[1].Title != "Chapter 2" {
		t.Errorf("result[1].Title: got %q, want 'Chapter 2'", *result[1].Title)
	}
}

func TestChapterMetadataGetByFolderEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "EmptyFolderSeries")

	result, err := cms.GetChapterMetadataByFolder(folderID)
	if err != nil {
		t.Fatalf("GetChapterMetadataByFolder: %v", err)
	}
	if result == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestBulkUpdateChapterMetadata(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "BulkSeries")
	ch1 := createTestChapter(t, s, folderID, "/lib/Bulk/ch1.cbz", "hashbulk1")
	ch2 := createTestChapter(t, s, folderID, "/lib/Bulk/ch2.cbz", "hashbulk2")

	updates := map[int64]*metadata.ChapterMetadata{
		ch1: {Title: ptrTo("Bulk Ch1")},
		ch2: {Title: ptrTo("Bulk Ch2"), Genres: []string{"Action"}},
	}

	updated, skipped, err := cms.BulkUpdateChapterMetadata(updates)
	if err != nil {
		t.Fatalf("BulkUpdateChapterMetadata: %v", err)
	}

	if len(updated) != 2 {
		t.Errorf("expected 2 updated, got %d: %v", len(updated), updated)
	}
	if len(skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d: %v", len(skipped), skipped)
	}

	// Verify data was written.
	got1, _ := cms.GetChapterMetadata(ch1)
	if got1.Title == nil || *got1.Title != "Bulk Ch1" {
		t.Errorf("ch1 title: got %v, want 'Bulk Ch1'", got1.Title)
	}
	got2, _ := cms.GetChapterMetadata(ch2)
	if got2.Title == nil || *got2.Title != "Bulk Ch2" {
		t.Errorf("ch2 title: got %v, want 'Bulk Ch2'", got2.Title)
	}
	if len(got2.Genres) != 1 || got2.Genres[0] != "Action" {
		t.Errorf("ch2 genres: got %v", got2.Genres)
	}
}

func TestBulkUpdateChapterMetadataLimitExceeded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cms := store.NewChapterMetadataStore(db)

	// Build a map with 501 entries.
	updates := make(map[int64]*metadata.ChapterMetadata, 501)
	for i := int64(1); i <= 501; i++ {
		updates[i] = &metadata.ChapterMetadata{Title: ptrTo("title")}
	}

	_, _, err := cms.BulkUpdateChapterMetadata(updates)
	if err == nil {
		t.Fatal("expected ErrBulkUpdateLimitExceeded, got nil")
	}
	if err != store.ErrBulkUpdateLimitExceeded {
		t.Errorf("expected ErrBulkUpdateLimitExceeded, got %v", err)
	}
}

func TestBulkUpdateChapterMetadataMixedLocks(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "MixedLockSeries")
	ch1 := createTestChapter(t, s, folderID, "/lib/Mixed/ch1.cbz", "hashmixed1")
	ch2 := createTestChapter(t, s, folderID, "/lib/Mixed/ch2.cbz", "hashmixed2")

	// Set initial titles.
	if err := cms.UpsertChapterMetadata(ch1, &metadata.ChapterMetadata{
		Title: ptrTo("Ch1 Original"),
	}); err != nil {
		t.Fatalf("upsert ch1: %v", err)
	}
	if err := cms.UpsertChapterMetadata(ch2, &metadata.ChapterMetadata{
		Title: ptrTo("Ch2 Original"),
	}); err != nil {
		t.Fatalf("upsert ch2: %v", err)
	}

	// Lock title on ch1 only.
	if err := cms.UpdateChapterMetadataLocks(ch1, &metadata.ChapterMetadataLocks{
		TitleLock: true,
	}); err != nil {
		t.Fatalf("lock ch1: %v", err)
	}

	// Bulk update both with title-only updates.
	// ch1: title locked → all fields locked → skipped
	// ch2: title unlocked → updated
	updates := map[int64]*metadata.ChapterMetadata{
		ch1: {Title: ptrTo("Ch1 New")},
		ch2: {Title: ptrTo("Ch2 New")},
	}

	updated, skipped, err := cms.BulkUpdateChapterMetadata(updates)
	if err != nil {
		t.Fatalf("BulkUpdateChapterMetadata: %v", err)
	}

	if len(updated) != 1 || !slices.Contains(updated, ch2) {
		t.Errorf("updated: got %v, want [%d]", updated, ch2)
	}
	if len(skipped) != 1 || !slices.Contains(skipped, ch1) {
		t.Errorf("skipped: got %v, want [%d]", skipped, ch1)
	}

	// Verify ch1 title is unchanged.
	got1, _ := cms.GetChapterMetadata(ch1)
	if *got1.Title != "Ch1 Original" {
		t.Errorf("ch1 locked title was overwritten: got %q", *got1.Title)
	}

	// Verify ch2 title is updated.
	got2, _ := cms.GetChapterMetadata(ch2)
	if *got2.Title != "Ch2 New" {
		t.Errorf("ch2 title not updated: got %q", *got2.Title)
	}
}

func TestUpdateChapterMetadataLocks(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "LocksUpdateSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/LocksUpdate/ch1.cbz", "hashlocksup")

	// Create metadata row first.
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title: ptrTo("Test"),
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Set locks.
	locks := &metadata.ChapterMetadataLocks{
		TitleLock:     true,
		SummaryLock:   true,
		AgeRatingLock: true,
		StoryArcLock:  true,
	}
	if err := cms.UpdateChapterMetadataLocks(chapterID, locks); err != nil {
		t.Fatalf("UpdateChapterMetadataLocks: %v", err)
	}

	// Verify locks are set.
	got, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if !got.TitleLock {
		t.Error("TitleLock should be true")
	}
	if !got.SummaryLock {
		t.Error("SummaryLock should be true")
	}
	if !got.AgeRatingLock {
		t.Error("AgeRatingLock should be true")
	}
	if !got.StoryArcLock {
		t.Error("StoryArcLock should be true")
	}
	// Unset locks should remain false.
	if got.NumberLock {
		t.Error("NumberLock should be false")
	}
	if got.VolumeLock {
		t.Error("VolumeLock should be false")
	}
}

func TestUpdateChapterMetadataLocksNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	cms := store.NewChapterMetadataStore(db)

	folderID := createTestFolder(t, s, "LocksNFSeries")
	chapterID := createTestChapter(t, s, folderID, "/lib/LocksNF/ch1.cbz", "hashlocksnf")

	// No metadata row exists. UpdateChapterMetadataLocks should return error.
	err := cms.UpdateChapterMetadataLocks(chapterID, &metadata.ChapterMetadataLocks{TitleLock: true})
	if err == nil {
		t.Fatal("expected ErrChapterMetadataNotFound, got nil")
	}
	if err != store.ErrChapterMetadataNotFound {
		t.Errorf("expected ErrChapterMetadataNotFound, got %v", err)
	}
}

func TestUpdateChapterMetadataLocksNilIsNoop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cms := store.NewChapterMetadataStore(db)

	// Nil locks should be a no-op.
	if err := cms.UpdateChapterMetadataLocks(999, nil); err != nil {
		t.Errorf("UpdateChapterMetadataLocks(nil) should return nil, got %v", err)
	}
}
