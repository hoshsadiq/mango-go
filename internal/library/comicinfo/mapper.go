package comicinfo

import (
	"fmt"
	"strings"

	"github.com/vrsandeep/mango-go/internal/metadata"
)

// MapToChapterMetadata converts a ComicInfo struct to a ChapterMetadata struct.
// Series-level fields (Series, Publisher, Count, Manga) are intentionally skipped.
func MapToChapterMetadata(ci *ComicInfo) *metadata.ChapterMetadata {
	meta := &metadata.ChapterMetadata{}

	setStringPtr(&meta.Title, ci.Title)
	setStringPtr(&meta.Number, ci.Number)
	setStringPtr(&meta.Volume, ci.Volume)
	setStringPtr(&meta.Summary, ci.Summary)
	setStringPtr(&meta.Notes, ci.Notes)
	setStringPtr(&meta.Language, ci.LanguageISO)
	setStringPtr(&meta.AgeRating, ci.AgeRating)
	setStringPtr(&meta.Web, ci.Web)
	setStringPtr(&meta.Characters, ci.Characters)
	setStringPtr(&meta.Teams, ci.Teams)
	setStringPtr(&meta.Locations, ci.Locations)
	setStringPtr(&meta.ScanlationGroup, ci.ScanInformation)
	setStringPtr(&meta.StoryArc, ci.StoryArc)
	setStringPtr(&meta.StoryArcNumber, ci.StoryArcNumber)

	if ci.Year != "" {
		meta.ReleaseDate = buildReleaseDate(ci.Year, ci.Month, ci.Day)
	}

	meta.Genres = splitCommaSeparated(ci.Genre)
	meta.Tags = splitCommaSeparated(ci.Tags)

	meta.Authors = mapAuthors(ci)

	if ci.Format != "" {
		var ct string
		switch strings.TrimSpace(ci.Format) {
		case "Special":
			ct = string(metadata.ChapterTypeSpecial)
		case "Annual":
			ct = string(metadata.ChapterTypeAnnual)
		case "Omnibus":
			ct = string(metadata.ChapterTypeOmnibus)
		case "One-Shot", "OneShot":
			ct = string(metadata.ChapterTypeOneShot)
		case "Bonus":
			ct = string(metadata.ChapterTypeBonus)
		case "Omake":
			ct = string(metadata.ChapterTypeOmake)
		}
		if ct != "" {
			meta.ChapterType = &ct
		}
	}

	return meta
}

// setStringPtr sets the target pointer to the value if it is non-empty.
func setStringPtr(target **string, value string) {
	if value != "" {
		*target = &value
	}
}

// buildReleaseDate constructs a "YYYY-MM-DD" date string.
// Uses "01" for missing Month or Day.
func buildReleaseDate(year, month, day string) *string {
	m := month
	if m == "" {
		m = "01"
	}
	d := day
	if d == "" {
		d = "01"
	}
	date := fmt.Sprintf("%04s-%02s-%02s", year, m, d)
	return &date
}

// splitCommaSeparated splits a comma-separated string into trimmed, non-empty parts.
func splitCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// mapAuthors extracts all author entries from ComicInfo role fields.
func mapAuthors(ci *ComicInfo) []metadata.ChapterMetadataAuthor {
	var authors []metadata.ChapterMetadataAuthor

	addAuthors(&authors, ci.Writer, string(metadata.AuthorRoleWriter))
	addAuthors(&authors, ci.Penciller, string(metadata.AuthorRolePenciller))
	addAuthors(&authors, ci.Inker, string(metadata.AuthorRoleInker))
	addAuthors(&authors, ci.Colorist, string(metadata.AuthorRoleColorist))
	addAuthors(&authors, ci.Letterer, string(metadata.AuthorRoleLetterer))
	addAuthors(&authors, ci.CoverArtist, string(metadata.AuthorRoleCoverArtist))
	addAuthors(&authors, ci.Editor, string(metadata.AuthorRoleEditor))
	addAuthors(&authors, ci.Translator, string(metadata.AuthorRoleTranslator))

	if len(authors) == 0 {
		return nil
	}
	return authors
}

// addAuthors splits a comma-separated names field and appends one ChapterMetadataAuthor per name.
func addAuthors(authors *[]metadata.ChapterMetadataAuthor, names string, role string) {
	if names == "" {
		return
	}
	for _, name := range strings.Split(names, ",") {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		*authors = append(*authors, metadata.ChapterMetadataAuthor{
			Name: trimmed,
			Role: role,
		})
	}
}
