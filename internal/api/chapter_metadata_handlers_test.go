package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func setupBulkTestData(t *testing.T) (*store.ChapterMetadataStore, http.Handler, *http.Cookie, int64, []int64) {
	t.Helper()
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "bulkuser", "pw", "user")

	folder, err := server.Store().CreateFolder("/library/BulkTest", "BulkTest", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	var chapterIDs []int64
	for i := 0; i < 5; i++ {
		ch, err := server.Store().CreateChapter(folder.ID, fmt.Sprintf("/library/BulkTest/ch%d.cbz", i+1), fmt.Sprintf("hash%d", i+1), 10, fmt.Sprintf("thumb%d", i+1))
		if err != nil {
			t.Fatalf("CreateChapter: %v", err)
		}
		chapterIDs = append(chapterIDs, ch.ID)
	}

	cms := store.NewChapterMetadataStore(db)
	return cms, router, cookie, folder.ID, chapterIDs
}

func TestBulkUpdateChapterMetadata_Success(t *testing.T) {
	cms, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	body := fmt.Sprintf(`{"chapter_ids":[%d,%d,%d,%d,%d],"fields":{"language":"en","age_rating":"Teen"}}`,
		chapterIDs[0], chapterIDs[1], chapterIDs[2], chapterIDs[3], chapterIDs[4])

	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Updated []int64 `json:"updated"`
		Skipped []struct {
			ID     int64  `json:"id"`
			Reason string `json:"reason"`
		} `json:"skipped"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("total: got %d, want 5", resp.Total)
	}
	if len(resp.Updated) != 5 {
		t.Errorf("updated count: got %d, want 5", len(resp.Updated))
	}
	if len(resp.Skipped) != 0 {
		t.Errorf("skipped count: got %d, want 0", len(resp.Skipped))
	}

	for _, id := range chapterIDs {
		m, err := cms.GetChapterMetadata(id)
		if err != nil {
			t.Fatalf("GetChapterMetadata(%d): %v", id, err)
		}
		if m == nil {
			t.Fatalf("GetChapterMetadata(%d): nil", id)
		}
		if m.Language == nil || *m.Language != "en" {
			t.Errorf("chapter %d language: got %v, want en", id, m.Language)
		}
		if m.AgeRating == nil || *m.AgeRating != "Teen" {
			t.Errorf("chapter %d age_rating: got %v, want Teen", id, m.AgeRating)
		}
	}
}

func TestBulkUpdateChapterMetadata_AutoLocks(t *testing.T) {
	cms, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	body := fmt.Sprintf(`{"chapter_ids":[%d],"fields":{"language":"ja"}}`, chapterIDs[0])
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	m, err := cms.GetChapterMetadata(chapterIDs[0])
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if !m.LanguageLock {
		t.Error("language_lock should be true after bulk update (auto-lock)")
	}
}

func TestBulkUpdateChapterMetadata_SkipsLockedChapters(t *testing.T) {
	cms, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	lang := "ja"
	if err := cms.UpsertChapterMetadata(chapterIDs[2], &metadata.ChapterMetadata{Language: &lang}); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}
	if err := cms.UpdateChapterMetadataLocks(chapterIDs[2], &metadata.ChapterMetadataLocks{LanguageLock: true}); err != nil {
		t.Fatalf("UpdateChapterMetadataLocks: %v", err)
	}

	body := fmt.Sprintf(`{"chapter_ids":[%d,%d,%d],"fields":{"language":"en"}}`,
		chapterIDs[0], chapterIDs[1], chapterIDs[2])
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Updated []int64 `json:"updated"`
		Skipped []struct {
			ID     int64  `json:"id"`
			Reason string `json:"reason"`
		} `json:"skipped"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("total: got %d, want 3", resp.Total)
	}
	if len(resp.Updated) != 2 {
		t.Errorf("updated count: got %d, want 2", len(resp.Updated))
	}
	if len(resp.Skipped) != 1 {
		t.Errorf("skipped count: got %d, want 1", len(resp.Skipped))
	}
	if len(resp.Skipped) > 0 {
		if resp.Skipped[0].ID != chapterIDs[2] {
			t.Errorf("skipped id: got %d, want %d", resp.Skipped[0].ID, chapterIDs[2])
		}
		if resp.Skipped[0].Reason != "language_lock" {
			t.Errorf("skipped reason: got %q, want language_lock", resp.Skipped[0].Reason)
		}
	}

	lockedMeta, _ := cms.GetChapterMetadata(chapterIDs[2])
	if lockedMeta.Language == nil || *lockedMeta.Language != "ja" {
		t.Errorf("locked chapter language should remain ja, got %v", lockedMeta.Language)
	}
}

func TestBulkUpdateChapterMetadata_RejectOver500(t *testing.T) {
	_, router, cookie, folderID, _ := setupBulkTestData(t)

	ids := make([]string, 501)
	for i := range ids {
		ids[i] = fmt.Sprintf("%d", i+1)
	}
	body := fmt.Sprintf(`{"chapter_ids":[%s],"fields":{"language":"en"}}`, strings.Join(ids, ","))

	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBulkUpdateChapterMetadata_RejectPerChapterFields(t *testing.T) {
	_, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	perChapterFields := []string{"title", "number", "sort_number", "summary", "release_date", "volume"}
	for _, field := range perChapterFields {
		t.Run(field, func(t *testing.T) {
			body := fmt.Sprintf(`{"chapter_ids":[%d],"fields":{"%s":"value"}}`, chapterIDs[0], field)
			req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for field %q, got %d body=%s", field, rr.Code, rr.Body.String())
			}

			var errResp map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !strings.Contains(errResp["error"], "not allowed in bulk updates") {
				t.Errorf("error message should mention 'not allowed in bulk updates', got %q", errResp["error"])
			}
		})
	}
}

func TestBulkUpdateChapterMetadata_EmptyChapterIDs(t *testing.T) {
	_, router, cookie, folderID, _ := setupBulkTestData(t)

	body := `{"chapter_ids":[],"fields":{"language":"en"}}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBulkUpdateChapterMetadata_EmptyFields(t *testing.T) {
	_, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	body := fmt.Sprintf(`{"chapter_ids":[%d],"fields":{}}`, chapterIDs[0])
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBulkUpdateChapterMetadata_TagsAndGenres(t *testing.T) {
	cms, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	body := fmt.Sprintf(`{"chapter_ids":[%d,%d],"fields":{"tags":["action","comedy"],"genres":["shounen"]}}`,
		chapterIDs[0], chapterIDs[1])
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	for _, id := range chapterIDs[:2] {
		m, err := cms.GetChapterMetadata(id)
		if err != nil {
			t.Fatalf("GetChapterMetadata(%d): %v", id, err)
		}
		if len(m.Tags) != 2 {
			t.Errorf("chapter %d tags: got %v, want [action comedy]", id, m.Tags)
		}
		if len(m.Genres) != 1 || m.Genres[0] != "shounen" {
			t.Errorf("chapter %d genres: got %v, want [shounen]", id, m.Genres)
		}
	}
}

func TestBulkUpdateChapterMetadata_InvalidFolderID(t *testing.T) {
	_, router, cookie, _, _ := setupBulkTestData(t)

	tests := []struct {
		name string
		url  string
	}{
		{"zero", "/api/folders/0/chapters/metadata/bulk"},
		{"negative", "/api/folders/-1/chapters/metadata/bulk"},
		{"non-numeric", "/api/folders/abc/chapters/metadata/bulk"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"chapter_ids":[1],"fields":{"language":"en"}}`
			req, _ := http.NewRequest("PATCH", tc.url, strings.NewReader(body))
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestBulkUpdateChapterMetadata_PreservesExistingLocks(t *testing.T) {
	cms, router, cookie, folderID, chapterIDs := setupBulkTestData(t)

	title := "Test Title"
	if err := cms.UpsertChapterMetadata(chapterIDs[0], &metadata.ChapterMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}
	if err := cms.UpdateChapterMetadataLocks(chapterIDs[0], &metadata.ChapterMetadataLocks{TitleLock: true}); err != nil {
		t.Fatalf("UpdateChapterMetadataLocks: %v", err)
	}

	body := fmt.Sprintf(`{"chapter_ids":[%d],"fields":{"language":"en"}}`, chapterIDs[0])
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	m, err := cms.GetChapterMetadata(chapterIDs[0])
	if err != nil {
		t.Fatalf("GetChapterMetadata: %v", err)
	}
	if !m.TitleLock {
		t.Error("title_lock should still be true after bulk update of language")
	}
	if !m.LanguageLock {
		t.Error("language_lock should be true after bulk update (auto-lock)")
	}
}

func TestBulkUpdateChapterMetadata_Unauthorized(t *testing.T) {
	_, router, _, folderID, chapterIDs := setupBulkTestData(t)

	body := fmt.Sprintf(`{"chapter_ids":[%d],"fields":{"language":"en"}}`, chapterIDs[0])
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/chapters/metadata/bulk", folderID), strings.NewReader(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rr.Code)
	}
}

func setupChapterMetadataTestData(t *testing.T) (*store.ChapterMetadataStore, http.Handler, *http.Cookie, int64) {
	t.Helper()
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "chmetauser", "pw", "user")

	folder, err := server.Store().CreateFolder("/library/ChMeta", "ChMeta", nil)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	ch, err := server.Store().CreateChapter(folder.ID, "/library/ChMeta/ch1.cbz", "hash1", 10, "thumb1")
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	cms := store.NewChapterMetadataStore(db)
	return cms, router, cookie, ch.ID
}

func TestGetChapterMetadata_NoMetadata_Returns200WithNulls(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var metaRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["metadata"], &metaRaw); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if string(metaRaw["title"]) != "null" {
		t.Errorf("title should be null, got %s", metaRaw["title"])
	}
	if string(metaRaw["summary"]) != "null" {
		t.Errorf("summary should be null, got %s", metaRaw["summary"])
	}
	if string(metaRaw["authors"]) != "[]" {
		t.Errorf("authors should be [], got %s", metaRaw["authors"])
	}
	if string(metaRaw["genres"]) != "[]" {
		t.Errorf("genres should be [], got %s", metaRaw["genres"])
	}
	if string(metaRaw["tags"]) != "[]" {
		t.Errorf("tags should be [], got %s", metaRaw["tags"])
	}
}

func TestGetChapterMetadata_ChapterNotFound_Returns404(t *testing.T) {
	_, router, cookie, _ := setupChapterMetadataTestData(t)

	req, _ := http.NewRequest("GET", "/api/chapters/999999/metadata", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetChapterMetadata_InvalidID_Returns400(t *testing.T) {
	_, router, cookie, _ := setupChapterMetadataTestData(t)

	tests := []struct {
		name string
		url  string
	}{
		{"zero", "/api/chapters/0/metadata"},
		{"negative", "/api/chapters/-1/metadata"},
		{"non-numeric", "/api/chapters/abc/metadata"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.url, nil)
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestGetChapterMetadata_WithMetadata_ReturnsFields(t *testing.T) {
	cms, router, cookie, chapterID := setupChapterMetadataTestData(t)

	title := "Chapter One"
	number := "1"
	sortNum := 1.5
	lang := "en"
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title:      &title,
		Number:     &number,
		SortNumber: &sortNum,
		Language:   &lang,
		Authors:    []metadata.ChapterMetadataAuthor{{Name: "Writer A", Role: "Writer"}},
		Genres:     []string{"Action"},
		Tags:       []string{"tag1"},
	}); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title      *string  `json:"title"`
			Number     *string  `json:"number"`
			SortNumber *float64 `json:"sort_number"`
			Language   *string  `json:"language"`
			Authors    []struct {
				Name string `json:"name"`
				Role string `json:"role"`
			} `json:"authors"`
			Genres []string `json:"genres"`
			Tags   []string `json:"tags"`
			Locks  struct {
				Title bool `json:"title"`
			} `json:"locks"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Metadata.Title == nil || *resp.Metadata.Title != "Chapter One" {
		t.Errorf("title: got %v", resp.Metadata.Title)
	}
	if resp.Metadata.Number == nil || *resp.Metadata.Number != "1" {
		t.Errorf("number: got %v", resp.Metadata.Number)
	}
	if resp.Metadata.SortNumber == nil || *resp.Metadata.SortNumber != 1.5 {
		t.Errorf("sort_number: got %v", resp.Metadata.SortNumber)
	}
	if len(resp.Metadata.Authors) != 1 || resp.Metadata.Authors[0].Name != "Writer A" {
		t.Errorf("authors: got %v", resp.Metadata.Authors)
	}
	if len(resp.Metadata.Genres) != 1 || resp.Metadata.Genres[0] != "Action" {
		t.Errorf("genres: got %v", resp.Metadata.Genres)
	}
	if len(resp.Metadata.Tags) != 1 || resp.Metadata.Tags[0] != "tag1" {
		t.Errorf("tags: got %v", resp.Metadata.Tags)
	}
}

func TestEditChapterMetadata_UpdatesAndAutoLocks(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	body := `{"title":"New Title","language":"ja"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title    *string `json:"title"`
			Language *string `json:"language"`
			Locks    struct {
				Title    bool `json:"title"`
				Language bool `json:"language"`
				Summary  bool `json:"summary"`
			} `json:"locks"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Metadata.Title == nil || *resp.Metadata.Title != "New Title" {
		t.Errorf("title: got %v, want New Title", resp.Metadata.Title)
	}
	if resp.Metadata.Language == nil || *resp.Metadata.Language != "ja" {
		t.Errorf("language: got %v, want ja", resp.Metadata.Language)
	}
	if !resp.Metadata.Locks.Title {
		t.Error("title lock should be true after edit (auto-lock)")
	}
	if !resp.Metadata.Locks.Language {
		t.Error("language lock should be true after edit (auto-lock)")
	}
	if resp.Metadata.Locks.Summary {
		t.Error("summary lock should be false (not edited)")
	}
}

func TestEditChapterMetadata_LockedField_Returns409(t *testing.T) {
	cms, router, cookie, chapterID := setupChapterMetadataTestData(t)

	title := "Locked Title"
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}
	if err := cms.UpdateChapterMetadataLocks(chapterID, &metadata.ChapterMetadataLocks{TitleLock: true}); err != nil {
		t.Fatalf("UpdateChapterMetadataLocks: %v", err)
	}

	body := `{"title":"Try Override"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Error        string   `json:"error"`
		LockedFields []string `json:"locked_fields"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.LockedFields) != 1 || resp.LockedFields[0] != "title" {
		t.Errorf("locked_fields: got %v, want [title]", resp.LockedFields)
	}
}

func TestEditChapterMetadata_AbsentFieldUntouched(t *testing.T) {
	cms, router, cookie, chapterID := setupChapterMetadataTestData(t)

	title := "Original"
	lang := "en"
	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{
		Title:    &title,
		Language: &lang,
	}); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}

	body := `{"title":"Updated"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title    *string `json:"title"`
			Language *string `json:"language"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "Updated" {
		t.Errorf("title: got %v, want Updated", resp.Metadata.Title)
	}
	if resp.Metadata.Language == nil || *resp.Metadata.Language != "en" {
		t.Errorf("language should be unchanged en, got %v", resp.Metadata.Language)
	}
}

func TestEditChapterMetadata_ChapterNotFound_Returns404(t *testing.T) {
	_, router, cookie, _ := setupChapterMetadataTestData(t)

	body := `{"title":"test"}`
	req, _ := http.NewRequest("PATCH", "/api/chapters/999999/metadata", strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEditChapterMetadata_InvalidChapterType_Returns400(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	body := `{"chapter_type":"INVALID"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEditChapterMetadata_ValidChapterType(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	body := `{"chapter_type":"Special"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			ChapterType *string `json:"chapter_type"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.ChapterType == nil || *resp.Metadata.ChapterType != "Special" {
		t.Errorf("chapter_type: got %v, want Special", resp.Metadata.ChapterType)
	}
}

func TestGetChapterMetadataLocks_NoMetadata_ReturnsAllFalse(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/chapters/%d/metadata/locks", chapterID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expectedFields := []string{
		"title", "number", "sort_number", "volume", "summary", "notes",
		"release_date", "language", "chapter_type", "age_rating", "web",
		"characters", "teams", "locations", "scanlation_group",
		"story_arc", "story_arc_number",
	}
	for _, field := range expectedFields {
		val, ok := resp[field]
		if !ok {
			t.Errorf("response missing lock field: %s", field)
			continue
		}
		if val != false {
			t.Errorf("%s should be false, got %v", field, val)
		}
	}
}

func TestGetChapterMetadataLocks_ChapterNotFound_Returns404(t *testing.T) {
	_, router, cookie, _ := setupChapterMetadataTestData(t)

	req, _ := http.NewRequest("GET", "/api/chapters/999999/metadata/locks", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEditChapterMetadataLocks_TogglesSpecifiedLocks(t *testing.T) {
	cms, router, cookie, chapterID := setupChapterMetadataTestData(t)

	if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{}); err != nil {
		t.Fatalf("UpsertChapterMetadata: %v", err)
	}

	body := `{"title_lock":true,"language_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata/locks", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Title    bool `json:"title"`
		Language bool `json:"language"`
		Summary  bool `json:"summary"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Title {
		t.Error("title should be true")
	}
	if !resp.Language {
		t.Error("language should be true")
	}
	if resp.Summary {
		t.Error("summary should be false (not toggled)")
	}

	body2 := `{"title_lock":false}`
	req2, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata/locks", chapterID), strings.NewReader(body2))
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	var resp2 struct {
		Title    bool `json:"title"`
		Language bool `json:"language"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp2.Title {
		t.Error("title should be false after toggle off")
	}
	if !resp2.Language {
		t.Error("language should still be true")
	}
}

func TestEditChapterMetadataLocks_CreatesMetadataRowIfAbsent(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	body := `{"title_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata/locks", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Title bool `json:"title"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Title {
		t.Error("title should be true")
	}
}

func TestEditChapterMetadataLocks_UnknownField_Returns400(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	body := `{"unknown_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata/locks", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEditChapterMetadataLocks_ChapterNotFound_Returns404(t *testing.T) {
	_, router, cookie, _ := setupChapterMetadataTestData(t)

	body := `{"title_lock":true}`
	req, _ := http.NewRequest("PATCH", "/api/chapters/999999/metadata/locks", strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEditChapterMetadataLocks_ReturnsAll17Fields(t *testing.T) {
	_, router, cookie, chapterID := setupChapterMetadataTestData(t)

	body := `{"title_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/chapters/%d/metadata/locks", chapterID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expectedFields := []string{
		"title", "number", "sort_number", "volume", "summary", "notes",
		"release_date", "language", "chapter_type", "age_rating", "web",
		"characters", "teams", "locations", "scanlation_group",
		"story_arc", "story_arc_number",
	}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing lock field: %s", field)
		}
	}
	if len(resp) != 17 {
		t.Errorf("expected 17 lock fields, got %d", len(resp))
	}
}

func TestGetChapterMetadata_Unauthorized(t *testing.T) {
	_, router, _, chapterID := setupChapterMetadataTestData(t)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/chapters/%d/metadata", chapterID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rr.Code)
	}
}
