package metadata

import (
	"encoding/json"
	"testing"
)

func ptrTo[T any](v T) *T {
	return &v
}

// TestSeriesMetadataJSONNil verifies nil pointer fields serialize as JSON null
func TestSeriesMetadataJSONNil(t *testing.T) {
	meta := &SeriesMetadata{}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["status"] != nil {
		t.Errorf("status should be null, got %v", result["status"])
	}
	if result["title"] != nil {
		t.Errorf("title should be null, got %v", result["title"])
	}
	if result["summary"] != nil {
		t.Errorf("summary should be null, got %v", result["summary"])
	}
	if result["age_rating"] != nil {
		t.Errorf("age_rating should be null, got %v", result["age_rating"])
	}
	if result["community_score"] != nil {
		t.Errorf("community_score should be null, got %v", result["community_score"])
	}
	if result["release_year"] != nil {
		t.Errorf("release_year should be null, got %v", result["release_year"])
	}
	if result["release_month"] != nil {
		t.Errorf("release_month should be null, got %v", result["release_month"])
	}
	if result["release_day"] != nil {
		t.Errorf("release_day should be null, got %v", result["release_day"])
	}
	if result["thumbnail_url"] != nil {
		t.Errorf("thumbnail_url should be null, got %v", result["thumbnail_url"])
	}

	if result["genres"] != nil {
		t.Errorf("genres should be null, got %v", result["genres"])
	}
	if result["tags"] != nil {
		t.Errorf("tags should be null, got %v", result["tags"])
	}
	if result["authors"] != nil {
		t.Errorf("authors should be null, got %v", result["authors"])
	}
	if result["links"] != nil {
		t.Errorf("links should be null, got %v", result["links"])
	}
	if result["titles"] != nil {
		t.Errorf("titles should be null, got %v", result["titles"])
	}
}

// TestSeriesMetadataJSONZeroValue verifies zero-value pointers serialize as their value, not null
func TestSeriesMetadataJSONZeroValue(t *testing.T) {
	meta := &SeriesMetadata{
		Title:          ptrTo(""),
		AgeRating:      ptrTo(0),
		CommunityScore: ptrTo(0.0),
		ReleaseYear:    ptrTo(0),
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["title"] == nil {
		t.Errorf("title should be empty string, not null")
	}
	if result["title"] != "" {
		t.Errorf("title should be empty string, got %v", result["title"])
	}

	if result["age_rating"] == nil {
		t.Errorf("age_rating should be 0, not null")
	}
	if result["age_rating"] != float64(0) {
		t.Errorf("age_rating should be 0, got %v", result["age_rating"])
	}

	if result["community_score"] == nil {
		t.Errorf("community_score should be 0.0, not null")
	}
	if result["community_score"] != float64(0) {
		t.Errorf("community_score should be 0.0, got %v", result["community_score"])
	}

	if result["release_year"] == nil {
		t.Errorf("release_year should be 0, not null")
	}
	if result["release_year"] != float64(0) {
		t.Errorf("release_year should be 0, got %v", result["release_year"])
	}
}

// TestSliceSerialization verifies nil slices serialize as null, empty slices as []
func TestSliceSerialization(t *testing.T) {
	meta := &SeriesMetadata{
		Genres: nil,
		Tags:   []string{},
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["genres"] != nil {
		t.Errorf("genres (nil slice) should be null, got %v", result["genres"])
	}

	if result["tags"] == nil {
		t.Errorf("tags (empty slice) should be [], not null")
	}
	tags, ok := result["tags"].([]interface{})
	if !ok {
		t.Errorf("tags should be array, got %T", result["tags"])
	}
	if len(tags) != 0 {
		t.Errorf("tags should be empty array, got %v", tags)
	}
}

// TestAuthorRoleConstants verifies AuthorRole constants are correct strings
func TestAuthorRoleConstants(t *testing.T) {
	tests := []struct {
		role     AuthorRole
		expected string
	}{
		{AuthorRoleWriter, "WRITER"},
		{AuthorRolePenciller, "PENCILLER"},
		{AuthorRoleInker, "INKER"},
		{AuthorRoleColorist, "COLORIST"},
		{AuthorRoleLetterer, "LETTERER"},
		{AuthorRoleCoverArtist, "COVER_ARTIST"},
		{AuthorRoleEditor, "EDITOR"},
		{AuthorRoleTranslator, "TRANSLATOR"},
		{AuthorRoleOther, "OTHER"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("AuthorRole %v should be %q, got %q", tt.role, tt.expected, string(tt.role))
		}
	}
}

// TestSeriesStatusConstants verifies SeriesStatus constants are correct strings
func TestSeriesStatusConstants(t *testing.T) {
	tests := []struct {
		status   SeriesStatus
		expected string
	}{
		{SeriesStatusOngoing, "ONGOING"},
		{SeriesStatusCompleted, "COMPLETED"},
		{SeriesStatusAbandoned, "ABANDONED"},
		{SeriesStatusHiatus, "HIATUS"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("SeriesStatus %v should be %q, got %q", tt.status, tt.expected, string(tt.status))
		}
	}
}

// TestReadingDirectionConstants verifies ReadingDirection constants are correct strings
func TestReadingDirectionConstants(t *testing.T) {
	tests := []struct {
		direction ReadingDirection
		expected  string
	}{
		{ReadingDirectionLeftToRight, "LEFT_TO_RIGHT"},
		{ReadingDirectionRightToLeft, "RIGHT_TO_LEFT"},
		{ReadingDirectionVertical, "VERTICAL"},
		{ReadingDirectionWebtoon, "WEBTOON"},
	}

	for _, tt := range tests {
		if string(tt.direction) != tt.expected {
			t.Errorf("ReadingDirection %v should be %q, got %q", tt.direction, tt.expected, string(tt.direction))
		}
	}
}

// TestSeriesMetadataRoundTrip verifies full marshal/unmarshal round-trip
func TestSeriesMetadataRoundTrip(t *testing.T) {
	original := &SeriesMetadata{
		Status:           ptrTo(SeriesStatusOngoing),
		Title:            ptrTo("Naruto"),
		Summary:          ptrTo("A ninja story"),
		Publisher:        ptrTo("Shonen Jump"),
		ReadingDirection: ptrTo(ReadingDirectionLeftToRight),
		AgeRating:        ptrTo(13),
		Language:         ptrTo("en"),
		TotalBookCount:   ptrTo(72),
		CommunityScore:   ptrTo(8.5),
		ReleaseYear:      ptrTo(1999),
		ReleaseMonth:     ptrTo(9),
		ReleaseDay:       ptrTo(21),
		ThumbnailURL:     ptrTo("https://example.com/cover.jpg"),
		Genres:           []string{"Action", "Adventure"},
		Tags:             []string{"Ninja", "Shonen"},
		Authors: []Author{
			{Name: "Masashi Kishimoto", Role: AuthorRoleWriter},
		},
		Links: []WebLink{
			{Label: "AniList", URL: "https://anilist.co/manga/30015/"},
		},
		Titles: []SeriesTitle{
			{Title: "Naruto", Type: "ROMAJI", Language: "ja"},
			{Title: "Naruto", Type: "LOCALIZED", Language: "en"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored SeriesMetadata
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Title == nil || *restored.Title != "Naruto" {
		t.Errorf("Title mismatch: %v", restored.Title)
	}
	if restored.AgeRating == nil || *restored.AgeRating != 13 {
		t.Errorf("AgeRating mismatch: %v", restored.AgeRating)
	}
	if len(restored.Genres) != 2 {
		t.Errorf("Genres count mismatch: %d", len(restored.Genres))
	}
	if len(restored.Authors) != 1 {
		t.Errorf("Authors count mismatch: %d", len(restored.Authors))
	}
}

// TestBookMetadataJSONNil verifies BookMetadata nil fields serialize as null
func TestBookMetadataJSONNil(t *testing.T) {
	meta := &BookMetadata{}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["title"] != nil {
		t.Errorf("title should be null, got %v", result["title"])
	}
	if result["summary"] != nil {
		t.Errorf("summary should be null, got %v", result["summary"])
	}
	if result["number"] != nil {
		t.Errorf("number should be null, got %v", result["number"])
	}
	if result["isbn"] != nil {
		t.Errorf("isbn should be null, got %v", result["isbn"])
	}
}

// TestSeriesSearchResultFields verifies SeriesSearchResult has all required non-pointer fields
func TestSeriesSearchResultFields(t *testing.T) {
	result := SeriesSearchResult{
		URL:          "https://anilist.co/manga/30015/",
		ImageURL:     "https://example.com/cover.jpg",
		Title:        "Naruto",
		ProviderName: "AniList",
		ResultID:     "30015",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored SeriesSearchResult
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.URL != result.URL {
		t.Errorf("URL mismatch: %q != %q", restored.URL, result.URL)
	}
	if restored.ImageURL != result.ImageURL {
		t.Errorf("ImageURL mismatch: %q != %q", restored.ImageURL, result.ImageURL)
	}
	if restored.Title != result.Title {
		t.Errorf("Title mismatch: %q != %q", restored.Title, result.Title)
	}
	if restored.ProviderName != result.ProviderName {
		t.Errorf("ProviderName mismatch: %q != %q", restored.ProviderName, result.ProviderName)
	}
	if restored.ResultID != result.ResultID {
		t.Errorf("ResultID mismatch: %q != %q", restored.ResultID, result.ResultID)
	}
}
