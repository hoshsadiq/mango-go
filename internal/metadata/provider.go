package metadata

import (
	"context"
	"errors"
)

// ErrNotSupported is returned when a provider does not support a particular operation
var ErrNotSupported = errors.New("operation not supported by this provider")

// MetadataProvider defines the interface for metadata providers
// Implementations fetch metadata from external sources (e.g., AniList)
type MetadataProvider interface {
	// Name returns the name of the provider (e.g., "AniList")
	Name() string

	// GetSeriesMetadata fetches comprehensive metadata for a series by its provider ID
	GetSeriesMetadata(ctx context.Context, seriesID string) (*SeriesMetadata, error)

	// GetSeriesCover fetches the cover image URL for a series by its provider ID
	GetSeriesCover(ctx context.Context, seriesID string) (string, error)

	// GetBookMetadata fetches metadata for a specific book/volume in a series
	// May return ErrNotSupported if the provider doesn't support book-level metadata
	GetBookMetadata(ctx context.Context, seriesID, bookID string) (*BookMetadata, error)

	// SearchSeries searches for series matching a query
	// Returns up to 'limit' results; empty slice (not nil) if no results found
	SearchSeries(ctx context.Context, query string, limit int) ([]SeriesSearchResult, error)

	// MatchSeries attempts to find a single best match for a series name
	// Returns nil if no match found (not an error)
	MatchSeries(ctx context.Context, seriesName string) (*SeriesSearchResult, error)
}
