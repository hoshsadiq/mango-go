package api

import (
	"encoding/json"
	"errors"
	"fmt"
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

// linkMetadataRequest is the JSON body for POST /api/folders/{folderID}/metadata/link.
type linkMetadataRequest struct {
	ProviderName string `json:"provider_name"`
	ProviderID   string `json:"provider_id"`
}

// knownProviders is the set of valid provider names.
var knownProviders = map[string]bool{
	"anilist": true,
}

// fetchAndStoreMetadata fetches metadata (and optionally cover) from the provider,
// upserts the metadata into the store, and returns the updated metadata response.
// It is shared by handleLinkMetadata and handleRefreshMetadata.
func (s *Server) fetchAndStoreMetadata(w http.ResponseWriter, r *http.Request, folderID int64, providerID string) {
	ctx := r.Context()

	meta, err := s.metadataProvider.GetSeriesMetadata(ctx, providerID)
	if err != nil {
		log.Printf("GetSeriesMetadata(provider=%q, id=%q): %v", s.metadataProvider.Name(), providerID, err)
		RespondWithError(w, http.StatusBadGateway, "failed to fetch metadata from provider: "+err.Error())
		return
	}

	coverURL, err := s.metadataProvider.GetSeriesCover(ctx, providerID)
	if err != nil {
		log.Printf("GetSeriesCover(provider=%q, id=%q): %v", s.metadataProvider.Name(), providerID, err)
		RespondWithError(w, http.StatusBadGateway, "failed to fetch cover from provider: "+err.Error())
		return
	}
	if coverURL != "" {
		meta.ThumbnailURL = &coverURL
	}

	if err := s.store.UpsertSeriesMetadata(folderID, meta); err != nil {
		log.Printf("UpsertSeriesMetadata(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to store metadata")
		return
	}

	row, err := s.store.GetSeriesMetadata(folderID)
	if err != nil {
		log.Printf("GetSeriesMetadata(%d) after upsert: %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
		return
	}

	resp := getMetadataResponse{
		Metadata: buildMetadataFieldsResponse(row),
	}

	link, err := s.store.GetProviderLink(folderID)
	if err != nil {
		if !errors.Is(err, store.ErrProviderLinkNotFound) {
			log.Printf("GetProviderLink(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load provider link")
			return
		}
	} else {
		resp.Provider = &providerResponse{
			Name: link.ProviderName,
			ID:   link.ProviderID,
		}
	}

	RespondWithJSON(w, http.StatusOK, resp)
}

// handleLinkMetadata links a folder to a metadata provider and fetches metadata.
func (s *Server) handleLinkMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	var req linkMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !knownProviders[req.ProviderName] {
		RespondWithError(w, http.StatusBadRequest, "unknown provider_name: "+req.ProviderName)
		return
	}
	if req.ProviderID == "" {
		RespondWithError(w, http.StatusBadRequest, "provider_id must not be empty")
		return
	}

	if s.metadataProvider == nil {
		RespondWithError(w, http.StatusInternalServerError, "No metadata provider configured")
		return
	}

	if err := s.store.UpsertProviderLink(folderID, req.ProviderName, req.ProviderID); err != nil {
		log.Printf("UpsertProviderLink(%d, %q, %q): %v", folderID, req.ProviderName, req.ProviderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to store provider link")
		return
	}

	s.fetchAndStoreMetadata(w, r, folderID, req.ProviderID)
}

// handleRefreshMetadata re-fetches metadata from the linked provider.
func (s *Server) handleRefreshMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	link, err := s.store.GetProviderLink(folderID)
	if err != nil {
		if errors.Is(err, store.ErrProviderLinkNotFound) {
			RespondWithError(w, http.StatusNotFound, "no provider link found for this folder")
			return
		}
		log.Printf("GetProviderLink(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load provider link")
		return
	}

	if s.metadataProvider == nil {
		RespondWithError(w, http.StatusInternalServerError, "No metadata provider configured")
		return
	}

	s.fetchAndStoreMetadata(w, r, folderID, link.ProviderID)
}

// handleResetMetadata clears all metadata fields but preserves the provider link.
func (s *Server) handleResetMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	// Check existence first — ResetSeriesMetadata silently succeeds when no row exists.
	_, err = s.store.GetSeriesMetadata(folderID)
	if err != nil {
		if errors.Is(err, store.ErrMetadataNotFound) {
			RespondWithError(w, http.StatusNotFound, "no metadata found for this folder")
			return
		}
		log.Printf("GetSeriesMetadata(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
		return
	}

	if err := s.store.ResetSeriesMetadata(folderID); err != nil {
		log.Printf("ResetSeriesMetadata(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to reset metadata")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":                  "reset",
		"provider_link_preserved": true,
	})
}

// handleUnlinkMetadata clears all metadata and removes the provider link.
func (s *Server) handleUnlinkMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	// Check existence first — UnlinkSeriesMetadata silently succeeds when no row exists.
	_, err = s.store.GetSeriesMetadata(folderID)
	if err != nil {
		if errors.Is(err, store.ErrMetadataNotFound) {
			RespondWithError(w, http.StatusNotFound, "no metadata found for this folder")
			return
		}
		log.Printf("GetSeriesMetadata(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
		return
	}

	if err := s.store.UnlinkSeriesMetadata(folderID); err != nil {
		log.Printf("UnlinkSeriesMetadata(%d): %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to unlink metadata")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"status": "unlinked",
	})
}

// validStatuses is the set of accepted SeriesStatus values for the edit endpoint.
var validStatuses = map[string]bool{
	"ONGOING":   true,
	"COMPLETED": true,
	"ABANDONED": true,
	"HIATUS":    true,
}

// validReadingDirections is the set of accepted ReadingDirection values for the edit endpoint.
var validReadingDirections = map[string]bool{
	"LEFT_TO_RIGHT": true,
	"RIGHT_TO_LEFT": true,
	"VERTICAL":      true,
	"WEBTOON":       true,
}

// validLockFieldNames enumerates the lock column names accepted by the locks endpoint.
var validLockFieldNames = map[string]bool{
	"status_lock":            true,
	"title_lock":             true,
	"summary_lock":           true,
	"publisher_lock":         true,
	"reading_direction_lock": true,
	"age_rating_lock":        true,
	"language_lock":          true,
	"total_book_count_lock":  true,
	"community_score_lock":   true,
	"release_date_lock":      true,
	"thumbnail_url_lock":     true,
	"genres_lock":            true,
	"tags_lock":              true,
	"authors_lock":           true,
	"links_lock":             true,
	"titles_lock":            true,
}

// handleEditMetadata handles PATCH /api/folders/{folderID}/metadata.
// Only fields PRESENT in the JSON body are updated; absent fields are left
// untouched; a field present with explicit null clears it (and auto-locks).
func (s *Server) handleEditMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Collect validated field updates before applying any.
	type fieldUpdate struct {
		name  string
		value interface{} // nil means clear (set to NULL)
	}
	var updates []fieldUpdate

	for key, raw := range body {
		isNull := string(raw) == "null"

		switch key {
		case "status":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v string
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for status: expected string")
					return
				}
				if !validStatuses[v] {
					RespondWithError(w, http.StatusBadRequest, "invalid status: must be one of ONGOING, COMPLETED, ABANDONED, HIATUS")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "title", "summary", "publisher", "language", "thumbnail_url":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v string
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid value for %s: expected string", key))
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "reading_direction":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v string
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for reading_direction: expected string")
					return
				}
				if !validReadingDirections[v] {
					RespondWithError(w, http.StatusBadRequest, "invalid reading_direction: must be one of LEFT_TO_RIGHT, RIGHT_TO_LEFT, VERTICAL, WEBTOON")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "age_rating":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v int64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for age_rating: expected integer")
					return
				}
				if v < 0 {
					RespondWithError(w, http.StatusBadRequest, "invalid age_rating: must be non-negative")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "total_book_count":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v int64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for total_book_count: expected integer")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "community_score":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v float64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for community_score: expected number")
					return
				}
				if v < 0.0 || v > 10.0 {
					RespondWithError(w, http.StatusBadRequest, "invalid community_score: must be between 0.0 and 10.0")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "release_year":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v int64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for release_year: expected integer")
					return
				}
				if v < 1000 || v > 9999 {
					RespondWithError(w, http.StatusBadRequest, "invalid release_year: must be a 4-digit year")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "release_month":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v int64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for release_month: expected integer")
					return
				}
				if v < 1 || v > 12 {
					RespondWithError(w, http.StatusBadRequest, "invalid release_month: must be between 1 and 12")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		case "release_day":
			if isNull {
				updates = append(updates, fieldUpdate{key, nil})
			} else {
				var v int64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for release_day: expected integer")
					return
				}
				if v < 1 || v > 31 {
					RespondWithError(w, http.StatusBadRequest, "invalid release_day: must be between 1 and 31")
					return
				}
				updates = append(updates, fieldUpdate{key, v})
			}

		default:
			// Unknown or collection fields — ignored (Task 10 is scalar-only).
		}
	}

	// Ensure metadata row exists (Task 10: create if absent, do not 404).
	if _, err := s.store.GetSeriesMetadata(folderID); err != nil {
		if !errors.Is(err, store.ErrMetadataNotFound) {
			log.Printf("GetSeriesMetadata(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
			return
		}
		if err := s.store.UpsertSeriesMetadata(folderID, &metadata.SeriesMetadata{}); err != nil {
			log.Printf("UpsertSeriesMetadata(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to create metadata record")
			return
		}
	}

	// Apply all validated field updates.
	for _, u := range updates {
		if err := s.store.UpdateMetadataField(folderID, u.name, u.value); err != nil {
			log.Printf("UpdateMetadataField(%d, %q): %v", folderID, u.name, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to update field: "+u.name)
			return
		}
	}

	// Read back the full metadata and return in same format as GET.
	row, err := s.store.GetSeriesMetadata(folderID)
	if err != nil {
		log.Printf("GetSeriesMetadata(%d) after edit: %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata after update")
		return
	}

	resp := getMetadataResponse{
		Metadata: buildMetadataFieldsResponse(row),
	}

	link, err := s.store.GetProviderLink(folderID)
	if err != nil {
		if !errors.Is(err, store.ErrProviderLinkNotFound) {
			log.Printf("GetProviderLink(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load provider link")
			return
		}
	} else {
		resp.Provider = &providerResponse{
			Name: link.ProviderName,
			ID:   link.ProviderID,
		}
	}

	RespondWithJSON(w, http.StatusOK, resp)
}

// handleEditLocks handles PATCH /api/folders/{folderID}/metadata/locks.
// Only lock keys present in the JSON body are toggled. Returns 404 if no
// metadata record exists. Returns the complete current lock state.
func (s *Server) handleEditLocks(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	locks := make(map[string]bool)
	for key, raw := range body {
		if !validLockFieldNames[key] {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("unknown lock field: %s", key))
			return
		}
		var v bool
		if err := json.Unmarshal(raw, &v); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid value for %s: expected boolean", key))
			return
		}
		locks[key] = v
	}

	if len(locks) > 0 {
		if err := s.store.UpdateMetadataLocks(folderID, locks); err != nil {
			if errors.Is(err, store.ErrMetadataNotFound) {
				RespondWithError(w, http.StatusNotFound, "no metadata found for this folder")
				return
			}
			log.Printf("UpdateMetadataLocks(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to update locks")
			return
		}
	} else {
		// Empty body — still check existence for 404.
		if _, err := s.store.GetSeriesMetadata(folderID); err != nil {
			if errors.Is(err, store.ErrMetadataNotFound) {
				RespondWithError(w, http.StatusNotFound, "no metadata found for this folder")
				return
			}
			log.Printf("GetSeriesMetadata(%d): %v", folderID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
			return
		}
	}

	// Read back full lock state.
	row, err := s.store.GetSeriesMetadata(folderID)
	if err != nil {
		log.Printf("GetSeriesMetadata(%d) after lock update: %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load metadata")
		return
	}

	RespondWithJSON(w, http.StatusOK, metadataLocksResponse{
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
	})
}
