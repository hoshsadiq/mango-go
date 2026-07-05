package api

import (
	"database/sql"
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

var bulkApplicableFields = map[string]bool{
	"scanlation_group": true,
	"chapter_type":     true,
	"language":         true,
	"age_rating":       true,
	"tags":             true,
	"genres":           true,
}

const maxBulkChapterIDs = 500

// bulkUpdateRequest is the JSON body for PATCH /api/folders/{folderID}/chapters/metadata/bulk.
type bulkUpdateRequest struct {
	ChapterIDs []int64                    `json:"chapter_ids"`
	Fields     map[string]json.RawMessage `json:"fields"`
}

// bulkSkippedEntry represents a chapter that was skipped during bulk update.
type bulkSkippedEntry struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

// bulkUpdateResponse is the JSON response for the bulk update endpoint.
type bulkUpdateResponse struct {
	Updated []int64            `json:"updated"`
	Skipped []bulkSkippedEntry `json:"skipped"`
	Total   int                `json:"total"`
}

// handleBulkUpdateChapterMetadata handles PATCH /api/folders/{folderID}/chapters/metadata/bulk.
// It applies the same metadata fields to multiple chapters in a single transaction.
// Locked fields are skipped per-chapter; updated fields are auto-locked afterward.
func (s *Server) handleBulkUpdateChapterMetadata(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	var req bulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate chapter_ids count.
	if len(req.ChapterIDs) == 0 {
		RespondWithError(w, http.StatusBadRequest, "chapter_ids must not be empty")
		return
	}
	if len(req.ChapterIDs) > maxBulkChapterIDs {
		RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("chapter_ids exceeds maximum of %d", maxBulkChapterIDs))
		return
	}

	// Validate that only bulk-applicable fields are present.
	if len(req.Fields) == 0 {
		RespondWithError(w, http.StatusBadRequest, "fields must not be empty")
		return
	}
	for fieldName := range req.Fields {
		if !bulkApplicableFields[fieldName] {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("field %q is not allowed in bulk updates", fieldName))
			return
		}
	}

	// Parse the fields into a ChapterMetadata struct.
	meta, err := parseBulkFields(req.Fields)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build the updates map: same metadata for all chapter IDs.
	cms := s.app.ChapterMetadataStore
	if cms == nil {
		cms = store.NewChapterMetadataStore(s.db)
	}

	updates := make(map[int64]*metadata.ChapterMetadata, len(req.ChapterIDs))
	for _, id := range req.ChapterIDs {
		updates[id] = meta
	}

	updated, skipped, err := cms.BulkUpdateChapterMetadata(updates)
	if err != nil {
		if errors.Is(err, store.ErrBulkUpdateLimitExceeded) {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("BulkUpdateChapterMetadata: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to bulk update chapter metadata")
		return
	}

	// Auto-lock the updated fields on all successfully updated chapters.
	for _, chapterID := range updated {
		existing, err := cms.GetChapterMetadata(chapterID)
		if err != nil {
			log.Printf("GetChapterMetadata(%d) for auto-lock: %v", chapterID, err)
			continue
		}
		locks := mergeAutoLocks(existing, req.Fields)
		if err := cms.UpdateChapterMetadataLocks(chapterID, locks); err != nil {
			log.Printf("UpdateChapterMetadataLocks(%d): %v", chapterID, err)
		}
	}

	// Build skipped entries with reasons.
	skippedEntries := buildSkippedEntries(cms, skipped, req.Fields)

	// Ensure slices are never null in JSON.
	if updated == nil {
		updated = []int64{}
	}
	if skippedEntries == nil {
		skippedEntries = []bulkSkippedEntry{}
	}

	resp := bulkUpdateResponse{
		Updated: updated,
		Skipped: skippedEntries,
		Total:   len(req.ChapterIDs),
	}

	RespondWithJSON(w, http.StatusOK, resp)
}

// parseBulkFields converts the raw JSON fields map into a ChapterMetadata struct.
func parseBulkFields(fields map[string]json.RawMessage) (*metadata.ChapterMetadata, error) {
	meta := &metadata.ChapterMetadata{}

	for name, raw := range fields {
		switch name {
		case "scanlation_group":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid value for scanlation_group: expected string")
			}
			meta.ScanlationGroup = &v

		case "chapter_type":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid value for chapter_type: expected string")
			}
			meta.ChapterType = &v

		case "language":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid value for language: expected string")
			}
			meta.Language = &v

		case "age_rating":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid value for age_rating: expected string")
			}
			meta.AgeRating = &v

		case "tags":
			var v []string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid value for tags: expected array of strings")
			}
			meta.Tags = v

		case "genres":
			var v []string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid value for genres: expected array of strings")
			}
			meta.Genres = v
		}
	}

	return meta, nil
}

// buildSkippedEntries determines the skip reason for each skipped chapter by
// checking which requested fields were locked.
func buildSkippedEntries(cms *store.ChapterMetadataStore, skippedIDs []int64, requestedFields map[string]json.RawMessage) []bulkSkippedEntry {
	entries := make([]bulkSkippedEntry, 0, len(skippedIDs))

	for _, id := range skippedIDs {
		reason := "locked"
		// Try to determine which specific field was locked.
		m, err := cms.GetChapterMetadata(id)
		if err == nil && m != nil {
			reason = determineSkipReason(m, requestedFields)
		}
		entries = append(entries, bulkSkippedEntry{ID: id, Reason: reason})
	}

	return entries
}

// determineSkipReason checks which requested fields are locked on the chapter
// and returns a descriptive reason string.
func determineSkipReason(m *metadata.ChapterMetadata, requestedFields map[string]json.RawMessage) string {
	var lockedFields []string

	for name := range requestedFields {
		switch name {
		case "scanlation_group":
			if m.ScanlationGroupLock {
				lockedFields = append(lockedFields, "scanlation_group_lock")
			}
		case "chapter_type":
			if m.ChapterTypeLock {
				lockedFields = append(lockedFields, "chapter_type_lock")
			}
		case "language":
			if m.LanguageLock {
				lockedFields = append(lockedFields, "language_lock")
			}
		case "age_rating":
			if m.AgeRatingLock {
				lockedFields = append(lockedFields, "age_rating_lock")
			}
		}
	}

	if len(lockedFields) > 0 {
		return lockedFields[0]
	}
	return "locked"
}

func mergeAutoLocks(existing *metadata.ChapterMetadata, fields map[string]json.RawMessage) *metadata.ChapterMetadataLocks {
	locks := &metadata.ChapterMetadataLocks{}
	if existing != nil {
		locks.TitleLock = existing.TitleLock
		locks.NumberLock = existing.NumberLock
		locks.SortNumberLock = existing.SortNumberLock
		locks.VolumeLock = existing.VolumeLock
		locks.SummaryLock = existing.SummaryLock
		locks.NotesLock = existing.NotesLock
		locks.ReleaseDateLock = existing.ReleaseDateLock
		locks.LanguageLock = existing.LanguageLock
		locks.ChapterTypeLock = existing.ChapterTypeLock
		locks.AgeRatingLock = existing.AgeRatingLock
		locks.WebLock = existing.WebLock
		locks.CharactersLock = existing.CharactersLock
		locks.TeamsLock = existing.TeamsLock
		locks.LocationsLock = existing.LocationsLock
		locks.ScanlationGroupLock = existing.ScanlationGroupLock
		locks.StoryArcLock = existing.StoryArcLock
		locks.StoryArcNumberLock = existing.StoryArcNumberLock
	}
	for name := range fields {
		switch name {
		case "scanlation_group":
			locks.ScanlationGroupLock = true
		case "chapter_type":
			locks.ChapterTypeLock = true
		case "language":
			locks.LanguageLock = true
		case "age_rating":
			locks.AgeRatingLock = true
		}
	}
	return locks
}

// --- Individual chapter metadata endpoints ---

type chapterMetadataLocksResponse struct {
	Title           bool `json:"title"`
	Number          bool `json:"number"`
	SortNumber      bool `json:"sort_number"`
	Volume          bool `json:"volume"`
	Summary         bool `json:"summary"`
	Notes           bool `json:"notes"`
	ReleaseDate     bool `json:"release_date"`
	Language        bool `json:"language"`
	ChapterType     bool `json:"chapter_type"`
	AgeRating       bool `json:"age_rating"`
	Web             bool `json:"web"`
	Characters      bool `json:"characters"`
	Teams           bool `json:"teams"`
	Locations       bool `json:"locations"`
	ScanlationGroup bool `json:"scanlation_group"`
	StoryArc        bool `json:"story_arc"`
	StoryArcNumber  bool `json:"story_arc_number"`
}

type chapterMetadataFieldsResponse struct {
	Title           *string                          `json:"title"`
	Number          *string                          `json:"number"`
	SortNumber      *float64                         `json:"sort_number"`
	Volume          *string                          `json:"volume"`
	Summary         *string                          `json:"summary"`
	Notes           *string                          `json:"notes"`
	ReleaseDate     *string                          `json:"release_date"`
	Language        *string                          `json:"language"`
	ChapterType     *string                          `json:"chapter_type"`
	AgeRating       *string                          `json:"age_rating"`
	Web             *string                          `json:"web"`
	Characters      *string                          `json:"characters"`
	Teams           *string                          `json:"teams"`
	Locations       *string                          `json:"locations"`
	ScanlationGroup *string                          `json:"scanlation_group"`
	StoryArc        *string                          `json:"story_arc"`
	StoryArcNumber  *string                          `json:"story_arc_number"`
	Authors         []metadata.ChapterMetadataAuthor `json:"authors"`
	Genres          []string                         `json:"genres"`
	Tags            []string                         `json:"tags"`
	Locks           chapterMetadataLocksResponse     `json:"locks"`
}

type getChapterMetadataResponse struct {
	Metadata chapterMetadataFieldsResponse `json:"metadata"`
}

func buildChapterMetadataLocksFromMeta(meta *metadata.ChapterMetadata) chapterMetadataLocksResponse {
	if meta == nil {
		return chapterMetadataLocksResponse{}
	}
	return chapterMetadataLocksResponse{
		Title:           meta.TitleLock,
		Number:          meta.NumberLock,
		SortNumber:      meta.SortNumberLock,
		Volume:          meta.VolumeLock,
		Summary:         meta.SummaryLock,
		Notes:           meta.NotesLock,
		ReleaseDate:     meta.ReleaseDateLock,
		Language:        meta.LanguageLock,
		ChapterType:     meta.ChapterTypeLock,
		AgeRating:       meta.AgeRatingLock,
		Web:             meta.WebLock,
		Characters:      meta.CharactersLock,
		Teams:           meta.TeamsLock,
		Locations:       meta.LocationsLock,
		ScanlationGroup: meta.ScanlationGroupLock,
		StoryArc:        meta.StoryArcLock,
		StoryArcNumber:  meta.StoryArcNumberLock,
	}
}

func buildChapterMetadataFieldsResponse(meta *metadata.ChapterMetadata) chapterMetadataFieldsResponse {
	resp := chapterMetadataFieldsResponse{
		Authors: []metadata.ChapterMetadataAuthor{},
		Genres:  []string{},
		Tags:    []string{},
	}
	if meta == nil {
		return resp
	}

	resp.Title = meta.Title
	resp.Number = meta.Number
	resp.SortNumber = meta.SortNumber
	resp.Volume = meta.Volume
	resp.Summary = meta.Summary
	resp.Notes = meta.Notes
	resp.ReleaseDate = meta.ReleaseDate
	resp.Language = meta.Language
	resp.ChapterType = meta.ChapterType
	resp.AgeRating = meta.AgeRating
	resp.Web = meta.Web
	resp.Characters = meta.Characters
	resp.Teams = meta.Teams
	resp.Locations = meta.Locations
	resp.ScanlationGroup = meta.ScanlationGroup
	resp.StoryArc = meta.StoryArc
	resp.StoryArcNumber = meta.StoryArcNumber

	if meta.Authors != nil {
		resp.Authors = meta.Authors
	}
	if meta.Genres != nil {
		resp.Genres = meta.Genres
	}
	if meta.Tags != nil {
		resp.Tags = meta.Tags
	}

	resp.Locks = buildChapterMetadataLocksFromMeta(meta)
	return resp
}

func (s *Server) chapterExists(chapterID int64) (bool, error) {
	var exists int
	err := s.db.QueryRow("SELECT 1 FROM chapters WHERE id = ?", chapterID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Server) chapterMetadataStore() *store.ChapterMetadataStore {
	if s.app.ChapterMetadataStore != nil {
		return s.app.ChapterMetadataStore
	}
	return store.NewChapterMetadataStore(s.db)
}

var validChapterTypes = map[string]bool{
	metadata.ChapterTypeRegular: true,
	metadata.ChapterTypeSpecial: true,
	metadata.ChapterTypeBonus:   true,
	metadata.ChapterTypeOmake:   true,
	metadata.ChapterTypeOneShot: true,
	metadata.ChapterTypeAnnual:  true,
	metadata.ChapterTypeOmnibus: true,
}

var chapterMetadataFieldToLock = map[string]func(*metadata.ChapterMetadata) bool{
	"title":            func(m *metadata.ChapterMetadata) bool { return m.TitleLock },
	"number":           func(m *metadata.ChapterMetadata) bool { return m.NumberLock },
	"sort_number":      func(m *metadata.ChapterMetadata) bool { return m.SortNumberLock },
	"volume":           func(m *metadata.ChapterMetadata) bool { return m.VolumeLock },
	"summary":          func(m *metadata.ChapterMetadata) bool { return m.SummaryLock },
	"notes":            func(m *metadata.ChapterMetadata) bool { return m.NotesLock },
	"release_date":     func(m *metadata.ChapterMetadata) bool { return m.ReleaseDateLock },
	"language":         func(m *metadata.ChapterMetadata) bool { return m.LanguageLock },
	"chapter_type":     func(m *metadata.ChapterMetadata) bool { return m.ChapterTypeLock },
	"age_rating":       func(m *metadata.ChapterMetadata) bool { return m.AgeRatingLock },
	"web":              func(m *metadata.ChapterMetadata) bool { return m.WebLock },
	"characters":       func(m *metadata.ChapterMetadata) bool { return m.CharactersLock },
	"teams":            func(m *metadata.ChapterMetadata) bool { return m.TeamsLock },
	"locations":        func(m *metadata.ChapterMetadata) bool { return m.LocationsLock },
	"scanlation_group": func(m *metadata.ChapterMetadata) bool { return m.ScanlationGroupLock },
	"story_arc":        func(m *metadata.ChapterMetadata) bool { return m.StoryArcLock },
	"story_arc_number": func(m *metadata.ChapterMetadata) bool { return m.StoryArcNumberLock },
}

var validChapterLockFieldNames = map[string]bool{
	"title_lock":            true,
	"number_lock":           true,
	"sort_number_lock":      true,
	"volume_lock":           true,
	"summary_lock":          true,
	"notes_lock":            true,
	"release_date_lock":     true,
	"language_lock":         true,
	"chapter_type_lock":     true,
	"age_rating_lock":       true,
	"web_lock":              true,
	"characters_lock":       true,
	"teams_lock":            true,
	"locations_lock":        true,
	"scanlation_group_lock": true,
	"story_arc_lock":        true,
	"story_arc_number_lock": true,
}

func (s *Server) handleGetChapterMetadata(w http.ResponseWriter, r *http.Request) {
	chapterID, err := strconv.ParseInt(chi.URLParam(r, "chapterID"), 10, 64)
	if err != nil || chapterID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	exists, err := s.chapterExists(chapterID)
	if err != nil {
		log.Printf("chapterExists(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to check chapter existence")
		return
	}
	if !exists {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	meta, err := s.chapterMetadataStore().GetChapterMetadata(chapterID)
	if err != nil {
		log.Printf("GetChapterMetadata(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata")
		return
	}

	RespondWithJSON(w, http.StatusOK, getChapterMetadataResponse{
		Metadata: buildChapterMetadataFieldsResponse(meta),
	})
}

func (s *Server) handleEditChapterMetadata(w http.ResponseWriter, r *http.Request) {
	chapterID, err := strconv.ParseInt(chi.URLParam(r, "chapterID"), 10, 64)
	if err != nil || chapterID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	exists, err := s.chapterExists(chapterID)
	if err != nil {
		log.Printf("chapterExists(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to check chapter existence")
		return
	}
	if !exists {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updateMeta := &metadata.ChapterMetadata{}
	var fieldsToAutoLock []string

	for key, raw := range body {
		isNull := string(raw) == "null"

		switch key {
		case "title", "number", "volume", "summary", "notes",
			"release_date", "language", "age_rating", "web",
			"characters", "teams", "locations", "scanlation_group",
			"story_arc", "story_arc_number":
			if isNull {
				empty := ""
				setChapterStringField(updateMeta, key, &empty)
			} else {
				var v string
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid value for %s: expected string", key))
					return
				}
				setChapterStringField(updateMeta, key, &v)
			}
			fieldsToAutoLock = append(fieldsToAutoLock, key)

		case "chapter_type":
			if isNull {
				empty := ""
				updateMeta.ChapterType = &empty
			} else {
				var v string
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for chapter_type: expected string")
					return
				}
				if !validChapterTypes[v] {
					RespondWithError(w, http.StatusBadRequest, "invalid chapter_type: must be one of Regular, Special, Bonus, Omake, One-Shot, Annual, Omnibus")
					return
				}
				updateMeta.ChapterType = &v
			}
			fieldsToAutoLock = append(fieldsToAutoLock, key)

		case "sort_number":
			if isNull {
				zero := 0.0
				updateMeta.SortNumber = &zero
			} else {
				var v float64
				if err := json.Unmarshal(raw, &v); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for sort_number: expected number")
					return
				}
				updateMeta.SortNumber = &v
			}
			fieldsToAutoLock = append(fieldsToAutoLock, key)

		case "authors":
			if isNull {
				updateMeta.Authors = []metadata.ChapterMetadataAuthor{}
			} else {
				var authors []metadata.ChapterMetadataAuthor
				if err := json.Unmarshal(raw, &authors); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for authors: expected array of {name, role}")
					return
				}
				updateMeta.Authors = authors
			}

		case "genres":
			if isNull {
				updateMeta.Genres = []string{}
			} else {
				var genres []string
				if err := json.Unmarshal(raw, &genres); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for genres: expected array of strings")
					return
				}
				updateMeta.Genres = genres
			}

		case "tags":
			if isNull {
				updateMeta.Tags = []string{}
			} else {
				var tags []string
				if err := json.Unmarshal(raw, &tags); err != nil {
					RespondWithError(w, http.StatusBadRequest, "invalid value for tags: expected array of strings")
					return
				}
				updateMeta.Tags = tags
			}

		default:
			// Unknown fields ignored.
		}
	}

	cms := s.chapterMetadataStore()

	existingMeta, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		log.Printf("GetChapterMetadata(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata")
		return
	}

	if existingMeta != nil {
		var lockedFields []string
		for _, field := range fieldsToAutoLock {
			if lockCheck, ok := chapterMetadataFieldToLock[field]; ok {
				if lockCheck(existingMeta) {
					lockedFields = append(lockedFields, field)
				}
			}
		}
		if len(lockedFields) > 0 {
			RespondWithJSON(w, http.StatusConflict, map[string]interface{}{
				"error":         "Some fields are locked and cannot be edited",
				"locked_fields": lockedFields,
			})
			return
		}
	}

	if err := cms.UpsertChapterMetadata(chapterID, updateMeta); err != nil {
		log.Printf("UpsertChapterMetadata(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update chapter metadata")
		return
	}

	if len(fieldsToAutoLock) > 0 {
		currentMeta, err := cms.GetChapterMetadata(chapterID)
		if err != nil {
			log.Printf("GetChapterMetadata(%d) after upsert: %v", chapterID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata after update")
			return
		}
		if currentMeta != nil {
			locks := &metadata.ChapterMetadataLocks{
				TitleLock:           currentMeta.TitleLock,
				NumberLock:          currentMeta.NumberLock,
				SortNumberLock:      currentMeta.SortNumberLock,
				VolumeLock:          currentMeta.VolumeLock,
				SummaryLock:         currentMeta.SummaryLock,
				NotesLock:           currentMeta.NotesLock,
				ReleaseDateLock:     currentMeta.ReleaseDateLock,
				LanguageLock:        currentMeta.LanguageLock,
				ChapterTypeLock:     currentMeta.ChapterTypeLock,
				AgeRatingLock:       currentMeta.AgeRatingLock,
				WebLock:             currentMeta.WebLock,
				CharactersLock:      currentMeta.CharactersLock,
				TeamsLock:           currentMeta.TeamsLock,
				LocationsLock:       currentMeta.LocationsLock,
				ScanlationGroupLock: currentMeta.ScanlationGroupLock,
				StoryArcLock:        currentMeta.StoryArcLock,
				StoryArcNumberLock:  currentMeta.StoryArcNumberLock,
			}
			for _, field := range fieldsToAutoLock {
				setChapterLockField(locks, field, true)
			}
			if err := cms.UpdateChapterMetadataLocks(chapterID, locks); err != nil {
				log.Printf("UpdateChapterMetadataLocks(%d) auto-lock: %v", chapterID, err)
			}
		}
	}

	finalMeta, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		log.Printf("GetChapterMetadata(%d) after edit: %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata after update")
		return
	}

	RespondWithJSON(w, http.StatusOK, getChapterMetadataResponse{
		Metadata: buildChapterMetadataFieldsResponse(finalMeta),
	})
}

func (s *Server) handleGetChapterMetadataLocks(w http.ResponseWriter, r *http.Request) {
	chapterID, err := strconv.ParseInt(chi.URLParam(r, "chapterID"), 10, 64)
	if err != nil || chapterID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	exists, err := s.chapterExists(chapterID)
	if err != nil {
		log.Printf("chapterExists(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to check chapter existence")
		return
	}
	if !exists {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	meta, err := s.chapterMetadataStore().GetChapterMetadata(chapterID)
	if err != nil {
		log.Printf("GetChapterMetadata(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata")
		return
	}

	RespondWithJSON(w, http.StatusOK, buildChapterMetadataLocksFromMeta(meta))
}

func (s *Server) handleEditChapterMetadataLocks(w http.ResponseWriter, r *http.Request) {
	chapterID, err := strconv.ParseInt(chi.URLParam(r, "chapterID"), 10, 64)
	if err != nil || chapterID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	exists, err := s.chapterExists(chapterID)
	if err != nil {
		log.Printf("chapterExists(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to check chapter existence")
		return
	}
	if !exists {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	lockUpdates := make(map[string]bool)
	for key, raw := range body {
		if !validChapterLockFieldNames[key] {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("unknown lock field: %s", key))
			return
		}
		var v bool
		if err := json.Unmarshal(raw, &v); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid value for %s: expected boolean", key))
			return
		}
		lockUpdates[key] = v
	}

	cms := s.chapterMetadataStore()

	currentMeta, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		log.Printf("GetChapterMetadata(%d): %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata")
		return
	}

	if currentMeta == nil {
		if err := cms.UpsertChapterMetadata(chapterID, &metadata.ChapterMetadata{}); err != nil {
			log.Printf("UpsertChapterMetadata(%d): %v", chapterID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to create chapter metadata record")
			return
		}
		currentMeta, err = cms.GetChapterMetadata(chapterID)
		if err != nil {
			log.Printf("GetChapterMetadata(%d) after create: %v", chapterID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata")
			return
		}
	}

	if len(lockUpdates) > 0 {
		locks := &metadata.ChapterMetadataLocks{
			TitleLock:           currentMeta.TitleLock,
			NumberLock:          currentMeta.NumberLock,
			SortNumberLock:      currentMeta.SortNumberLock,
			VolumeLock:          currentMeta.VolumeLock,
			SummaryLock:         currentMeta.SummaryLock,
			NotesLock:           currentMeta.NotesLock,
			ReleaseDateLock:     currentMeta.ReleaseDateLock,
			LanguageLock:        currentMeta.LanguageLock,
			ChapterTypeLock:     currentMeta.ChapterTypeLock,
			AgeRatingLock:       currentMeta.AgeRatingLock,
			WebLock:             currentMeta.WebLock,
			CharactersLock:      currentMeta.CharactersLock,
			TeamsLock:           currentMeta.TeamsLock,
			LocationsLock:       currentMeta.LocationsLock,
			ScanlationGroupLock: currentMeta.ScanlationGroupLock,
			StoryArcLock:        currentMeta.StoryArcLock,
			StoryArcNumberLock:  currentMeta.StoryArcNumberLock,
		}
		for field, val := range lockUpdates {
			setChapterLockFieldByDBName(locks, field, val)
		}

		if err := cms.UpdateChapterMetadataLocks(chapterID, locks); err != nil {
			if errors.Is(err, store.ErrChapterMetadataNotFound) {
				RespondWithError(w, http.StatusNotFound, "no chapter metadata found")
				return
			}
			log.Printf("UpdateChapterMetadataLocks(%d): %v", chapterID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to update locks")
			return
		}
	}

	finalMeta, err := cms.GetChapterMetadata(chapterID)
	if err != nil {
		log.Printf("GetChapterMetadata(%d) after lock update: %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chapter metadata")
		return
	}

	RespondWithJSON(w, http.StatusOK, buildChapterMetadataLocksFromMeta(finalMeta))
}

func setChapterStringField(m *metadata.ChapterMetadata, field string, val *string) {
	switch field {
	case "title":
		m.Title = val
	case "number":
		m.Number = val
	case "volume":
		m.Volume = val
	case "summary":
		m.Summary = val
	case "notes":
		m.Notes = val
	case "release_date":
		m.ReleaseDate = val
	case "language":
		m.Language = val
	case "age_rating":
		m.AgeRating = val
	case "web":
		m.Web = val
	case "characters":
		m.Characters = val
	case "teams":
		m.Teams = val
	case "locations":
		m.Locations = val
	case "scanlation_group":
		m.ScanlationGroup = val
	case "story_arc":
		m.StoryArc = val
	case "story_arc_number":
		m.StoryArcNumber = val
	}
}

func setChapterLockField(locks *metadata.ChapterMetadataLocks, field string, val bool) {
	switch field {
	case "title":
		locks.TitleLock = val
	case "number":
		locks.NumberLock = val
	case "sort_number":
		locks.SortNumberLock = val
	case "volume":
		locks.VolumeLock = val
	case "summary":
		locks.SummaryLock = val
	case "notes":
		locks.NotesLock = val
	case "release_date":
		locks.ReleaseDateLock = val
	case "language":
		locks.LanguageLock = val
	case "chapter_type":
		locks.ChapterTypeLock = val
	case "age_rating":
		locks.AgeRatingLock = val
	case "web":
		locks.WebLock = val
	case "characters":
		locks.CharactersLock = val
	case "teams":
		locks.TeamsLock = val
	case "locations":
		locks.LocationsLock = val
	case "scanlation_group":
		locks.ScanlationGroupLock = val
	case "story_arc":
		locks.StoryArcLock = val
	case "story_arc_number":
		locks.StoryArcNumberLock = val
	}
}

func setChapterLockFieldByDBName(locks *metadata.ChapterMetadataLocks, dbField string, val bool) {
	switch dbField {
	case "title_lock":
		locks.TitleLock = val
	case "number_lock":
		locks.NumberLock = val
	case "sort_number_lock":
		locks.SortNumberLock = val
	case "volume_lock":
		locks.VolumeLock = val
	case "summary_lock":
		locks.SummaryLock = val
	case "notes_lock":
		locks.NotesLock = val
	case "release_date_lock":
		locks.ReleaseDateLock = val
	case "language_lock":
		locks.LanguageLock = val
	case "chapter_type_lock":
		locks.ChapterTypeLock = val
	case "age_rating_lock":
		locks.AgeRatingLock = val
	case "web_lock":
		locks.WebLock = val
	case "characters_lock":
		locks.CharactersLock = val
	case "teams_lock":
		locks.TeamsLock = val
	case "locations_lock":
		locks.LocationsLock = val
	case "scanlation_group_lock":
		locks.ScanlationGroupLock = val
	case "story_arc_lock":
		locks.StoryArcLock = val
	case "story_arc_number_lock":
		locks.StoryArcNumberLock = val
	}
}
