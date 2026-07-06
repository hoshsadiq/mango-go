package anilist

import (
	"context"
	"errors"
	"testing"

	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/metadata"
)

// mockAniListClient implements aniListClient for testing.
type mockAniListClient struct {
	getMediaFullFn    func(ctx context.Context, id int) (*anilist.MediaFull, error)
	searchMediaFullFn func(ctx context.Context, query string, limit int) ([]anilist.MediaFull, error)

	searchCalls []struct {
		query string
		limit int
	}
}

func (m *mockAniListClient) GetMediaFull(ctx context.Context, id int) (*anilist.MediaFull, error) {
	if m.getMediaFullFn != nil {
		return m.getMediaFullFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAniListClient) SearchMediaFull(ctx context.Context, query string, limit int) ([]anilist.MediaFull, error) {
	m.searchCalls = append(m.searchCalls, struct {
		query string
		limit int
	}{query, limit})
	if m.searchMediaFullFn != nil {
		return m.searchMediaFullFn(ctx, query, limit)
	}
	return nil, errors.New("not implemented")
}

func sampleMedia() *anilist.MediaFull {
	m := &anilist.MediaFull{
		ID:              20,
		Status:          "FINISHED",
		Description:     "A ninja story",
		Genres:          []string{"Action"},
		IsAdult:         false,
		CountryOfOrigin: "JP",
		SiteUrl:         "https://anilist.co/manga/20",
		Volumes:         ptrInt(72),
		AverageScore:    ptrInt(84),
	}
	m.Title.English = "Naruto"
	m.Title.Romaji = "NARUTO"
	m.CoverImage.Large = "https://img.anilist.co/naruto.jpg"
	return m
}

// --- Name ---

func TestProviderName(t *testing.T) {
	p := NewAniListProvider(&mockAniListClient{}, false, "ignore")
	if p.Name() != "anilist" {
		t.Errorf("Name() = %q, want anilist", p.Name())
	}
}

// --- GetSeriesMetadata ---

func TestGetSeriesMetadata(t *testing.T) {
	t.Run("successful fetch and mapping", func(t *testing.T) {
		media := sampleMedia()
		client := &mockAniListClient{
			getMediaFullFn: func(_ context.Context, id int) (*anilist.MediaFull, error) {
				if id != 20 {
					t.Errorf("expected id 20, got %d", id)
				}
				return media, nil
			},
		}
		p := NewAniListProvider(client, true, "ignore")
		result, err := p.GetSeriesMetadata(context.Background(), "20")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("got nil result")
		}
		if result.Title == nil || *result.Title != "Naruto" {
			t.Errorf("title: got %v, want Naruto", result.Title)
		}
		if result.Status == nil || *result.Status != metadata.SeriesStatusCompleted {
			t.Errorf("status: got %v, want COMPLETED", result.Status)
		}
	})

	t.Run("invalid series ID returns error", func(t *testing.T) {
		p := NewAniListProvider(&mockAniListClient{}, false, "ignore")
		_, err := p.GetSeriesMetadata(context.Background(), "not-a-number")
		if err == nil {
			t.Fatal("expected error for invalid ID")
		}
	})

	t.Run("client error propagated", func(t *testing.T) {
		client := &mockAniListClient{
			getMediaFullFn: func(_ context.Context, _ int) (*anilist.MediaFull, error) {
				return nil, errors.New("api error")
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		_, err := p.GetSeriesMetadata(context.Background(), "20")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- GetSeriesCover ---

func TestGetSeriesCover(t *testing.T) {
	t.Run("successful cover fetch", func(t *testing.T) {
		media := sampleMedia()
		client := &mockAniListClient{
			getMediaFullFn: func(_ context.Context, _ int) (*anilist.MediaFull, error) {
				return media, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		cover, err := p.GetSeriesCover(context.Background(), "20")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cover != "https://img.anilist.co/naruto.jpg" {
			t.Errorf("got %q", cover)
		}
	})

	t.Run("cover failure mode ignore returns empty no error", func(t *testing.T) {
		client := &mockAniListClient{
			getMediaFullFn: func(_ context.Context, _ int) (*anilist.MediaFull, error) {
				return nil, errors.New("api error")
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		cover, err := p.GetSeriesCover(context.Background(), "20")
		if err != nil {
			t.Fatalf("expected no error in ignore mode, got: %v", err)
		}
		if cover != "" {
			t.Errorf("expected empty cover, got %q", cover)
		}
	})

	t.Run("cover failure mode fail returns error", func(t *testing.T) {
		client := &mockAniListClient{
			getMediaFullFn: func(_ context.Context, _ int) (*anilist.MediaFull, error) {
				return nil, errors.New("api error")
			},
		}
		p := NewAniListProvider(client, false, "fail")
		cover, err := p.GetSeriesCover(context.Background(), "20")
		if err == nil {
			t.Fatal("expected error in fail mode")
		}
		if cover != "" {
			t.Errorf("expected empty cover, got %q", cover)
		}
	})

	t.Run("invalid ID in ignore mode returns empty no error", func(t *testing.T) {
		p := NewAniListProvider(&mockAniListClient{}, false, "ignore")
		cover, err := p.GetSeriesCover(context.Background(), "bad-id")
		if err != nil {
			t.Fatalf("expected no error in ignore mode, got: %v", err)
		}
		if cover != "" {
			t.Errorf("expected empty cover, got %q", cover)
		}
	})

	t.Run("invalid ID in fail mode returns error", func(t *testing.T) {
		p := NewAniListProvider(&mockAniListClient{}, false, "fail")
		_, err := p.GetSeriesCover(context.Background(), "bad-id")
		if err == nil {
			t.Fatal("expected error in fail mode for invalid ID")
		}
	})
}

// --- GetBookMetadata ---

func TestGetBookMetadataNotSupported(t *testing.T) {
	p := NewAniListProvider(&mockAniListClient{}, false, "ignore")
	result, err := p.GetBookMetadata(context.Background(), "123", "1")
	if !errors.Is(err, metadata.ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got: %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

// --- MatchSeries ---

func TestMatchSeriesNotSupported(t *testing.T) {
	p := NewAniListProvider(&mockAniListClient{}, false, "ignore")
	result, err := p.MatchSeries(context.Background(), "Naruto")
	if !errors.Is(err, metadata.ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got: %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

// --- SearchSeries ---

func TestSearchSeries(t *testing.T) {
	t.Run("returns mapped results", func(t *testing.T) {
		media := sampleMedia()
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, _ string, _ int) ([]anilist.MediaFull, error) {
				return []anilist.MediaFull{*media}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "Naruto", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if results[0].Title != "Naruto" {
			t.Errorf("title: got %q, want Naruto", results[0].Title)
		}
		if results[0].ProviderName != "anilist" {
			t.Errorf("providerName: got %q", results[0].ProviderName)
		}
		if results[0].ResultID != "20" {
			t.Errorf("resultID: got %q", results[0].ResultID)
		}
	})

	t.Run("strips parentheses as fallback when raw query yields zero", func(t *testing.T) {
		media := sampleMedia()
		callCount := 0
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, query string, _ int) ([]anilist.MediaFull, error) {
				callCount++
				if callCount == 1 {
					// First call with raw query returns empty
					if query != "Naruto (2002)" {
						t.Errorf("first call query: got %q, want 'Naruto (2002)'", query)
					}
					return []anilist.MediaFull{}, nil
				}
				// Second call with stripped query returns results
				if query != "Naruto" {
					t.Errorf("second call query: got %q, want 'Naruto'", query)
				}
				return []anilist.MediaFull{*media}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "Naruto (2002)", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 2 {
			t.Errorf("expected 2 search calls, got %d", callCount)
		}
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
	})

	t.Run("strips brackets as fallback", func(t *testing.T) {
		media := sampleMedia()
		callCount := 0
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, query string, _ int) ([]anilist.MediaFull, error) {
				callCount++
				if callCount == 1 {
					return []anilist.MediaFull{}, nil
				}
				return []anilist.MediaFull{*media}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "Naruto [Manga]", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 2 {
			t.Errorf("expected 2 search calls, got %d", callCount)
		}
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
	})

	t.Run("does not retry if stripped query is same as raw", func(t *testing.T) {
		callCount := 0
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, _ string, _ int) ([]anilist.MediaFull, error) {
				callCount++
				return []anilist.MediaFull{}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "Naruto", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 1 {
			t.Errorf("expected 1 search call (no retry since stripped == raw), got %d", callCount)
		}
		if results == nil {
			t.Fatal("expected non-nil empty slice")
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("does not retry if stripped query is empty", func(t *testing.T) {
		callCount := 0
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, _ string, _ int) ([]anilist.MediaFull, error) {
				callCount++
				return []anilist.MediaFull{}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "(2002)", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 1 {
			t.Errorf("expected 1 search call (stripped is empty), got %d", callCount)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("both variants empty returns empty non-nil slice", func(t *testing.T) {
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, _ string, _ int) ([]anilist.MediaFull, error) {
				return []anilist.MediaFull{}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "NonExistent (2099)", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results == nil {
			t.Fatal("expected non-nil empty slice")
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("raw query returns results skips fallback", func(t *testing.T) {
		media := sampleMedia()
		callCount := 0
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, _ string, _ int) ([]anilist.MediaFull, error) {
				callCount++
				return []anilist.MediaFull{*media}, nil
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		results, err := p.SearchSeries(context.Background(), "Naruto (2002)", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 1 {
			t.Errorf("expected 1 search call (raw returned results), got %d", callCount)
		}
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
	})

	t.Run("client error propagated", func(t *testing.T) {
		client := &mockAniListClient{
			searchMediaFullFn: func(_ context.Context, _ string, _ int) ([]anilist.MediaFull, error) {
				return nil, errors.New("api error")
			},
		}
		p := NewAniListProvider(client, false, "ignore")
		_, err := p.SearchSeries(context.Background(), "Naruto", 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- Compile-time interface check ---

var _ metadata.MetadataProvider = (*AniListProvider)(nil)
