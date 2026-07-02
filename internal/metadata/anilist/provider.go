package anilist

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/metadata"
)

var bracketStripRe = regexp.MustCompile(`\s*[\(\[\{][^\)\]\}]*[\)\]\}]\s*`)

// aniListClient defines the subset of anilist.Client methods used by the provider.
// This enables test mocking without modifying the anilist package.
type aniListClient interface {
	GetMediaFull(ctx context.Context, id int) (*anilist.MediaFull, error)
	SearchMediaFull(ctx context.Context, query string, limit int) ([]anilist.MediaFull, error)
}

// AniListProvider implements metadata.MetadataProvider using the AniList API.
type AniListProvider struct {
	client           aniListClient
	excludeSpoilers  bool
	coverFailureMode string
}

// NewAniListProvider creates a new AniList metadata provider.
func NewAniListProvider(client aniListClient, excludeSpoilers bool, coverFailureMode string) *AniListProvider {
	return &AniListProvider{
		client:           client,
		excludeSpoilers:  excludeSpoilers,
		coverFailureMode: coverFailureMode,
	}
}

// Name returns the provider name.
func (p *AniListProvider) Name() string {
	return "anilist"
}

// GetSeriesMetadata fetches and maps full metadata for a series by AniList ID.
func (p *AniListProvider) GetSeriesMetadata(ctx context.Context, seriesID string) (*metadata.SeriesMetadata, error) {
	id, err := strconv.Atoi(seriesID)
	if err != nil {
		return nil, fmt.Errorf("invalid anilist series ID %q: %w", seriesID, err)
	}

	media, err := p.client.GetMediaFull(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch anilist media %d: %w", id, err)
	}

	return MapMediaToSeriesMetadata(*media, p.excludeSpoilers), nil
}

// GetSeriesCover fetches the cover image URL for a series.
// Behavior on failure depends on coverFailureMode: "ignore" returns ("", nil), "fail" returns the error.
func (p *AniListProvider) GetSeriesCover(ctx context.Context, seriesID string) (string, error) {
	id, err := strconv.Atoi(seriesID)
	if err != nil {
		if p.coverFailureMode == "ignore" {
			return "", nil
		}
		return "", fmt.Errorf("invalid anilist series ID %q: %w", seriesID, err)
	}

	media, err := p.client.GetMediaFull(ctx, id)
	if err != nil {
		if p.coverFailureMode == "ignore" {
			return "", nil
		}
		return "", fmt.Errorf("fetch anilist cover %d: %w", id, err)
	}

	return media.CoverImage.Large, nil
}

// GetBookMetadata returns ErrNotSupported — AniList has no book-level metadata.
func (p *AniListProvider) GetBookMetadata(_ context.Context, _, _ string) (*metadata.BookMetadata, error) {
	return nil, metadata.ErrNotSupported
}

// SearchSeries searches AniList for manga matching the query.
// If the raw query yields zero results and stripping parenthetical/bracket content
// produces a different non-empty string, that variant is tried as a fallback.
func (p *AniListProvider) SearchSeries(ctx context.Context, query string, limit int) ([]metadata.SeriesSearchResult, error) {
	// Try raw query first
	results, err := p.client.SearchMediaFull(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search anilist: %w", err)
	}

	if len(results) > 0 {
		return mapSearchResults(results), nil
	}

	// Try stripped variant if different and non-empty
	stripped := strings.TrimSpace(bracketStripRe.ReplaceAllString(query, " "))
	stripped = strings.TrimSpace(stripped)
	if stripped != "" && stripped != query {
		results, err = p.client.SearchMediaFull(ctx, stripped, limit)
		if err != nil {
			return nil, fmt.Errorf("search anilist (stripped): %w", err)
		}
		if len(results) > 0 {
			return mapSearchResults(results), nil
		}
	}

	return []metadata.SeriesSearchResult{}, nil
}

// MatchSeries returns ErrNotSupported — auto-matching is Plan 4 territory.
func (p *AniListProvider) MatchSeries(_ context.Context, _ string) (*metadata.SeriesSearchResult, error) {
	return nil, metadata.ErrNotSupported
}

// mapSearchResults converts a slice of MediaFull to SeriesSearchResult.
func mapSearchResults(media []anilist.MediaFull) []metadata.SeriesSearchResult {
	results := make([]metadata.SeriesSearchResult, len(media))
	for i, m := range media {
		results[i] = MapMediaToSearchResult(m)
	}
	return results
}
