package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// mockMetadataProvider implements metadata.MetadataProvider for testing.
type mockMetadataProvider struct {
	searchResults []metadata.SeriesSearchResult
	searchErr     error
	lastQuery     string
	lastLimit     int
}

func (m *mockMetadataProvider) Name() string { return "mock" }

func (m *mockMetadataProvider) GetSeriesMetadata(_ context.Context, _ string) (*metadata.SeriesMetadata, error) {
	return nil, metadata.ErrNotSupported
}

func (m *mockMetadataProvider) GetSeriesCover(_ context.Context, _ string) (string, error) {
	return "", metadata.ErrNotSupported
}

func (m *mockMetadataProvider) GetBookMetadata(_ context.Context, _, _ string) (*metadata.BookMetadata, error) {
	return nil, metadata.ErrNotSupported
}

func (m *mockMetadataProvider) SearchSeries(_ context.Context, query string, limit int) ([]metadata.SeriesSearchResult, error) {
	m.lastQuery = query
	m.lastLimit = limit
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchResults, nil
}

func (m *mockMetadataProvider) MatchSeries(_ context.Context, _ string) (*metadata.SeriesSearchResult, error) {
	return nil, metadata.ErrNotSupported
}

func setupMetadataTestData(t *testing.T) (*api.Server, http.Handler, *http.Cookie) {
	t.Helper()
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	cookie := testutil.GetAuthCookie(t, server, "metauser", "pw", "user")
	return server, router, cookie
}

func TestHandleGetMetadata_NoMetadata_Returns404(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/NoMeta", "NoMeta", nil)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp["error"] != "no metadata found for this folder" {
		t.Errorf("error message: got %q", errResp["error"])
	}
}

func TestHandleGetMetadata_InvalidFolderID(t *testing.T) {
	_, router, cookie := setupMetadataTestData(t)

	tests := []struct {
		name string
		url  string
	}{
		{"zero", "/api/folders/0/metadata"},
		{"negative", "/api/folders/-1/metadata"},
		{"non-numeric", "/api/folders/abc/metadata"},
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

func TestHandleGetMetadata_WithMetadata_NoProvider(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/HasMeta", "HasMeta", nil)

	status := metadata.SeriesStatusOngoing
	title := "Test Manga"
	summary := "A test summary"
	score := 8.5
	year := 1999
	month := 9
	day := 21
	bookCount := 72
	lang := "ja"
	thumbURL := "https://example.com/cover.jpg"

	meta := &metadata.SeriesMetadata{
		Status:         &status,
		Title:          &title,
		Summary:        &summary,
		CommunityScore: &score,
		ReleaseYear:    &year,
		ReleaseMonth:   &month,
		ReleaseDay:     &day,
		TotalBookCount: &bookCount,
		Language:       &lang,
		ThumbnailURL:   &thumbURL,
		Genres:         []string{"Action", "Adventure"},
		Tags:           []string{"Shounen"},
		Authors:        []metadata.Author{{Name: "Author One", Role: metadata.AuthorRoleWriter}},
		Links:          []metadata.WebLink{{Label: "AniList", URL: "https://anilist.co/manga/1"}},
		Titles:         []metadata.SeriesTitle{{Title: "テスト", Type: "NATIVE", Language: "ja"}},
	}
	if err := server.Store().UpsertSeriesMetadata(folder.ID, meta); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Status         *string  `json:"status"`
			Title          *string  `json:"title"`
			Summary        *string  `json:"summary"`
			CommunityScore *float64 `json:"community_score"`
			ReleaseYear    *int64   `json:"release_year"`
			ReleaseMonth   *int64   `json:"release_month"`
			ReleaseDay     *int64   `json:"release_day"`
			TotalBookCount *int64   `json:"total_book_count"`
			Language       *string  `json:"language"`
			ThumbnailURL   *string  `json:"thumbnail_url"`
			Genres         []string `json:"genres"`
			Tags           []string `json:"tags"`
			Authors        []struct {
				Name string `json:"name"`
				Role string `json:"role"`
			} `json:"authors"`
			Links []struct {
				Label string `json:"label"`
				URL   string `json:"url"`
			} `json:"links"`
			Titles []struct {
				Title    string `json:"title"`
				Type     string `json:"type"`
				Language string `json:"language"`
			} `json:"titles"`
			Locks struct {
				Status bool `json:"status"`
				Title  bool `json:"title"`
			} `json:"locks"`
		} `json:"metadata"`
		Provider *struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Metadata.Status == nil || *resp.Metadata.Status != "ONGOING" {
		t.Errorf("status: got %v", resp.Metadata.Status)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "Test Manga" {
		t.Errorf("title: got %v", resp.Metadata.Title)
	}
	if resp.Metadata.Summary == nil || *resp.Metadata.Summary != "A test summary" {
		t.Errorf("summary: got %v", resp.Metadata.Summary)
	}
	if resp.Metadata.CommunityScore == nil || *resp.Metadata.CommunityScore != 8.5 {
		t.Errorf("community_score: got %v", resp.Metadata.CommunityScore)
	}
	if resp.Metadata.ReleaseYear == nil || *resp.Metadata.ReleaseYear != 1999 {
		t.Errorf("release_year: got %v", resp.Metadata.ReleaseYear)
	}
	if resp.Metadata.TotalBookCount == nil || *resp.Metadata.TotalBookCount != 72 {
		t.Errorf("total_book_count: got %v", resp.Metadata.TotalBookCount)
	}
	if len(resp.Metadata.Genres) != 2 || resp.Metadata.Genres[0] != "Action" {
		t.Errorf("genres: got %v", resp.Metadata.Genres)
	}
	if len(resp.Metadata.Tags) != 1 || resp.Metadata.Tags[0] != "Shounen" {
		t.Errorf("tags: got %v", resp.Metadata.Tags)
	}
	if len(resp.Metadata.Authors) != 1 || resp.Metadata.Authors[0].Name != "Author One" {
		t.Errorf("authors: got %v", resp.Metadata.Authors)
	}
	if len(resp.Metadata.Links) != 1 || resp.Metadata.Links[0].Label != "AniList" {
		t.Errorf("links: got %v", resp.Metadata.Links)
	}
	if len(resp.Metadata.Titles) != 1 || resp.Metadata.Titles[0].Title != "テスト" {
		t.Errorf("titles: got %v", resp.Metadata.Titles)
	}
	if resp.Provider != nil {
		t.Errorf("provider should be null, got %+v", resp.Provider)
	}
}

func TestHandleGetMetadata_WithProvider(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/Linked", "Linked", nil)

	title := "Linked Manga"
	meta := &metadata.SeriesMetadata{
		Title:  &title,
		Genres: []string{},
		Tags:   []string{},
	}
	if err := server.Store().UpsertSeriesMetadata(folder.ID, meta); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "12345"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Provider *struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Provider == nil {
		t.Fatal("provider should not be null")
	}
	if resp.Provider.Name != "anilist" {
		t.Errorf("provider name: got %q", resp.Provider.Name)
	}
	if resp.Provider.ID != "12345" {
		t.Errorf("provider id: got %q", resp.Provider.ID)
	}
}

func TestHandleGetMetadata_NilFieldsAreNull_EmptySlicesAreArray(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/NilFields", "NilFields", nil)

	// Create metadata with only genres set (empty), everything else nil/default
	meta := &metadata.SeriesMetadata{
		Genres: []string{},
	}
	if err := server.Store().UpsertSeriesMetadata(folder.ID, meta); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	// Parse as raw JSON to check null vs [] distinction
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var metaRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["metadata"], &metaRaw); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	// Nil pointer fields should be JSON null
	if string(metaRaw["status"]) != "null" {
		t.Errorf("status should be null, got %s", metaRaw["status"])
	}
	if string(metaRaw["title"]) != "null" {
		t.Errorf("title should be null, got %s", metaRaw["title"])
	}
	if string(metaRaw["summary"]) != "null" {
		t.Errorf("summary should be null, got %s", metaRaw["summary"])
	}
	if string(metaRaw["community_score"]) != "null" {
		t.Errorf("community_score should be null, got %s", metaRaw["community_score"])
	}

	// Slice fields should be [] (empty array), never null
	if string(metaRaw["genres"]) != "[]" {
		t.Errorf("genres should be [], got %s", metaRaw["genres"])
	}
	if string(metaRaw["tags"]) != "[]" {
		t.Errorf("tags should be [], got %s", metaRaw["tags"])
	}
	if string(metaRaw["authors"]) != "[]" {
		t.Errorf("authors should be [], got %s", metaRaw["authors"])
	}
	if string(metaRaw["links"]) != "[]" {
		t.Errorf("links should be [], got %s", metaRaw["links"])
	}
	if string(metaRaw["titles"]) != "[]" {
		t.Errorf("titles should be [], got %s", metaRaw["titles"])
	}

	// Provider should be null
	if string(raw["provider"]) != "null" {
		t.Errorf("provider should be null, got %s", raw["provider"])
	}
}

func TestHandleSearchMetadata_MissingQuery_Returns400(t *testing.T) {
	_, router, cookie := setupMetadataTestData(t)

	tests := []struct {
		name string
		url  string
	}{
		{"no q param", "/api/metadata/search"},
		{"empty q param", "/api/metadata/search?q="},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.url, nil)
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
			}
			var errResp map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if errResp["error"] != "query parameter 'q' is required" {
				t.Errorf("error message: got %q", errResp["error"])
			}
		})
	}
}

func TestHandleSearchMetadata_ReturnsResults(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	mock := &mockMetadataProvider{
		searchResults: []metadata.SeriesSearchResult{
			{URL: "https://anilist.co/manga/1", ImageURL: "https://img.example/1.jpg", Title: "Naruto", ProviderName: "anilist", ResultID: "1"},
			{URL: "https://anilist.co/manga/2", ImageURL: "https://img.example/2.jpg", Title: "One Piece", ProviderName: "anilist", ResultID: "2"},
			{URL: "https://anilist.co/manga/3", ImageURL: "https://img.example/3.jpg", Title: "Bleach", ProviderName: "anilist", ResultID: "3"},
		},
	}
	server.SetMetadataProvider(mock)

	req, _ := http.NewRequest("GET", "/api/metadata/search?q=naruto&limit=10", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Results []struct {
			URL          string `json:"url"`
			ImageURL     string `json:"image_url"`
			Title        string `json:"title"`
			ProviderName string `json:"provider_name"`
			ResultID     string `json:"result_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}
	if resp.Results[0].Title != "Naruto" {
		t.Errorf("first result title: got %q", resp.Results[0].Title)
	}
	if resp.Results[0].ProviderName != "anilist" {
		t.Errorf("first result provider_name: got %q", resp.Results[0].ProviderName)
	}
	if mock.lastQuery != "naruto" {
		t.Errorf("provider received query: got %q, want %q", mock.lastQuery, "naruto")
	}
	if mock.lastLimit != 10 {
		t.Errorf("provider received limit: got %d, want 10", mock.lastLimit)
	}
}

func TestHandleSearchMetadata_EmptyResults_ReturnsEmptyArray(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	mock := &mockMetadataProvider{
		searchResults: []metadata.SeriesSearchResult{},
	}
	server.SetMetadataProvider(mock)

	req, _ := http.NewRequest("GET", "/api/metadata/search?q=nonexistent", nil)
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
	if string(raw["results"]) != "[]" {
		t.Errorf("results should be [], got %s", raw["results"])
	}
}

func TestHandleSearchMetadata_NilResults_ReturnsEmptyArray(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	mock := &mockMetadataProvider{
		searchResults: nil,
	}
	server.SetMetadataProvider(mock)

	req, _ := http.NewRequest("GET", "/api/metadata/search?q=test", nil)
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
	if string(raw["results"]) != "[]" {
		t.Errorf("results should be [] even when provider returns nil, got %s", raw["results"])
	}
}

func TestHandleSearchMetadata_LimitClamping(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	mock := &mockMetadataProvider{
		searchResults: []metadata.SeriesSearchResult{},
	}
	server.SetMetadataProvider(mock)

	tests := []struct {
		name          string
		url           string
		expectedLimit int
	}{
		{"default limit", "/api/metadata/search?q=test", 10},
		{"explicit limit 5", "/api/metadata/search?q=test&limit=5", 5},
		{"limit 25 (max)", "/api/metadata/search?q=test&limit=25", 25},
		{"limit 50 clamped to 25", "/api/metadata/search?q=test&limit=50", 25},
		{"limit 100 clamped to 25", "/api/metadata/search?q=test&limit=100", 25},
		{"invalid limit uses default", "/api/metadata/search?q=test&limit=abc", 10},
		{"zero limit uses default", "/api/metadata/search?q=test&limit=0", 10},
		{"negative limit uses default", "/api/metadata/search?q=test&limit=-5", 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.url, nil)
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
			}
			if mock.lastLimit != tc.expectedLimit {
				t.Errorf("expected limit %d, provider got %d", tc.expectedLimit, mock.lastLimit)
			}
		})
	}
}

func TestHandleSearchMetadata_ProviderError_Returns502(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	mock := &mockMetadataProvider{
		searchErr: fmt.Errorf("network timeout"),
	}
	server.SetMetadataProvider(mock)

	req, _ := http.NewRequest("GET", "/api/metadata/search?q=test", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSearchMetadata_Unauthorized(t *testing.T) {
	server, _, _ := setupMetadataTestData(t)
	mock := &mockMetadataProvider{
		searchResults: []metadata.SeriesSearchResult{},
	}
	server.SetMetadataProvider(mock)
	router := server.Router()

	req, _ := http.NewRequest("GET", "/api/metadata/search?q=test", nil)
	// no cookie
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rr.Code)
	}
}
