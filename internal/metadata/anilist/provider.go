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

// CoverFailureMode controls how GetSeriesCover behaves when the cover cannot be fetched.
type CoverFailureMode string

const (
	// CoverFailureModeIgnore silently returns an empty string on cover fetch errors.
	CoverFailureModeIgnore CoverFailureMode = "ignore"
	// CoverFailureModeFail propagates the error to the caller.
	CoverFailureModeFail CoverFailureMode = "fail"
)

// aniListClient defines the subset of anilist.Client methods used by the provider.
// This enables test mocking without modifying the anilist package.
type aniListClient interface {
	GetMediaFull(ctx context.Context, id int) (*anilist.MediaFull, error)
	SearchMediaFull(ctx context.Context, query string, limit int) ([]anilist.MediaFull, error)
}

type AniListProvider struct {
	client           aniListClient
	excludeSpoilers  bool
	coverFailureMode CoverFailureMode
}

func NewAniListProvider(client aniListClient, excludeSpoilers bool, coverFailureMode CoverFailureMode) *AniListProvider {
	return &AniListProvider{
		client:           client,
		excludeSpoilers:  excludeSpoilers,
		coverFailureMode: coverFailureMode,
	}
}

func (p *AniListProvider) Name() string {
	return "anilist"
}

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

func (p *AniListProvider) GetSeriesCover(ctx context.Context, seriesID string) (string, error) {
	id, err := strconv.Atoi(seriesID)
	if err != nil {
		if p.coverFailureMode == CoverFailureModeIgnore {
			return "", nil
		}
		return "", fmt.Errorf("invalid anilist series ID %q: %w", seriesID, err)
	}

	media, err := p.client.GetMediaFull(ctx, id)
	if err != nil {
		if p.coverFailureMode == CoverFailureModeIgnore {
			return "", nil
		}
		return "", fmt.Errorf("fetch anilist cover %d: %w", id, err)
	}

	return media.CoverImage.Large, nil
}

func (p *AniListProvider) GetBookMetadata(_ context.Context, _, _ string) (*metadata.BookMetadata, error) {
	return nil, metadata.ErrNotSupported
}

// SearchSeries tries the raw query first; if no results, retries with
// parenthetical/bracket content stripped as a fallback.
func (p *AniListProvider) SearchSeries(ctx context.Context, query string, limit int) ([]metadata.SeriesSearchResult, error) {
	results, err := p.client.SearchMediaFull(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search anilist: %w", err)
	}

	if len(results) > 0 {
		return mapSearchResults(results), nil
	}

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

func (p *AniListProvider) MatchSeries(_ context.Context, _ string) (*metadata.SeriesSearchResult, error) {
	return nil, metadata.ErrNotSupported
}

func mapSearchResults(media []anilist.MediaFull) []metadata.SeriesSearchResult {
	results := make([]metadata.SeriesSearchResult, len(media))
	for i, m := range media {
		results[i] = MapMediaToSearchResult(m)
	}
	return results
}
