package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
)

// metadataLocksResponse is the JSON shape for per-field lock states.
type metadataLocksResponse struct {
	Status           bool `json:"status"`
	Title            bool `json:"title"`
	Summary          bool `json:"summary"`
	Publisher        bool `json:"publisher"`
	ReadingDirection bool `json:"reading_direction"`
	AgeRating        bool `json:"age_rating"`
	Language         bool `json:"language"`
	TotalBookCount   bool `json:"total_book_count"`
	CommunityScore   bool `json:"community_score"`
	ReleaseDate      bool `json:"release_date"`
	ThumbnailURL     bool `json:"thumbnail_url"`
	Genres           bool `json:"genres"`
	Tags             bool `json:"tags"`
	Authors          bool `json:"authors"`
	Links            bool `json:"links"`
	Titles           bool `json:"titles"`
}

// metadataFieldsResponse is the JSON shape for the metadata object including locks.
type metadataFieldsResponse struct {
	Status           *string                `json:"status"`
	Title            *string                `json:"title"`
	Titles           []metadata.SeriesTitle `json:"titles"`
	Summary          *string                `json:"summary"`
	Publisher        *string                `json:"publisher"`
	ReadingDirection *string                `json:"reading_direction"`
	AgeRating        *int64                 `json:"age_rating"`
	Language         *string                `json:"language"`
	Genres           []string               `json:"genres"`
	Tags             []string               `json:"tags"`
	TotalBookCount   *int64                 `json:"total_book_count"`
	Authors          []metadata.Author      `json:"authors"`
	ReleaseYear      *int64                 `json:"release_year"`
	ReleaseMonth     *int64                 `json:"release_month"`
	ReleaseDay       *int64                 `json:"release_day"`
	Links            []metadata.WebLink     `json:"links"`
	CommunityScore   *float64               `json:"community_score"`
	ThumbnailURL     *string                `json:"thumbnail_url"`
	Locks            metadataLocksResponse  `json:"locks"`
}

// providerResponse is the JSON shape for the provider link.
type providerResponse struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// getMetadataResponse is the top-level JSON response for GET /api/folders/{folderID}/metadata.
type getMetadataResponse struct {
	Metadata metadataFieldsResponse `json:"metadata"`
	Provider *providerResponse      `json:"provider"`
}

// searchMetadataResponse is the JSON response for GET /api/metadata/search.
type searchMetadataResponse struct {
	Results []metadata.SeriesSearchResult `json:"results"`
}

// buildMetadataFieldsResponse converts a store.SeriesMetadataRow to the API response DTO.
func buildMetadataFieldsResponse(row *store.SeriesMetadataRow) metadataFieldsResponse {
	resp := metadataFieldsResponse{
		Locks: metadataLocksResponse{
			Status:           row.StatusLock,
			Title:            row.TitleLock,
			Summary:          row.SummaryLock,
			Publisher:        row.PublisherLock,
			ReadingDirection: row.ReadingDirectionLock,
			AgeRating:        row.AgeRatingLock,
			Language:         row.LanguageLock,
			TotalBookCount:   row.TotalBookCountLock,
			CommunityScore:   row.CommunityScoreLock,
			ReleaseDate:      row.ReleaseDateLock,
			ThumbnailURL:     row.ThumbnailURLLock,
			Genres:           row.GenresLock,
			Tags:             row.TagsLock,
			Authors:          row.AuthorsLock,
			Links:            row.LinksLock,
			Titles:           row.TitlesLock,
		},
	}

	// Convert sql.Null* types to pointers for JSON null serialization.
	if row.Status.Valid {
		resp.Status = &row.Status.String
	}
	if row.Title.Valid {
		resp.Title = &row.Title.String
	}
	if row.Summary.Valid {
		resp.Summary = &row.Summary.String
	}
	if row.Publisher.Valid {
		resp.Publisher = &row.Publisher.String
	}
	if row.ReadingDirection.Valid {
		resp.ReadingDirection = &row.ReadingDirection.String
	}
	if row.AgeRating.Valid {
		resp.AgeRating = &row.AgeRating.Int64
	}
	if row.Language.Valid {
		resp.Language = &row.Language.String
	}
	if row.TotalBookCount.Valid {
		resp.TotalBookCount = &row.TotalBookCount.Int64
	}
	if row.CommunityScore.Valid {
		resp.CommunityScore = &row.CommunityScore.Float64
	}
	if row.ReleaseYear.Valid {
		resp.ReleaseYear = &row.ReleaseYear.Int64
	}
	if row.ReleaseMonth.Valid {
		resp.ReleaseMonth = &row.ReleaseMonth.Int64
	}
	if row.ReleaseDay.Valid {
		resp.ReleaseDay = &row.ReleaseDay.Int64
	}
	if row.ThumbnailURL.Valid {
		resp.ThumbnailURL = &row.ThumbnailURL.String
	}

	// Ensure slices are never null in JSON — always [].
	resp.Genres = row.Genres
	if resp.Genres == nil {
		resp.Genres = []string{}
	}
	resp.Tags = row.Tags
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	resp.Authors = row.Authors
	if resp.Authors == nil {
		resp.Authors = []metadata.Author{}
	}
	resp.Links = row.Links
	if resp.Links == nil {
		resp.Links = []metadata.WebLink{}
	}
	resp.Titles = row.Titles
	if resp.Titles == nil {
		resp.Titles = []metadata.SeriesTitle{}
	}

	return resp
}

// handleGetMetadata returns metadata for a folder, or 404 if none exists.
func (s *Server) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	row, err := s.store.GetSeriesMetadata(folderID)
	if err != nil {
		if errors.Is(err, store.ErrMetadataNotFound) {
			RespondWithError(w, http.StatusNotFound, "no metadata found for this folder")
			return
		}
		log.Printf("GetSeriesMetadata(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
		return
	}

	resp := getMetadataResponse{
		Metadata: buildMetadataFieldsResponse(row),
	}

	// Provider link is optional — missing link means provider: null, not a 404.
	link, err := s.store.GetProviderLink(folderID)
	if err != nil {
		if !errors.Is(err, store.ErrProviderLinkNotFound) {
			log.Printf("GetProviderLink(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load provider link")
			return
		}
		// ErrProviderLinkNotFound → provider stays nil
	} else {
		resp.Provider = &providerResponse{
			Name: link.ProviderName,
			ID:   link.ProviderID,
		}
	}

	RespondWithJSON(w, http.StatusOK, resp)
}

// handleSearchMetadata searches for series via the configured metadata provider.
func (s *Server) handleSearchMetadata(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		RespondWithError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 25 {
		limit = 25
	}

	if s.metadataProvider == nil {
		RespondWithError(w, http.StatusInternalServerError, "No metadata provider configured")
		return
	}

	results, err := s.metadataProvider.SearchSeries(r.Context(), q, limit)
	if err != nil {
		log.Printf("SearchSeries(%q, %d): %v", q, limit, err)
		RespondWithError(w, http.StatusBadGateway, "Failed to search metadata provider")
		return
	}

	// Ensure results is never null in JSON.
	if results == nil {
		results = []metadata.SeriesSearchResult{}
	}

	RespondWithJSON(w, http.StatusOK, searchMetadataResponse{Results: results})
}
