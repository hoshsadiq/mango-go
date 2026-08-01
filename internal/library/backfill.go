// This file contains the background job for backfilling chapter metadata.
// It processes chapters that have no corresponding chapter_metadata row,
// extracting metadata from ComicInfo.xml and filenames.

package library

import (
	"database/sql"
	"log"
	"path/filepath"
	"strings"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/library/chapterparse"
	"github.com/vrsandeep/mango-go/internal/library/comicinfo"
	"github.com/vrsandeep/mango-go/internal/metadata"
	"github.com/vrsandeep/mango-go/internal/store"
)

const backfillBatchSize = 50

// chapterRow holds the minimal info needed to process a chapter for metadata backfill.
type chapterRow struct {
	ID   int64
	Path string
}

// ParseChapterMetadata is a background job that processes chapters without
// existing chapter_metadata rows. It extracts metadata from ComicInfo.xml
// (if present) and the filename, merges them (ComicInfo takes priority),
// and upserts the result. Processes in batches of 50, logging per-chapter
// errors without aborting.
func ParseChapterMetadata(ctx jobs.JobContext) {
	jobID := "parse-chapter-metadata"
	sendProgress(ctx, jobID, "Starting chapter metadata backfill...", 0, false)

	metaStore := store.NewChapterMetadataStore(ctx.DB())
	totalProcessed := 0
	totalErrors := 0

	for {
		chapters, err := getChaptersWithoutMetadata(ctx.DB(), backfillBatchSize)
		if err != nil {
			log.Printf("[%s] Error querying chapters without metadata: %v", jobID, err)
			break
		}
		if len(chapters) == 0 {
			break
		}

		for _, ch := range chapters {
			if err := processChapterMetadata(metaStore, ch); err != nil {
				totalErrors++
				log.Printf("[%s] Error processing chapter %d (%s): %v", jobID, ch.ID, ch.Path, err)
				continue
			}
			totalProcessed++
		}

		log.Printf("[%s] Batch complete: processed %d chapters so far (%d errors)", jobID, totalProcessed, totalErrors)

		// If we got fewer than a full batch, we're done.
		if len(chapters) < backfillBatchSize {
			break
		}
	}

	log.Printf("[%s] Finished: %d chapters processed, %d errors", jobID, totalProcessed, totalErrors)
	sendProgress(ctx, jobID, "Chapter metadata backfill completed.", 100, true)
}

// getChaptersWithoutMetadata returns up to `limit` chapters that have no
// corresponding row in chapter_metadata.
func getChaptersWithoutMetadata(db *sql.DB, limit int) ([]chapterRow, error) {
	rows, err := db.Query(`
		SELECT c.id, c.path
		FROM chapters c
		LEFT JOIN chapter_metadata cm ON c.id = cm.chapter_id
		WHERE cm.id IS NULL
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []chapterRow
	for rows.Next() {
		var ch chapterRow
		if err := rows.Scan(&ch.ID, &ch.Path); err != nil {
			return nil, err
		}
		chapters = append(chapters, ch)
	}
	return chapters, rows.Err()
}

// processChapterMetadata extracts metadata from a single chapter's archive
// (ComicInfo.xml) and filename, merges them, and upserts the result.
func processChapterMetadata(metaStore *store.ChapterMetadataStore, ch chapterRow) error {
	merged := &metadata.ChapterMetadata{}

	// 1. Parse filename (lower priority, applied first, then overwritten by ComicInfo).
	base := filepath.Base(ch.Path)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	parsed := chapterparse.ParseFilename(nameWithoutExt)
	applyParsedFilename(merged, parsed)

	// 2. Extract ComicInfo.xml (higher priority, overwrites filename fields).
	ci, err := comicinfo.ExtractFromArchive(ch.Path)
	if err != nil {
		// Log but still save whatever we got from the filename.
		log.Printf("[parse-chapter-metadata] ComicInfo extraction failed for %s: %v", ch.Path, err)
	}
	if ci != nil {
		ciMeta := comicinfo.MapToChapterMetadata(ci)
		mergeComicInfoOver(merged, ciMeta)
	}

	// 3. Upsert, even if both sources yielded nothing, this creates the
	//    metadata row so the chapter won't be re-processed.
	return metaStore.UpsertChapterMetadata(ch.ID, merged)
}

// applyParsedFilename sets metadata fields from a ParsedFilename.
func applyParsedFilename(dst *metadata.ChapterMetadata, pf *chapterparse.ParsedFilename) {
	if pf == nil {
		return
	}
	dst.Title = pf.Title
	dst.Number = pf.Number
	dst.SortNumber = pf.SortNumber
	dst.Volume = pf.Volume
}

// mergeComicInfoOver overwrites dst fields with non-nil values from src (ComicInfo).
func mergeComicInfoOver(dst, src *metadata.ChapterMetadata) {
	if src == nil {
		return
	}
	if src.Title != nil {
		dst.Title = src.Title
	}
	if src.Number != nil {
		dst.Number = src.Number
	}
	if src.SortNumber != nil {
		dst.SortNumber = src.SortNumber
	}
	if src.Volume != nil {
		dst.Volume = src.Volume
	}
	if src.Summary != nil {
		dst.Summary = src.Summary
	}
	if src.Notes != nil {
		dst.Notes = src.Notes
	}
	if src.ReleaseDate != nil {
		dst.ReleaseDate = src.ReleaseDate
	}
	if src.Language != nil {
		dst.Language = src.Language
	}
	if src.ChapterType != nil {
		dst.ChapterType = src.ChapterType
	}
	if src.AgeRating != nil {
		dst.AgeRating = src.AgeRating
	}
	if src.Web != nil {
		dst.Web = src.Web
	}
	if src.Characters != nil {
		dst.Characters = src.Characters
	}
	if src.Teams != nil {
		dst.Teams = src.Teams
	}
	if src.Locations != nil {
		dst.Locations = src.Locations
	}
	if src.ScanlationGroup != nil {
		dst.ScanlationGroup = src.ScanlationGroup
	}
	if src.StoryArc != nil {
		dst.StoryArc = src.StoryArc
	}
	if src.StoryArcNumber != nil {
		dst.StoryArcNumber = src.StoryArcNumber
	}
	if src.Authors != nil {
		dst.Authors = src.Authors
	}
	if src.Genres != nil {
		dst.Genres = src.Genres
	}
	if src.Tags != nil {
		dst.Tags = src.Tags
	}
}
