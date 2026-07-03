package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// mockMetadataProvider implements metadata.MetadataProvider for testing.
type mockMetadataProvider struct {
	searchResults []metadata.SeriesSearchResult
	searchErr     error
	lastQuery     string
	lastLimit     int

	seriesMeta    *metadata.SeriesMetadata
	seriesMetaErr error
	coverURL      string
	coverErr      error
}

func (m *mockMetadataProvider) Name() string { return "mock" }

func (m *mockMetadataProvider) GetSeriesMetadata(_ context.Context, _ string) (*metadata.SeriesMetadata, error) {
	if m.seriesMetaErr != nil {
		return nil, m.seriesMetaErr
	}
	return m.seriesMeta, nil
}

func (m *mockMetadataProvider) GetSeriesCover(_ context.Context, _ string) (string, error) {
	if m.coverErr != nil {
		return "", m.coverErr
	}
	return m.coverURL, nil
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

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var metaRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["metadata"], &metaRaw); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

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

func newTestMeta() *metadata.SeriesMetadata {
	status := metadata.SeriesStatusOngoing
	title := "Provider Title"
	summary := "Provider summary"
	score := 7.5
	return &metadata.SeriesMetadata{
		Status:         &status,
		Title:          &title,
		Summary:        &summary,
		CommunityScore: &score,
		Genres:         []string{"Action"},
		Tags:           []string{"Shounen"},
		Authors:        []metadata.Author{{Name: "Author A", Role: metadata.AuthorRoleWriter}},
		Links:          []metadata.WebLink{{Label: "AniList", URL: "https://anilist.co/manga/20"}},
		Titles:         []metadata.SeriesTitle{{Title: "テスト", Type: "NATIVE", Language: "ja"}},
	}
}

func TestHandleLinkMetadata_Success(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/LinkTest", "LinkTest", nil)

	mock := &mockMetadataProvider{
		seriesMeta: newTestMeta(),
		coverURL:   "https://img.example/cover.jpg",
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"20"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title        *string `json:"title"`
			ThumbnailURL *string `json:"thumbnail_url"`
		} `json:"metadata"`
		Provider *struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "Provider Title" {
		t.Errorf("title: got %v", resp.Metadata.Title)
	}
	if resp.Metadata.ThumbnailURL == nil || *resp.Metadata.ThumbnailURL != "https://img.example/cover.jpg" {
		t.Errorf("thumbnail_url: got %v", resp.Metadata.ThumbnailURL)
	}
	if resp.Provider == nil || resp.Provider.Name != "anilist" || resp.Provider.ID != "20" {
		t.Errorf("provider: got %+v", resp.Provider)
	}
}

func TestHandleLinkMetadata_InvalidProviderName_Returns400(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/LinkBadProv", "LinkBadProv", nil)
	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	body := `{"provider_name":"unknown","provider_id":"20"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLinkMetadata_EmptyProviderID_Returns400(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/LinkEmptyID", "LinkEmptyID", nil)
	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	body := `{"provider_name":"anilist","provider_id":""}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLinkMetadata_ProviderFetchError_Returns502(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/LinkFetchErr", "LinkFetchErr", nil)

	mock := &mockMetadataProvider{
		seriesMetaErr: fmt.Errorf("network timeout"),
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"20"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLinkMetadata_CoverFailureIgnoreMode(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/LinkCoverIgnore", "LinkCoverIgnore", nil)

	mock := &mockMetadataProvider{
		seriesMeta: newTestMeta(),
		coverURL:   "",
		coverErr:   nil,
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"20"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (cover ignored), got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			ThumbnailURL *string `json:"thumbnail_url"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.ThumbnailURL != nil {
		t.Errorf("thumbnail_url should be null when cover ignored, got %v", *resp.Metadata.ThumbnailURL)
	}
}

func TestHandleLinkMetadata_CoverFailureFailMode_Returns502(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/LinkCoverFail", "LinkCoverFail", nil)

	mock := &mockMetadataProvider{
		seriesMeta: newTestMeta(),
		coverErr:   fmt.Errorf("cover fetch failed"),
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"20"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502 (cover fail mode), got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRefreshMetadata_Success(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/RefreshTest", "RefreshTest", nil)

	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "20"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}
	oldTitle := "Old Title"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &oldTitle}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	mock := &mockMetadataProvider{
		seriesMeta: newTestMeta(),
		coverURL:   "https://img.example/refreshed.jpg",
	}
	server.SetMetadataProvider(mock)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/refresh", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title        *string `json:"title"`
			ThumbnailURL *string `json:"thumbnail_url"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "Provider Title" {
		t.Errorf("title should be updated to provider value, got %v", resp.Metadata.Title)
	}
	if resp.Metadata.ThumbnailURL == nil || *resp.Metadata.ThumbnailURL != "https://img.example/refreshed.jpg" {
		t.Errorf("thumbnail_url: got %v", resp.Metadata.ThumbnailURL)
	}
}

func TestHandleRefreshMetadata_NoProviderLink_Returns404(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/RefreshNoLink", "RefreshNoLink", nil)
	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/refresh", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp["error"] != "no provider link found for this folder" {
		t.Errorf("error message: got %q", errResp["error"])
	}
}

func TestHandleResetMetadata_Success(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/ResetTest", "ResetTest", nil)

	title := "Reset Me"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &title, Genres: []string{"Action"}}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "20"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/reset", folder.ID), nil)
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
	if resp["status"] != "reset" {
		t.Errorf("status: got %v", resp["status"])
	}
	if resp["provider_link_preserved"] != true {
		t.Errorf("provider_link_preserved: got %v", resp["provider_link_preserved"])
	}
}

func TestHandleResetMetadata_NoMetadata_Returns404(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/ResetNoMeta", "ResetNoMeta", nil)
	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/reset", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp["error"] != "no metadata found for this folder" {
		t.Errorf("error message: got %q", errResp["error"])
	}
}

func TestHandleResetMetadata_PreservesProviderLink(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/ResetKeepLink", "ResetKeepLink", nil)

	title := "Will Be Reset"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "42"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/reset", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	link, err := server.Store().GetProviderLink(folder.ID)
	if err != nil {
		t.Fatalf("GetProviderLink after reset should succeed: %v", err)
	}
	if link.ProviderName != "anilist" || link.ProviderID != "42" {
		t.Errorf("provider link should be preserved, got name=%q id=%q", link.ProviderName, link.ProviderID)
	}
}

func TestHandleUnlinkMetadata_Success(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/UnlinkTest", "UnlinkTest", nil)

	title := "Unlink Me"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "20"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/unlink", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "unlinked" {
		t.Errorf("status: got %q", resp["status"])
	}
}

func TestHandleUnlinkMetadata_NoMetadata_Returns404(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/UnlinkNoMeta", "UnlinkNoMeta", nil)
	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/unlink", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp["error"] != "no metadata found for this folder" {
		t.Errorf("error message: got %q", errResp["error"])
	}
}

func TestHandleUnlinkMetadata_RemovesProviderLink(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/UnlinkRemLink", "UnlinkRemLink", nil)

	title := "Will Be Unlinked"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "99"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/unlink", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	_, err := server.Store().GetProviderLink(folder.ID)
	if err == nil {
		t.Fatal("GetProviderLink after unlink should return error")
	}
	if !errors.Is(err, store.ErrProviderLinkNotFound) {
		t.Errorf("expected ErrProviderLinkNotFound, got %v", err)
	}
}

func TestHandleEditMetadata_PresentFieldUpdates(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/EditUpdate", "EditUpdate", nil)

	status := metadata.SeriesStatusOngoing
	title := "Original Title"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{
		Status: &status,
		Title:  &title,
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	body := `{"status":"COMPLETED","title":"New Title"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Status *string `json:"status"`
			Title  *string `json:"title"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Status == nil || *resp.Metadata.Status != "COMPLETED" {
		t.Errorf("status: got %v, want COMPLETED", resp.Metadata.Status)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "New Title" {
		t.Errorf("title: got %v, want New Title", resp.Metadata.Title)
	}
}

func TestHandleEditMetadata_AbsentFieldUntouched(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/EditAbsent", "EditAbsent", nil)

	status := metadata.SeriesStatusOngoing
	title := "Original Title"
	summary := "Original Summary"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{
		Status:  &status,
		Title:   &title,
		Summary: &summary,
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	body := `{"title":"New Title"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Status  *string `json:"status"`
			Title   *string `json:"title"`
			Summary *string `json:"summary"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "New Title" {
		t.Errorf("title: got %v, want New Title", resp.Metadata.Title)
	}
	if resp.Metadata.Status == nil || *resp.Metadata.Status != "ONGOING" {
		t.Errorf("status should be unchanged ONGOING, got %v", resp.Metadata.Status)
	}
	if resp.Metadata.Summary == nil || *resp.Metadata.Summary != "Original Summary" {
		t.Errorf("summary should be unchanged, got %v", resp.Metadata.Summary)
	}
}

func TestHandleEditMetadata_NullClearsAndLocks(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/EditNull", "EditNull", nil)

	title := "Will Be Cleared"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{
		Title: &title,
	}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	body := `{"title":null}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title *string `json:"title"`
			Locks struct {
				Title bool `json:"title"`
			} `json:"locks"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Title != nil {
		t.Errorf("title should be null after clearing, got %v", *resp.Metadata.Title)
	}
	if !resp.Metadata.Locks.Title {
		t.Errorf("title_lock should be true after null update (auto-lock)")
	}
}

func TestHandleEditMetadata_InvalidEnum_Returns400(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/EditBadEnum", "EditBadEnum", nil)

	tests := []struct {
		name string
		body string
	}{
		{"invalid status", `{"status":"INVALID"}`},
		{"invalid reading_direction", `{"reading_direction":"DIAGONAL"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), strings.NewReader(tc.body))
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleEditMetadata_InvalidNumericRange_Returns400(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/EditBadNum", "EditBadNum", nil)

	tests := []struct {
		name string
		body string
	}{
		{"negative age_rating", `{"age_rating":-1}`},
		{"community_score too high", `{"community_score":10.1}`},
		{"community_score negative", `{"community_score":-0.1}`},
		{"release_year too short", `{"release_year":99}`},
		{"release_year too long", `{"release_year":10000}`},
		{"release_month zero", `{"release_month":0}`},
		{"release_month 13", `{"release_month":13}`},
		{"release_day zero", `{"release_day":0}`},
		{"release_day 32", `{"release_day":32}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), strings.NewReader(tc.body))
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleEditMetadata_CreatesRowIfNoneExists(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/EditCreate", "EditCreate", nil)

	body := `{"title":"Created From Scratch"}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Metadata struct {
			Title *string `json:"title"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Metadata.Title == nil || *resp.Metadata.Title != "Created From Scratch" {
		t.Errorf("title: got %v, want 'Created From Scratch'", resp.Metadata.Title)
	}

	row, err := server.Store().GetSeriesMetadata(folder.ID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata should succeed after edit: %v", err)
	}
	if !row.Title.Valid || row.Title.String != "Created From Scratch" {
		t.Errorf("store title: got %v", row.Title)
	}
}

func TestHandleEditLocks_TogglesOnlySpecified(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/LockToggle", "LockToggle", nil)

	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	body := `{"status_lock":true,"title_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata/locks", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Status  bool `json:"status"`
		Title   bool `json:"title"`
		Summary bool `json:"summary"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Status {
		t.Error("status should be true")
	}
	if !resp.Title {
		t.Error("title should be true")
	}
	if resp.Summary {
		t.Error("summary should be false (not toggled)")
	}

	body2 := `{"status_lock":false}`
	req2, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata/locks", folder.ID), strings.NewReader(body2))
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	var resp2 struct {
		Status bool `json:"status"`
		Title  bool `json:"title"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp2.Status {
		t.Error("status should be false after toggle off")
	}
	if !resp2.Title {
		t.Error("title should still be true (not toggled)")
	}
}

func TestHandleEditLocks_NoMetadata_Returns404(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/LockNoMeta", "LockNoMeta", nil)

	body := `{"status_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata/locks", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp["error"] != "no metadata found for this folder" {
		t.Errorf("error message: got %q", errResp["error"])
	}
}

func TestHandleEditLocks_ReturnsFullLockState(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	server.SetMetadataProvider(&mockMetadataProvider{})
	folder, _ := server.Store().CreateFolder("/library/LockFull", "LockFull", nil)

	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}

	body := `{"genres_lock":true}`
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/folders/%d/metadata/locks", folder.ID), strings.NewReader(body))
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
		"status", "title", "summary", "publisher", "reading_direction",
		"age_rating", "language", "total_book_count", "community_score",
		"release_date", "thumbnail_url", "genres", "tags", "authors",
		"links", "titles",
	}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing lock field: %s", field)
		}
	}

	if genres, ok := resp["genres"].(bool); !ok || !genres {
		t.Errorf("genres should be true, got %v", resp["genres"])
	}
	if status, ok := resp["status"].(bool); !ok || status {
		t.Errorf("status should be false, got %v", resp["status"])
	}
}

func TestHandleLinkMetadata_MergesTagsIntoFolderTags(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/TagMerge", "TagMerge", nil)

	mock := &mockMetadataProvider{
		seriesMeta: &metadata.SeriesMetadata{
			Genres: []string{"Action", "Adventure"},
			Tags:   []string{"Shounen", "Martial Arts"},
		},
		coverURL: "",
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"100"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	f, err := server.Store().GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}

	tagNames := make(map[string]bool)
	for _, tag := range f.Tags {
		tagNames[tag.Name] = true
	}

	// Only Tags should be merged into folder tags, NOT Genres
	expected := []string{"shounen", "martial arts"}
	for _, name := range expected {
		if !tagNames[name] {
			t.Errorf("expected folder tag %q, got tags: %v", name, tagNames)
		}
	}
	// Genres ("action", "adventure") must NOT appear as folder tags
	for _, name := range []string{"action", "adventure"} {
		if tagNames[name] {
			t.Errorf("genre %q should NOT be merged into folder tags, got tags: %v", name, tagNames)
		}
	}
	if len(f.Tags) != 2 {
		t.Errorf("expected 2 folder tags, got %d", len(f.Tags))
	}
}

func TestHandleLinkMetadata_TagMergeIdempotent(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/TagIdempotent", "TagIdempotent", nil)

	mock := &mockMetadataProvider{
		seriesMeta: &metadata.SeriesMetadata{
			Genres: []string{"Action"},
			Tags:   []string{"Shounen", "Martial Arts"},
		},
		coverURL: "",
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"200"}`

	req1, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req1.AddCookie(cookie)
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first link: expected 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}

	req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second link: expected 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	f, err := server.Store().GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if len(f.Tags) != 2 {
		names := make([]string, len(f.Tags))
		for i, tag := range f.Tags {
			names[i] = tag.Name
		}
		t.Errorf("expected 2 folder tags (no duplicates), got %d: %v", len(f.Tags), names)
	}
}

func TestHandleLinkMetadata_TagMergeFailureNonFatal(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/TagPartialFail", "TagPartialFail", nil)

	mock := &mockMetadataProvider{
		seriesMeta: &metadata.SeriesMetadata{
			Genres: []string{"Action"},
			Tags:   []string{"Shounen", "  ", "Martial Arts"},
		},
		coverURL: "",
	}
	server.SetMetadataProvider(mock)

	body := `{"provider_name":"anilist","provider_id":"300"}`
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/link", folder.ID), strings.NewReader(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (non-fatal tag failure), got %d body=%s", rr.Code, rr.Body.String())
	}

	f, err := server.Store().GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	tagNames := make(map[string]bool)
	for _, tag := range f.Tags {
		tagNames[tag.Name] = true
	}
	if !tagNames["shounen"] {
		t.Error("expected folder tag 'shounen' to be present")
	}
	if !tagNames["martial arts"] {
		t.Error("expected folder tag 'martial arts' to be present")
	}
}

func TestHandleUnlinkMetadata_DoesNotRemoveFolderTags(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/UnlinkKeepTags", "UnlinkKeepTags", nil)

	server.Store().AddTagToFolder(folder.ID, "action", "user")
	server.Store().AddTagToFolder(folder.ID, "shounen", "anilist")
	server.Store().AddTagToFolder(folder.ID, "martial arts", "anilist")

	title := "Tagged Series"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "50"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/unlink", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	f, err := server.Store().GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if len(f.Tags) != 1 {
		t.Errorf("expected 1 user tag preserved after unlink (anilist tags removed), got %d", len(f.Tags))
	}
	tagNames := map[string]bool{}
	for _, tag := range f.Tags {
		tagNames[tag.Name] = true
	}
	if !tagNames["action"] {
		t.Error("expected user tag 'action' to be preserved after unlink")
	}
	if tagNames["shounen"] || tagNames["martial arts"] {
		t.Error("expected anilist tags to be removed after unlink")
	}
}

func TestHandleResetMetadata_DoesNotRemoveFolderTags(t *testing.T) {
	server, router, cookie := setupMetadataTestData(t)
	folder, _ := server.Store().CreateFolder("/library/ResetKeepTags", "ResetKeepTags", nil)

	server.Store().AddTagToFolder(folder.ID, "action", "user")
	server.Store().AddTagToFolder(folder.ID, "shounen", "anilist")
	server.Store().AddTagToFolder(folder.ID, "martial arts", "anilist")

	title := "Tagged Series"
	if err := server.Store().UpsertSeriesMetadata(folder.ID, &metadata.SeriesMetadata{Title: &title}); err != nil {
		t.Fatalf("UpsertSeriesMetadata: %v", err)
	}
	if err := server.Store().UpsertProviderLink(folder.ID, "anilist", "60"); err != nil {
		t.Fatalf("UpsertProviderLink: %v", err)
	}

	server.SetMetadataProvider(&mockMetadataProvider{seriesMeta: newTestMeta()})

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/folders/%d/metadata/reset", folder.ID), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	f, err := server.Store().GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if len(f.Tags) != 1 {
		t.Errorf("expected 1 user tag preserved after reset (anilist tags removed), got %d", len(f.Tags))
	}
	tagNames := map[string]bool{}
	for _, tag := range f.Tags {
		tagNames[tag.Name] = true
	}
	if !tagNames["action"] {
		t.Error("expected user tag 'action' to be preserved after reset")
	}
	if tagNames["shounen"] || tagNames["martial arts"] {
		t.Error("expected anilist tags to be removed after reset")
	}
}
