package anilist

import (
	"html"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/metadata"
)

var (
	htmlTagRe    = regexp.MustCompile(`<[^>]*>`)
	parenStripRe = regexp.MustCompile(`\s*\([^)]*\)\s*`)
)

// mapStatus converts an AniList status string to a SeriesStatus pointer.
func mapStatus(raw string) *metadata.SeriesStatus {
	switch raw {
	case "FINISHED":
		s := metadata.SeriesStatusCompleted
		return &s
	case "RELEASING":
		s := metadata.SeriesStatusOngoing
		return &s
	case "CANCELLED":
		s := metadata.SeriesStatusAbandoned
		return &s
	case "HIATUS":
		s := metadata.SeriesStatusHiatus
		return &s
	default:
		return nil
	}
}

// mapTitles builds SeriesTitle entries from AniList title fields.
func mapTitles(romaji, english, native, countryOfOrigin string) []metadata.SeriesTitle {
	var titles []metadata.SeriesTitle
	if romaji != "" {
		titles = append(titles, metadata.SeriesTitle{
			Title:    romaji,
			Type:     "ROMAJI",
			Language: "ja-Latn",
		})
	}
	if english != "" {
		titles = append(titles, metadata.SeriesTitle{
			Title:    english,
			Type:     "LOCALIZED",
			Language: "en",
		})
	}
	if native != "" {
		titles = append(titles, metadata.SeriesTitle{
			Title:    native,
			Type:     "NATIVE",
			Language: nativeLanguage(countryOfOrigin),
		})
	}
	return titles
}

func nativeLanguage(country string) string {
	switch country {
	case "JP":
		return "ja"
	case "KR":
		return "ko"
	case "CN":
		return "zh"
	case "TW":
		return "zh-TW"
	default:
		return ""
	}
}

// canonicalTitle returns the preferred display title: English > Romaji > Native.
func canonicalTitle(english, romaji, native string) string {
	if english != "" {
		return english
	}
	if romaji != "" {
		return romaji
	}
	return native
}

// mapRoleToAuthorRoles maps a cleaned AniList role string to AuthorRole(s).
// Returns nil for unknown roles (silently dropped).
func mapRoleToAuthorRoles(role string) []metadata.AuthorRole {
	lower := strings.ToLower(role)
	switch lower {
	case "story & art":
		return []metadata.AuthorRole{metadata.AuthorRoleWriter, metadata.AuthorRolePenciller}
	case "story", "original story", "original creator":
		return []metadata.AuthorRole{metadata.AuthorRoleWriter}
	case "art", "illustration":
		return []metadata.AuthorRole{metadata.AuthorRolePenciller}
	default:
		return nil
	}
}

// mapAuthorsFromMedia maps AniList staff edges to Author entries using an allowlist.
func mapAuthorsFromMedia(media anilist.MediaFull) []metadata.Author {
	type key struct {
		name string
		role metadata.AuthorRole
	}
	seen := make(map[key]bool)
	var authors []metadata.Author

	for _, edge := range media.Staff.Edges {
		cleaned := parenStripRe.ReplaceAllString(edge.Role, "")
		cleaned = strings.TrimSpace(cleaned)
		name := edge.Node.Name.Full

		roles := mapRoleToAuthorRoles(cleaned)
		for _, role := range roles {
			k := key{name: name, role: role}
			if !seen[k] {
				seen[k] = true
				authors = append(authors, metadata.Author{Name: name, Role: role})
			}
		}
	}

	if authors == nil {
		return []metadata.Author{}
	}
	return authors
}

// mapTagsFromMedia filters and sorts AniList tags by rank, spoiler status, and caps at 15.
func mapTagsFromMedia(media anilist.MediaFull, excludeSpoilers bool) []string {
	type ranked struct {
		name string
		rank int
	}
	var filtered []ranked

	for _, tag := range media.Tags {
		if tag.Rank == nil {
			continue
		}
		if excludeSpoilers && tag.IsMediaSpoiler {
			continue
		}
		if *tag.Rank < 60 {
			continue
		}
		filtered = append(filtered, ranked{name: tag.Name, rank: *tag.Rank})
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].rank > filtered[j].rank
	})

	if len(filtered) > 15 {
		filtered = filtered[:15]
	}

	result := make([]string, len(filtered))
	for i, f := range filtered {
		result[i] = f.name
	}
	return result
}

// mapScore converts AniList's 0-100 int score to a 0-10 float64, rounded to 1 decimal.
func mapScore(avg *int) *float64 {
	if avg == nil {
		return nil
	}
	v := math.Round(float64(*avg)/10.0*10) / 10
	return &v
}

// mapDate passes through date components independently.
func mapDate(year, month, day *int) (*int, *int, *int) {
	return year, month, day
}

// mapLanguage derives a BCP-47 language tag from AniList's countryOfOrigin.
func mapLanguage(countryOfOrigin string) *string {
	var lang string
	switch countryOfOrigin {
	case "JP":
		lang = "ja"
	case "KR":
		lang = "ko"
	case "CN":
		lang = "zh"
	case "TW":
		lang = "zh-TW"
	default:
		return nil
	}
	return &lang
}

// mapAgeRating converts AniList's isAdult boolean to an age rating pointer.
func mapAgeRating(isAdult bool) *int {
	if isAdult {
		v := 18
		return &v
	}
	return nil
}

// mapLinksFromMedia converts AniList external links and always appends the AniList site URL.
func mapLinksFromMedia(media anilist.MediaFull) []metadata.WebLink {
	links := make([]metadata.WebLink, 0, len(media.ExternalLinks)+1)
	for _, el := range media.ExternalLinks {
		links = append(links, metadata.WebLink{Label: el.Site, URL: el.URL})
	}
	links = append(links, metadata.WebLink{Label: "AniList", URL: media.SiteUrl})
	return links
}

// mapDescription strips HTML tags and unescapes entities from AniList descriptions.
func mapDescription(raw string) *string {
	if raw == "" {
		return nil
	}
	cleaned := htmlTagRe.ReplaceAllString(raw, "")
	cleaned = html.UnescapeString(cleaned)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

// mapCover returns a pointer to the cover URL, or nil if empty.
func mapCover(large string) *string {
	if large == "" {
		return nil
	}
	return &large
}

// MapMediaToSeriesMetadata converts an AniList MediaFull to a SeriesMetadata.
func MapMediaToSeriesMetadata(media anilist.MediaFull, excludeSpoilers bool) *metadata.SeriesMetadata {
	title := canonicalTitle(media.Title.English, media.Title.Romaji, media.Title.Native)
	releaseYear, releaseMonth, releaseDay := mapDate(media.StartDate.Year, media.StartDate.Month, media.StartDate.Day)

	m := &metadata.SeriesMetadata{
		Status:         mapStatus(media.Status),
		Titles:         mapTitles(media.Title.Romaji, media.Title.English, media.Title.Native, media.CountryOfOrigin),
		Summary:        mapDescription(media.Description),
		Genres:         media.Genres,
		Tags:           mapTagsFromMedia(media, excludeSpoilers),
		Authors:        mapAuthorsFromMedia(media),
		TotalBookCount: media.Volumes,
		CommunityScore: mapScore(media.AverageScore),
		ReleaseYear:    releaseYear,
		ReleaseMonth:   releaseMonth,
		ReleaseDay:     releaseDay,
		Language:       mapLanguage(media.CountryOfOrigin),
		AgeRating:      mapAgeRating(media.IsAdult),
		Links:          mapLinksFromMedia(media),
		ThumbnailURL:   mapCover(media.CoverImage.Large),
	}

	if title != "" {
		m.Title = &title
	}

	return m
}

// MapMediaToSearchResult converts an AniList MediaFull to a SeriesSearchResult.
func MapMediaToSearchResult(media anilist.MediaFull) metadata.SeriesSearchResult {
	title := media.Title.English
	if title == "" {
		title = media.Title.Romaji
	}
	return metadata.SeriesSearchResult{
		Title:        title,
		ProviderName: "anilist",
		ResultID:     strconv.Itoa(media.ID),
		ImageURL:     media.CoverImage.Large,
		URL:          media.SiteUrl,
	}
}
