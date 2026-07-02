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
	// ErrMetadataNotFound is returned when no series metadata exists for a folder.
	ErrMetadataNotFound = errors.New("series metadata not found")

	// ErrProviderLinkNotFound is returned when no provider link exists for a folder.
	ErrProviderLinkNotFound = errors.New("provider link not found")
)

// validLockFields is the known set of 16 lock column names.
var validLockFields = map[string]bool{
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

// validScalarFields maps scalar column names to their corresponding lock column.
var validScalarFields = map[string]string{
	"status":            "status_lock",
	"title":             "title_lock",
	"summary":           "summary_lock",
	"publisher":         "publisher_lock",
	"reading_direction": "reading_direction_lock",
	"age_rating":        "age_rating_lock",
	"language":          "language_lock",
	"total_book_count":  "total_book_count_lock",
	"community_score":   "community_score_lock",
	"release_year":      "release_date_lock",
	"release_month":     "release_date_lock",
	"release_day":       "release_date_lock",
	"thumbnail_url":     "thumbnail_url_lock",
}

// SeriesMetadataRow represents a series metadata record from the database,
// including all scalar fields, lock states, and child table data.
type SeriesMetadataRow struct {
	ID               int64
	FolderID         int64
	Status           sql.NullString
	Title            sql.NullString
	Summary          sql.NullString
	Publisher        sql.NullString
	ReadingDirection sql.NullString
	AgeRating        sql.NullInt64
	Language         sql.NullString
	TotalBookCount   sql.NullInt64
	CommunityScore   sql.NullFloat64
	ReleaseYear      sql.NullInt64
	ReleaseMonth     sql.NullInt64
	ReleaseDay       sql.NullInt64
	ThumbnailURL     sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time

	StatusLock           bool
	TitleLock            bool
	SummaryLock          bool
	PublisherLock        bool
	ReadingDirectionLock bool
	AgeRatingLock        bool
	LanguageLock         bool
	TotalBookCountLock   bool
	CommunityScoreLock   bool
	ReleaseDateLock      bool
	ThumbnailURLLock     bool
	GenresLock           bool
	TagsLock             bool
	AuthorsLock          bool
	LinksLock            bool
	TitlesLock           bool

	Genres  []string
	Tags    []string
	Authors []metadata.Author
	Links   []metadata.WebLink
	Titles  []metadata.SeriesTitle
}

// ProviderLink represents a link between a folder and a metadata provider.
type ProviderLink struct {
	FolderID     int64
	ProviderName string
	ProviderID   string
	CreatedAt    time.Time
}

// UpsertSeriesMetadata creates or updates series metadata for a folder.
// Nil scalar fields are skipped (existing values preserved). Locked fields
// are never overwritten. For collections, nil slices keep existing rows;
// non-nil slices (including empty) replace all rows if the lock is off.
func (s *Store) UpsertSeriesMetadata(folderID int64, meta *metadata.SeriesMetadata) error {
	if meta == nil {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure a metadata row exists for this folder.
	_, err = tx.Exec("INSERT OR IGNORE INTO series_metadata (folder_id) VALUES (?)", folderID)
	if err != nil {
		return err
	}

	// Read metadata ID and all lock states.
	var metadataID int64
	var statusLock, titleLock, summaryLock, publisherLock, rdLock bool
	var ageLock, langLock, bookCountLock, scoreLock bool
	var dateLock, thumbLock bool
	var genresLock, tagsLock, authorsLock, linksLock, titlesLock bool

	err = tx.QueryRow(`SELECT id,
		status_lock, title_lock, summary_lock, publisher_lock, reading_direction_lock,
		age_rating_lock, language_lock, total_book_count_lock, community_score_lock,
		release_date_lock, thumbnail_url_lock,
		genres_lock, tags_lock, authors_lock, links_lock, titles_lock
		FROM series_metadata WHERE folder_id = ?`, folderID).Scan(
		&metadataID,
		&statusLock, &titleLock, &summaryLock, &publisherLock, &rdLock,
		&ageLock, &langLock, &bookCountLock, &scoreLock,
		&dateLock, &thumbLock,
		&genresLock, &tagsLock, &authorsLock, &linksLock, &titlesLock,
	)
	if err != nil {
		return err
	}

	// Build dynamic UPDATE for non-nil, non-locked scalar fields.
	var setClauses []string
	var args []interface{}

	if meta.Status != nil && !statusLock {
		setClauses = append(setClauses, "status = ?")
		args = append(args, string(*meta.Status))
	}
	if meta.Title != nil && !titleLock {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *meta.Title)
	}
	if meta.Summary != nil && !summaryLock {
		setClauses = append(setClauses, "summary = ?")
		args = append(args, *meta.Summary)
	}
	if meta.Publisher != nil && !publisherLock {
		setClauses = append(setClauses, "publisher = ?")
		args = append(args, *meta.Publisher)
	}
	if meta.ReadingDirection != nil && !rdLock {
		setClauses = append(setClauses, "reading_direction = ?")
		args = append(args, string(*meta.ReadingDirection))
	}
	if meta.AgeRating != nil && !ageLock {
		setClauses = append(setClauses, "age_rating = ?")
		args = append(args, *meta.AgeRating)
	}
	if meta.Language != nil && !langLock {
		setClauses = append(setClauses, "language = ?")
		args = append(args, *meta.Language)
	}
	if meta.TotalBookCount != nil && !bookCountLock {
		setClauses = append(setClauses, "total_book_count = ?")
		args = append(args, *meta.TotalBookCount)
	}
	if meta.CommunityScore != nil && !scoreLock {
		setClauses = append(setClauses, "community_score = ?")
		args = append(args, *meta.CommunityScore)
	}
	if meta.ReleaseYear != nil && !dateLock {
		setClauses = append(setClauses, "release_year = ?")
		args = append(args, *meta.ReleaseYear)
	}
	if meta.ReleaseMonth != nil && !dateLock {
		setClauses = append(setClauses, "release_month = ?")
		args = append(args, *meta.ReleaseMonth)
	}
	if meta.ReleaseDay != nil && !dateLock {
		setClauses = append(setClauses, "release_day = ?")
		args = append(args, *meta.ReleaseDay)
	}
	if meta.ThumbnailURL != nil && !thumbLock {
		setClauses = append(setClauses, "thumbnail_url = ?")
		args = append(args, *meta.ThumbnailURL)
	}

	if len(setClauses) > 0 {
		query := fmt.Sprintf("UPDATE series_metadata SET %s WHERE id = ?", strings.Join(setClauses, ", "))
		args = append(args, metadataID)
		if _, err = tx.Exec(query, args...); err != nil {
			return err
		}
	}

	// Handle collection fields: nil = keep existing, non-nil = replace.
	if meta.Genres != nil && !genresLock {
		if _, err = tx.Exec("DELETE FROM series_metadata_genres WHERE metadata_id = ?", metadataID); err != nil {
			return err
		}
		for _, g := range meta.Genres {
			if _, err = tx.Exec("INSERT INTO series_metadata_genres (metadata_id, genre) VALUES (?, ?)", metadataID, g); err != nil {
				return err
			}
		}
	}

	if meta.Tags != nil && !tagsLock {
		if _, err = tx.Exec("DELETE FROM series_metadata_tags WHERE metadata_id = ?", metadataID); err != nil {
			return err
		}
		for _, t := range meta.Tags {
			if _, err = tx.Exec("INSERT INTO series_metadata_tags (metadata_id, tag) VALUES (?, ?)", metadataID, t); err != nil {
				return err
			}
		}
	}

	if meta.Authors != nil && !authorsLock {
		if _, err = tx.Exec("DELETE FROM series_metadata_authors WHERE metadata_id = ?", metadataID); err != nil {
			return err
		}
		for _, a := range meta.Authors {
			if _, err = tx.Exec("INSERT INTO series_metadata_authors (metadata_id, name, role) VALUES (?, ?, ?)",
				metadataID, a.Name, string(a.Role)); err != nil {
				return err
			}
		}
	}

	if meta.Links != nil && !linksLock {
		if _, err = tx.Exec("DELETE FROM series_metadata_links WHERE metadata_id = ?", metadataID); err != nil {
			return err
		}
		for _, l := range meta.Links {
			if _, err = tx.Exec("INSERT INTO series_metadata_links (metadata_id, label, url) VALUES (?, ?, ?)",
				metadataID, l.Label, l.URL); err != nil {
				return err
			}
		}
	}

	if meta.Titles != nil && !titlesLock {
		if _, err = tx.Exec("DELETE FROM series_metadata_titles WHERE metadata_id = ?", metadataID); err != nil {
			return err
		}
		for _, t := range meta.Titles {
			if _, err = tx.Exec("INSERT INTO series_metadata_titles (metadata_id, title, type, language) VALUES (?, ?, ?, ?)",
				metadataID, t.Title, t.Type, t.Language); err != nil {
				return err
			}
		}
	}

	// Always update timestamp.
	if _, err = tx.Exec("UPDATE series_metadata SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", metadataID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetSeriesMetadata returns the full metadata for a folder, including all
// child table data. Returns ErrMetadataNotFound if no metadata exists.
func (s *Store) GetSeriesMetadata(folderID int64) (*SeriesMetadataRow, error) {
	row := &SeriesMetadataRow{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(`SELECT id, folder_id,
		status, title, summary, publisher, reading_direction,
		age_rating, language, total_book_count, community_score,
		release_year, release_month, release_day, thumbnail_url,
		status_lock, title_lock, summary_lock, publisher_lock,
		reading_direction_lock, age_rating_lock, language_lock,
		total_book_count_lock, community_score_lock, release_date_lock,
		thumbnail_url_lock, genres_lock, tags_lock, authors_lock,
		links_lock, titles_lock,
		created_at, updated_at
		FROM series_metadata WHERE folder_id = ?`, folderID).Scan(
		&row.ID, &row.FolderID,
		&row.Status, &row.Title, &row.Summary, &row.Publisher, &row.ReadingDirection,
		&row.AgeRating, &row.Language, &row.TotalBookCount, &row.CommunityScore,
		&row.ReleaseYear, &row.ReleaseMonth, &row.ReleaseDay, &row.ThumbnailURL,
		&row.StatusLock, &row.TitleLock, &row.SummaryLock, &row.PublisherLock,
		&row.ReadingDirectionLock, &row.AgeRatingLock, &row.LanguageLock,
		&row.TotalBookCountLock, &row.CommunityScoreLock, &row.ReleaseDateLock,
		&row.ThumbnailURLLock, &row.GenresLock, &row.TagsLock, &row.AuthorsLock,
		&row.LinksLock, &row.TitlesLock,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMetadataNotFound
		}
		return nil, err
	}
	row.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	row.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	// Initialize child slices and load data.
	row.Genres = []string{}
	row.Tags = []string{}
	row.Authors = []metadata.Author{}
	row.Links = []metadata.WebLink{}
	row.Titles = []metadata.SeriesTitle{}

	if err := s.loadMetadataGenres(row); err != nil {
		return nil, err
	}
	if err := s.loadMetadataTags(row); err != nil {
		return nil, err
	}
	if err := s.loadMetadataAuthors(row); err != nil {
		return nil, err
	}
	if err := s.loadMetadataLinks(row); err != nil {
		return nil, err
	}
	if err := s.loadMetadataTitles(row); err != nil {
		return nil, err
	}

	return row, nil
}

func (s *Store) loadMetadataGenres(row *SeriesMetadataRow) error {
	rows, err := s.db.Query("SELECT genre FROM series_metadata_genres WHERE metadata_id = ?", row.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		row.Genres = append(row.Genres, v)
	}
	return rows.Err()
}

func (s *Store) loadMetadataTags(row *SeriesMetadataRow) error {
	rows, err := s.db.Query("SELECT tag FROM series_metadata_tags WHERE metadata_id = ?", row.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		row.Tags = append(row.Tags, v)
	}
	return rows.Err()
}

func (s *Store) loadMetadataAuthors(row *SeriesMetadataRow) error {
	rows, err := s.db.Query("SELECT name, role FROM series_metadata_authors WHERE metadata_id = ?", row.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name, role string
		if err := rows.Scan(&name, &role); err != nil {
			return err
		}
		row.Authors = append(row.Authors, metadata.Author{Name: name, Role: metadata.AuthorRole(role)})
	}
	return rows.Err()
}

func (s *Store) loadMetadataLinks(row *SeriesMetadataRow) error {
	rows, err := s.db.Query("SELECT label, url FROM series_metadata_links WHERE metadata_id = ?", row.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var l metadata.WebLink
		if err := rows.Scan(&l.Label, &l.URL); err != nil {
			return err
		}
		row.Links = append(row.Links, l)
	}
	return rows.Err()
}

func (s *Store) loadMetadataTitles(row *SeriesMetadataRow) error {
	rows, err := s.db.Query("SELECT title, type, language FROM series_metadata_titles WHERE metadata_id = ?", row.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t metadata.SeriesTitle
		var lang sql.NullString
		if err := rows.Scan(&t.Title, &t.Type, &lang); err != nil {
			return err
		}
		if lang.Valid {
			t.Language = lang.String
		}
		row.Titles = append(row.Titles, t)
	}
	return rows.Err()
}

// UpsertProviderLink creates or updates a provider link for a folder.
func (s *Store) UpsertProviderLink(folderID int64, providerName, providerID string) error {
	query := `INSERT INTO series_provider_link (folder_id, provider_name, provider_id)
	          VALUES (?, ?, ?)
	          ON CONFLICT(folder_id, provider_name) DO UPDATE SET
	            provider_id = excluded.provider_id`
	_, err := s.db.Exec(query, folderID, providerName, providerID)
	return err
}

// GetProviderLink returns the provider link for a folder,
// or ErrProviderLinkNotFound if none exists.
func (s *Store) GetProviderLink(folderID int64) (*ProviderLink, error) {
	link := &ProviderLink{}
	var createdAt string
	err := s.db.QueryRow(
		"SELECT folder_id, provider_name, provider_id, created_at FROM series_provider_link WHERE folder_id = ? LIMIT 1",
		folderID,
	).Scan(&link.FolderID, &link.ProviderName, &link.ProviderID, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProviderLinkNotFound
		}
		return nil, err
	}
	link.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return link, nil
}

// DeleteProviderLink removes the provider link for a folder.
func (s *Store) DeleteProviderLink(folderID int64) error {
	_, err := s.db.Exec("DELETE FROM series_provider_link WHERE folder_id = ?", folderID)
	return err
}

// UpdateMetadataLocks updates specific lock fields by name.
// Keys must be valid lock column names (e.g., "title_lock", "genres_lock").
// Returns ErrMetadataNotFound if no metadata row exists for the folder.
func (s *Store) UpdateMetadataLocks(folderID int64, locks map[string]bool) error {
	for key := range locks {
		if !validLockFields[key] {
			return fmt.Errorf("unknown lock field: %s", key)
		}
	}
	if len(locks) == 0 {
		return nil
	}

	var setClauses []string
	var args []interface{}
	for field, locked := range locks {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
		if locked {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	query := fmt.Sprintf("UPDATE series_metadata SET %s WHERE folder_id = ?", strings.Join(setClauses, ", "))
	args = append(args, folderID)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMetadataNotFound
	}
	return nil
}

// UpdateMetadataField updates a single scalar field by name and auto-locks it.
// Creates the metadata row if it doesn't exist yet.
func (s *Store) UpdateMetadataField(folderID int64, field string, value interface{}) error {
	lockCol, ok := validScalarFields[field]
	if !ok {
		return fmt.Errorf("unknown metadata field: %s", field)
	}

	// Ensure metadata row exists.
	if _, err := s.db.Exec("INSERT OR IGNORE INTO series_metadata (folder_id) VALUES (?)", folderID); err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE series_metadata SET %s = ?, %s = 1, updated_at = CURRENT_TIMESTAMP WHERE folder_id = ?", field, lockCol)
	_, err := s.db.Exec(query, value, folderID)
	return err
}

// ResetSeriesMetadata clears all metadata fields and locks for a folder
// but preserves the provider link.
func (s *Store) ResetSeriesMetadata(folderID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.resetMetadataInTx(tx, folderID); err != nil {
		return err
	}
	return tx.Commit()
}

// UnlinkSeriesMetadata clears all metadata fields and locks AND
// deletes the provider link for a folder.
func (s *Store) UnlinkSeriesMetadata(folderID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.resetMetadataInTx(tx, folderID); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM series_provider_link WHERE folder_id = ?", folderID); err != nil {
		return err
	}
	return tx.Commit()
}

// resetMetadataInTx is the shared implementation for Reset and Unlink.
func (s *Store) resetMetadataInTx(tx *sql.Tx, folderID int64) error {
	var metadataID int64
	err := tx.QueryRow("SELECT id FROM series_metadata WHERE folder_id = ?", folderID).Scan(&metadataID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	_, err = tx.Exec(`UPDATE series_metadata SET
		status = NULL, title = NULL, summary = NULL, publisher = NULL,
		reading_direction = NULL, age_rating = NULL, language = NULL,
		total_book_count = NULL, community_score = NULL,
		release_year = NULL, release_month = NULL, release_day = NULL,
		thumbnail_url = NULL,
		status_lock = 0, title_lock = 0, summary_lock = 0, publisher_lock = 0,
		reading_direction_lock = 0, age_rating_lock = 0, language_lock = 0,
		total_book_count_lock = 0, community_score_lock = 0, release_date_lock = 0,
		thumbnail_url_lock = 0, genres_lock = 0, tags_lock = 0, authors_lock = 0,
		links_lock = 0, titles_lock = 0,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, metadataID)
	if err != nil {
		return err
	}

	for _, table := range []string{
		"series_metadata_genres", "series_metadata_tags",
		"series_metadata_authors", "series_metadata_links",
		"series_metadata_titles",
	} {
		if _, err = tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE metadata_id = ?", table), metadataID); err != nil {
			return err
		}
	}
	return nil
}
