package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/metadata"
)

var (
	// ErrChapterMetadataNotFound is returned when no chapter metadata exists for a chapter.
	ErrChapterMetadataNotFound = errors.New("chapter metadata not found")

	// ErrBulkUpdateLimitExceeded is returned when bulk update exceeds the maximum chapter count.
	ErrBulkUpdateLimitExceeded = errors.New("bulk update exceeds maximum of 500 chapters")
)

const maxBulkUpdateChapters = 500

// ChapterMetadataStore handles chapter metadata database operations.
type ChapterMetadataStore struct {
	db *sql.DB
}

// NewChapterMetadataStore creates a new ChapterMetadataStore.
func NewChapterMetadataStore(db *sql.DB) *ChapterMetadataStore {
	return &ChapterMetadataStore{db: db}
}

// UpsertChapterMetadata creates or updates chapter metadata for a chapter.
// Nil scalar fields are skipped (existing values preserved). Locked fields
// are never overwritten. For collections (Authors, Genres, Tags), nil keeps
// existing rows; non-nil (including empty) replaces all rows.
func (s *ChapterMetadataStore) UpsertChapterMetadata(chapterID int64, meta *metadata.ChapterMetadata) error {
	if meta == nil {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := s.upsertInTx(tx, chapterID, meta); err != nil {
		return err
	}

	return tx.Commit()
}

// upsertInTx performs the chapter metadata upsert within a transaction.
// Returns true if any data was actually changed.
func (s *ChapterMetadataStore) upsertInTx(tx *sql.Tx, chapterID int64, meta *metadata.ChapterMetadata) (bool, error) {
	if meta == nil {
		return false, nil
	}

	changed := false

	// 1. Ensure a metadata row exists for this chapter.
	_, err := tx.Exec("INSERT OR IGNORE INTO chapter_metadata (chapter_id) VALUES (?)", chapterID)
	if err != nil {
		return false, err
	}

	// 2. Read metadataID and all lock states.
	var metadataID int64
	var titleLock, numberLock, sortNumberLock, volumeLock, summaryLock bool
	var notesLock, releaseDateLock, languageLock, chapterTypeLock bool
	var ageRatingLock, webLock, charactersLock, teamsLock, locationsLock bool
	var scanlationGroupLock, storyArcLock, storyArcNumberLock bool

	err = tx.QueryRow(`SELECT id,
		title_lock, number_lock, sort_number_lock, volume_lock, summary_lock,
		notes_lock, release_date_lock, language_lock, chapter_type_lock,
		age_rating_lock, web_lock, characters_lock, teams_lock, locations_lock,
		scanlation_group_lock, story_arc_lock, story_arc_number_lock
		FROM chapter_metadata WHERE chapter_id = ?`, chapterID).Scan(
		&metadataID,
		&titleLock, &numberLock, &sortNumberLock, &volumeLock, &summaryLock,
		&notesLock, &releaseDateLock, &languageLock, &chapterTypeLock,
		&ageRatingLock, &webLock, &charactersLock, &teamsLock, &locationsLock,
		&scanlationGroupLock, &storyArcLock, &storyArcNumberLock,
	)
	if err != nil {
		return false, err
	}

	// 3. Build dynamic UPDATE for non-nil, non-locked scalar fields.
	var setClauses []string
	var args []interface{}

	if meta.Title != nil && !titleLock {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *meta.Title)
	}
	if meta.Number != nil && !numberLock {
		setClauses = append(setClauses, "number = ?")
		args = append(args, *meta.Number)
	}
	if meta.SortNumber != nil && !sortNumberLock {
		setClauses = append(setClauses, "sort_number = ?")
		args = append(args, *meta.SortNumber)
	}
	if meta.Volume != nil && !volumeLock {
		setClauses = append(setClauses, "volume = ?")
		args = append(args, *meta.Volume)
	}
	if meta.Summary != nil && !summaryLock {
		setClauses = append(setClauses, "summary = ?")
		args = append(args, *meta.Summary)
	}
	if meta.Notes != nil && !notesLock {
		setClauses = append(setClauses, "notes = ?")
		args = append(args, *meta.Notes)
	}
	if meta.ReleaseDate != nil && !releaseDateLock {
		setClauses = append(setClauses, "release_date = ?")
		args = append(args, *meta.ReleaseDate)
	}
	if meta.Language != nil && !languageLock {
		setClauses = append(setClauses, "language = ?")
		args = append(args, *meta.Language)
	}
	if meta.ChapterType != nil && !chapterTypeLock {
		setClauses = append(setClauses, "chapter_type = ?")
		args = append(args, *meta.ChapterType)
	}
	if meta.AgeRating != nil && !ageRatingLock {
		setClauses = append(setClauses, "age_rating = ?")
		args = append(args, *meta.AgeRating)
	}
	if meta.Web != nil && !webLock {
		setClauses = append(setClauses, "web = ?")
		args = append(args, *meta.Web)
	}
	if meta.Characters != nil && !charactersLock {
		setClauses = append(setClauses, "characters = ?")
		args = append(args, *meta.Characters)
	}
	if meta.Teams != nil && !teamsLock {
		setClauses = append(setClauses, "teams = ?")
		args = append(args, *meta.Teams)
	}
	if meta.Locations != nil && !locationsLock {
		setClauses = append(setClauses, "locations = ?")
		args = append(args, *meta.Locations)
	}
	if meta.ScanlationGroup != nil && !scanlationGroupLock {
		setClauses = append(setClauses, "scanlation_group = ?")
		args = append(args, *meta.ScanlationGroup)
	}
	if meta.StoryArc != nil && !storyArcLock {
		setClauses = append(setClauses, "story_arc = ?")
		args = append(args, *meta.StoryArc)
	}
	if meta.StoryArcNumber != nil && !storyArcNumberLock {
		setClauses = append(setClauses, "story_arc_number = ?")
		args = append(args, *meta.StoryArcNumber)
	}

	if len(setClauses) > 0 {
		query := fmt.Sprintf("UPDATE chapter_metadata SET %s WHERE id = ?", strings.Join(setClauses, ", "))
		args = append(args, metadataID)
		if _, err = tx.Exec(query, args...); err != nil {
			return false, err
		}
		changed = true
	}

	// 4. Child tables: nil = keep existing, non-nil = replace.
	// Chapter metadata child tables have no lock columns.
	if meta.Authors != nil {
		if _, err = tx.Exec("DELETE FROM chapter_metadata_authors WHERE chapter_metadata_id = ?", metadataID); err != nil {
			return false, err
		}
		for _, a := range meta.Authors {
			if _, err = tx.Exec("INSERT INTO chapter_metadata_authors (chapter_metadata_id, name, role) VALUES (?, ?, ?)",
				metadataID, a.Name, a.Role); err != nil {
				return false, err
			}
		}
		changed = true
	}

	if meta.Genres != nil {
		if _, err = tx.Exec("DELETE FROM chapter_metadata_genres WHERE chapter_metadata_id = ?", metadataID); err != nil {
			return false, err
		}
		for _, g := range meta.Genres {
			if _, err = tx.Exec("INSERT INTO chapter_metadata_genres (chapter_metadata_id, name) VALUES (?, ?)",
				metadataID, g); err != nil {
				return false, err
			}
		}
		changed = true
	}

	if meta.Tags != nil {
		if _, err = tx.Exec("DELETE FROM chapter_metadata_tags WHERE chapter_metadata_id = ?", metadataID); err != nil {
			return false, err
		}
		for _, t := range meta.Tags {
			if _, err = tx.Exec("INSERT INTO chapter_metadata_tags (chapter_metadata_id, name) VALUES (?, ?)",
				metadataID, t); err != nil {
				return false, err
			}
		}
		changed = true
	}

	// 5. Always update timestamp.
	if _, err = tx.Exec("UPDATE chapter_metadata SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", metadataID); err != nil {
		return false, err
	}

	return changed, nil
}

// GetChapterMetadata returns the full metadata for a chapter, including all
// child table data. Returns nil, nil if no metadata exists (not an error).
func (s *ChapterMetadataStore) GetChapterMetadata(chapterID int64) (*metadata.ChapterMetadata, error) {
	var m metadata.ChapterMetadata
	var title, number, volume, summary, notes sql.NullString
	var releaseDate, language, chapterType, ageRating, web sql.NullString
	var characters, teams, locations, scanlationGroup sql.NullString
	var storyArc, storyArcNumber sql.NullString
	var sortNumber sql.NullFloat64
	var createdAt, updatedAt string

	err := s.db.QueryRow(`SELECT id, chapter_id,
		title, number, sort_number, volume, summary, notes,
		release_date, language, chapter_type, age_rating, web,
		characters, teams, locations, scanlation_group,
		story_arc, story_arc_number,
		title_lock, number_lock, sort_number_lock, volume_lock,
		summary_lock, notes_lock, release_date_lock, language_lock,
		chapter_type_lock, age_rating_lock, web_lock,
		characters_lock, teams_lock, locations_lock,
		scanlation_group_lock, story_arc_lock, story_arc_number_lock,
		created_at, updated_at
		FROM chapter_metadata WHERE chapter_id = ?`, chapterID).Scan(
		&m.ID, &m.ChapterID,
		&title, &number, &sortNumber, &volume, &summary, &notes,
		&releaseDate, &language, &chapterType, &ageRating, &web,
		&characters, &teams, &locations, &scanlationGroup,
		&storyArc, &storyArcNumber,
		&m.TitleLock, &m.NumberLock, &m.SortNumberLock, &m.VolumeLock,
		&m.SummaryLock, &m.NotesLock, &m.ReleaseDateLock, &m.LanguageLock,
		&m.ChapterTypeLock, &m.AgeRatingLock, &m.WebLock,
		&m.CharactersLock, &m.TeamsLock, &m.LocationsLock,
		&m.ScanlationGroupLock, &m.StoryArcLock, &m.StoryArcNumberLock,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Convert sql.Null* to pointers.
	if title.Valid {
		m.Title = &title.String
	}
	if number.Valid {
		m.Number = &number.String
	}
	if sortNumber.Valid {
		m.SortNumber = &sortNumber.Float64
	}
	if volume.Valid {
		m.Volume = &volume.String
	}
	if summary.Valid {
		m.Summary = &summary.String
	}
	if notes.Valid {
		m.Notes = &notes.String
	}
	if releaseDate.Valid {
		m.ReleaseDate = &releaseDate.String
	}
	if language.Valid {
		m.Language = &language.String
	}
	if chapterType.Valid {
		m.ChapterType = &chapterType.String
	}
	if ageRating.Valid {
		m.AgeRating = &ageRating.String
	}
	if web.Valid {
		m.Web = &web.String
	}
	if characters.Valid {
		m.Characters = &characters.String
	}
	if teams.Valid {
		m.Teams = &teams.String
	}
	if locations.Valid {
		m.Locations = &locations.String
	}
	if scanlationGroup.Valid {
		m.ScanlationGroup = &scanlationGroup.String
	}
	if storyArc.Valid {
		m.StoryArc = &storyArc.String
	}
	if storyArcNumber.Valid {
		m.StoryArcNumber = &storyArcNumber.String
	}

	m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	m.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	// Initialize and load child slices.
	m.Authors = []metadata.ChapterMetadataAuthor{}
	m.Genres = []string{}
	m.Tags = []string{}

	if err := s.loadChapterMetadataAuthors(&m); err != nil {
		return nil, err
	}
	if err := s.loadChapterMetadataGenres(&m); err != nil {
		return nil, err
	}
	if err := s.loadChapterMetadataTags(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// GetChapterMetadataByFolder returns all chapter metadata for chapters
// belonging to a folder, joined via chapters.folder_id.
func (s *ChapterMetadataStore) GetChapterMetadataByFolder(folderID int64) ([]*metadata.ChapterMetadata, error) {
	rows, err := s.db.Query(`SELECT cm.chapter_id FROM chapter_metadata cm
		JOIN chapters c ON cm.chapter_id = c.id
		WHERE c.folder_id = ?
		ORDER BY cm.chapter_id`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapterIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		chapterIDs = append(chapterIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*metadata.ChapterMetadata, 0, len(chapterIDs))
	for _, cid := range chapterIDs {
		m, err := s.GetChapterMetadata(cid)
		if err != nil {
			return nil, err
		}
		if m != nil {
			result = append(result, m)
		}
	}

	return result, nil
}

// BulkUpdateChapterMetadata updates metadata for multiple chapters in a single
// transaction. Returns lists of updated and skipped chapter IDs. A chapter is
// "skipped" when all provided fields were locked or nil. Returns an error if
// the number of chapters exceeds 500.
func (s *ChapterMetadataStore) BulkUpdateChapterMetadata(updates map[int64]*metadata.ChapterMetadata) ([]int64, []int64, error) {
	if len(updates) > maxBulkUpdateChapters {
		return nil, nil, ErrBulkUpdateLimitExceeded
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	var updated, skipped []int64
	for chapterID, meta := range updates {
		changed, err := s.upsertInTx(tx, chapterID, meta)
		if err != nil {
			return nil, nil, err
		}
		if changed {
			updated = append(updated, chapterID)
		} else {
			skipped = append(skipped, chapterID)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	return updated, skipped, nil
}

// UpdateChapterMetadataLocks updates all lock columns for a chapter's metadata.
// Returns ErrChapterMetadataNotFound if no metadata row exists for the chapter.
func (s *ChapterMetadataStore) UpdateChapterMetadataLocks(chapterID int64, locks *metadata.ChapterMetadataLocks) error {
	if locks == nil {
		return nil
	}

	result, err := s.db.Exec(`UPDATE chapter_metadata SET
		title_lock = ?, number_lock = ?, sort_number_lock = ?, volume_lock = ?,
		summary_lock = ?, notes_lock = ?, release_date_lock = ?, language_lock = ?,
		chapter_type_lock = ?, age_rating_lock = ?, web_lock = ?,
		characters_lock = ?, teams_lock = ?, locations_lock = ?,
		scanlation_group_lock = ?, story_arc_lock = ?, story_arc_number_lock = ?,
		updated_at = CURRENT_TIMESTAMP
		WHERE chapter_id = ?`,
		locks.TitleLock, locks.NumberLock, locks.SortNumberLock, locks.VolumeLock,
		locks.SummaryLock, locks.NotesLock, locks.ReleaseDateLock, locks.LanguageLock,
		locks.ChapterTypeLock, locks.AgeRatingLock, locks.WebLock,
		locks.CharactersLock, locks.TeamsLock, locks.LocationsLock,
		locks.ScanlationGroupLock, locks.StoryArcLock, locks.StoryArcNumberLock,
		chapterID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrChapterMetadataNotFound
	}
	return nil
}

// DeleteChapterMetadata deletes the chapter metadata row for the given chapter ID.
// The child table rows (authors, genres, tags) are deleted via ON DELETE CASCADE.
func (s *ChapterMetadataStore) DeleteChapterMetadata(chapterID int64) error {
	result, err := s.db.Exec(`DELETE FROM chapter_metadata WHERE chapter_id = ?`, chapterID)
	if err != nil {
		return fmt.Errorf("delete chapter metadata: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete chapter metadata rows affected: %w", err)
	}
	if rows == 0 {
		return ErrChapterMetadataNotFound
	}
	return nil
}

func (s *ChapterMetadataStore) loadChapterMetadataAuthors(m *metadata.ChapterMetadata) error {
	rows, err := s.db.Query("SELECT id, name, role FROM chapter_metadata_authors WHERE chapter_metadata_id = ?", m.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a metadata.ChapterMetadataAuthor
		if err := rows.Scan(&a.ID, &a.Name, &a.Role); err != nil {
			return err
		}
		m.Authors = append(m.Authors, a)
	}
	return rows.Err()
}

func (s *ChapterMetadataStore) loadChapterMetadataGenres(m *metadata.ChapterMetadata) error {
	rows, err := s.db.Query("SELECT name FROM chapter_metadata_genres WHERE chapter_metadata_id = ?", m.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return err
		}
		m.Genres = append(m.Genres, g)
	}
	return rows.Err()
}

func (s *ChapterMetadataStore) loadChapterMetadataTags(m *metadata.ChapterMetadata) error {
	rows, err := s.db.Query("SELECT name FROM chapter_metadata_tags WHERE chapter_metadata_id = ?", m.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return err
		}
		m.Tags = append(m.Tags, tag)
	}
	return rows.Err()
}
