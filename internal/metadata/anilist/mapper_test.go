package anilist

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/metadata"
)

func ptrTo[T any](v T) *T { return &v }

func TestMapStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  *metadata.SeriesStatus
	}{
		{"FINISHED", ptrTo(metadata.SeriesStatusCompleted)},
		{"RELEASING", ptrTo(metadata.SeriesStatusOngoing)},
		{"CANCELLED", ptrTo(metadata.SeriesStatusAbandoned)},
		{"HIATUS", ptrTo(metadata.SeriesStatusHiatus)},
		{"NOT_YET_RELEASED", nil},
		{"UNKNOWN_VALUE", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapStatus(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("mapStatus(%q) = %v, want nil", tt.input, *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("mapStatus(%q) = nil, want %v", tt.input, *tt.want)
			}
			if *got != *tt.want {
				t.Errorf("mapStatus(%q) = %v, want %v", tt.input, *got, *tt.want)
			}
		})
	}
}

func TestMapTitles(t *testing.T) {
	t.Parallel()
	t.Run("all three populated JP origin", func(t *testing.T) {
		titles := mapTitles("Naruto", "Naruto", "ナルト", "JP")
		if len(titles) != 3 {
			t.Fatalf("got %d titles, want 3", len(titles))
		}
		assertTitle(t, titles[0], "Naruto", "ROMAJI", "ja-Latn")
		assertTitle(t, titles[1], "Naruto", "LOCALIZED", "en")
		assertTitle(t, titles[2], "ナルト", "NATIVE", "ja")
	})

	t.Run("KR origin native language", func(t *testing.T) {
		titles := mapTitles("Solo Leveling", "", "나 혼자만 레벨업", "KR")
		if len(titles) != 2 {
			t.Fatalf("got %d titles, want 2", len(titles))
		}
		assertTitle(t, titles[0], "Solo Leveling", "ROMAJI", "ja-Latn")
		assertTitle(t, titles[1], "나 혼자만 레벨업", "NATIVE", "ko")
	})

	t.Run("CN origin native language", func(t *testing.T) {
		titles := mapTitles("", "Test", "测试", "CN")
		if len(titles) != 2 {
			t.Fatalf("got %d titles, want 2", len(titles))
		}
		assertTitle(t, titles[0], "Test", "LOCALIZED", "en")
		assertTitle(t, titles[1], "测试", "NATIVE", "zh")
	})

	t.Run("TW origin native language", func(t *testing.T) {
		titles := mapTitles("", "", "測試", "TW")
		if len(titles) != 1 {
			t.Fatalf("got %d titles, want 1", len(titles))
		}
		assertTitle(t, titles[0], "測試", "NATIVE", "zh-TW")
	})

	t.Run("skip empty strings", func(t *testing.T) {
		titles := mapTitles("Romaji Only", "", "", "JP")
		if len(titles) != 1 {
			t.Fatalf("got %d titles, want 1", len(titles))
		}
		assertTitle(t, titles[0], "Romaji Only", "ROMAJI", "ja-Latn")
	})

	t.Run("all empty", func(t *testing.T) {
		titles := mapTitles("", "", "", "JP")
		if len(titles) != 0 {
			t.Fatalf("got %d titles, want 0", len(titles))
		}
	})
}

func assertTitle(t *testing.T, got metadata.SeriesTitle, title, typ, lang string) {
	t.Helper()
	if got.Title != title || got.Type != typ || got.Language != lang {
		t.Errorf("got {%q, %q, %q}, want {%q, %q, %q}",
			got.Title, got.Type, got.Language, title, typ, lang)
	}
}

func TestCanonicalTitle(t *testing.T) {
	t.Parallel()
	t.Run("prefers English", func(t *testing.T) {
		got := canonicalTitle("English", "Romaji", "Native")
		if got != "English" {
			t.Errorf("got %q, want %q", got, "English")
		}
	})

	t.Run("falls back to Romaji", func(t *testing.T) {
		got := canonicalTitle("", "Romaji", "Native")
		if got != "Romaji" {
			t.Errorf("got %q, want %q", got, "Romaji")
		}
	})

	t.Run("falls back to Native", func(t *testing.T) {
		got := canonicalTitle("", "", "Native")
		if got != "Native" {
			t.Errorf("got %q, want %q", got, "Native")
		}
	})

	t.Run("all empty returns empty", func(t *testing.T) {
		got := canonicalTitle("", "", "")
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestMapAuthorsFromMedia(t *testing.T) {
	t.Parallel()
	makeMedia := func(edges ...struct{ role, name string }) anilist.MediaFull {
		var m anilist.MediaFull
		for _, e := range edges {
			m.Staff.Edges = append(m.Staff.Edges, anilist.MediaStaffEdge{
				Role: e.role,
				Node: anilist.MediaStaffNode{
					Name: anilist.MediaStaffName{Full: e.name},
				},
			})
		}
		return m
	}

	rn := func(role, name string) struct{ role, name string } {
		return struct{ role, name string }{role, name}
	}

	t.Run("Story & Art creates two entries", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Story & Art", "Masashi Kishimoto")))
		if len(authors) != 2 {
			t.Fatalf("got %d authors, want 2", len(authors))
		}
		assertAuthor(t, authors[0], "Masashi Kishimoto", metadata.AuthorRoleWriter)
		assertAuthor(t, authors[1], "Masashi Kishimoto", metadata.AuthorRolePenciller)
	})

	t.Run("Story maps to WRITER", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Story", "Writer Name")))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1", len(authors))
		}
		assertAuthor(t, authors[0], "Writer Name", metadata.AuthorRoleWriter)
	})

	t.Run("Original Story maps to WRITER", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Original Story", "Creator")))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1", len(authors))
		}
		assertAuthor(t, authors[0], "Creator", metadata.AuthorRoleWriter)
	})

	t.Run("Original Creator maps to WRITER", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Original Creator", "OC")))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1", len(authors))
		}
		assertAuthor(t, authors[0], "OC", metadata.AuthorRoleWriter)
	})

	t.Run("Art maps to PENCILLER only", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Art", "Artist")))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1", len(authors))
		}
		assertAuthor(t, authors[0], "Artist", metadata.AuthorRolePenciller)
	})

	t.Run("Illustration maps to PENCILLER", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Illustration", "Illustrator")))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1", len(authors))
		}
		assertAuthor(t, authors[0], "Illustrator", metadata.AuthorRolePenciller)
	})

	t.Run("parenthetical stripped before matching", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(rn("Story (Original)", "Author")))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1", len(authors))
		}
		assertAuthor(t, authors[0], "Author", metadata.AuthorRoleWriter)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(
			rn("story", "Lower"),
			rn("ART", "Upper"),
			rn("Story & art", "Mixed"),
		))
		if len(authors) != 4 {
			t.Fatalf("got %d authors, want 4 (1 writer + 1 penciller + 2 from Story & art)", len(authors))
		}
	})

	t.Run("unknown roles mapped to OTHER", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(
			rn("Director", "D"),
			rn("Character Design", "CD"),
		))
		if len(authors) != 2 {
			t.Fatalf("got %d authors, want 2", len(authors))
		}
		assertAuthor(t, authors[0], "D", metadata.AuthorRoleOther)
		assertAuthor(t, authors[1], "CD", metadata.AuthorRoleOther)
	})

	t.Run("dedup identical name+role pairs", func(t *testing.T) {
		authors := mapAuthorsFromMedia(makeMedia(
			rn("Story", "Same Person"),
			rn("Story", "Same Person"),
		))
		if len(authors) != 1 {
			t.Fatalf("got %d authors, want 1 (deduped)", len(authors))
		}
	})

	t.Run("empty edges", func(t *testing.T) {
		authors := mapAuthorsFromMedia(anilist.MediaFull{})
		if authors == nil {
			t.Fatal("got nil, want non-nil empty slice")
		}
		if len(authors) != 0 {
			t.Fatalf("got %d authors, want 0", len(authors))
		}
	})
}

func assertAuthor(t *testing.T, got metadata.Author, name string, role metadata.AuthorRole) {
	t.Helper()
	if got.Name != name || got.Role != role {
		t.Errorf("got {%q, %q}, want {%q, %q}", got.Name, got.Role, name, role)
	}
}

func TestMapTagsFromMedia(t *testing.T) {
	t.Parallel()
	makeMedia := func(tags ...struct {
		name    string
		rank    *int
		spoiler bool
	}) anilist.MediaFull {
		var m anilist.MediaFull
		for _, tg := range tags {
			m.Tags = append(m.Tags, anilist.MediaTag{Name: tg.name, Rank: tg.rank, IsMediaSpoiler: tg.spoiler})
		}
		return m
	}

	tag := func(name string, rank *int, spoiler bool) struct {
		name    string
		rank    *int
		spoiler bool
	} {
		return struct {
			name    string
			rank    *int
			spoiler bool
		}{name, rank, spoiler}
	}

	t.Run("filters by rank >= 60", func(t *testing.T) {
		m := makeMedia(
			tag("Action", ptrTo(90), false),
			tag("Low", ptrTo(59), false),
			tag("Threshold", ptrTo(60), false),
		)
		result := mapTagsFromMedia(m, false)
		if len(result) != 2 {
			t.Fatalf("got %d tags, want 2", len(result))
		}
		if result[0] != "Action" || result[1] != "Threshold" {
			t.Errorf("got %v, want [Action, Threshold]", result)
		}
	})

	t.Run("drops nil rank", func(t *testing.T) {
		m := makeMedia(
			tag("NoRank", nil, false),
			tag("HasRank", ptrTo(80), false),
		)
		result := mapTagsFromMedia(m, false)
		if len(result) != 1 {
			t.Fatalf("got %d tags, want 1", len(result))
		}
		if result[0] != "HasRank" {
			t.Errorf("got %q, want HasRank", result[0])
		}
	})

	t.Run("excludes spoilers when configured", func(t *testing.T) {
		m := makeMedia(
			tag("Action", ptrTo(90), false),
			tag("PlotTwist", ptrTo(80), true),
		)
		result := mapTagsFromMedia(m, true)
		if len(result) != 1 {
			t.Fatalf("got %d tags, want 1", len(result))
		}
		if result[0] != "Action" {
			t.Errorf("got %q, want Action", result[0])
		}
	})

	t.Run("includes spoilers when not configured", func(t *testing.T) {
		m := makeMedia(
			tag("Action", ptrTo(90), false),
			tag("PlotTwist", ptrTo(80), true),
		)
		result := mapTagsFromMedia(m, false)
		if len(result) != 2 {
			t.Fatalf("got %d tags, want 2", len(result))
		}
	})

	t.Run("caps at 15 and sorted descending", func(t *testing.T) {
		var tags []struct {
			name    string
			rank    *int
			spoiler bool
		}
		for i := 0; i < 20; i++ {
			tags = append(tags, tag("Tag"+string(rune('A'+i)), ptrTo(60+i), false))
		}
		m := makeMedia(tags...)
		result := mapTagsFromMedia(m, false)
		if len(result) != 15 {
			t.Fatalf("got %d tags, want 15 (capped)", len(result))
		}
		// first should be highest rank (60+19=79)
		if result[0] != "Tag"+string(rune('A'+19)) {
			t.Errorf("first tag should be highest rank, got %q", result[0])
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := mapTagsFromMedia(anilist.MediaFull{}, false)
		if result == nil {
			t.Fatal("got nil, want non-nil empty slice")
		}
		if len(result) != 0 {
			t.Fatalf("got %d tags, want 0", len(result))
		}
	})
}

func TestMapScore(t *testing.T) {
	t.Parallel()
	t.Run("nil returns nil", func(t *testing.T) {
		if got := mapScore(nil); got != nil {
			t.Errorf("got %v, want nil", *got)
		}
	})

	t.Run("84 becomes 8.4", func(t *testing.T) {
		got := mapScore(ptrTo(84))
		if got == nil {
			t.Fatal("got nil, want 8.4")
		}
		if *got != 8.4 {
			t.Errorf("got %v, want 8.4", *got)
		}
	})

	t.Run("0 becomes 0.0", func(t *testing.T) {
		got := mapScore(ptrTo(0))
		if got == nil {
			t.Fatal("got nil, want 0.0")
		}
		if *got != 0.0 {
			t.Errorf("got %v, want 0.0", *got)
		}
	})

	t.Run("100 becomes 10.0", func(t *testing.T) {
		got := mapScore(ptrTo(100))
		if got == nil {
			t.Fatal("got nil, want 10.0")
		}
		if *got != 10.0 {
			t.Errorf("got %v, want 10.0", *got)
		}
	})

	t.Run("73 becomes 7.3", func(t *testing.T) {
		got := mapScore(ptrTo(73))
		if got == nil {
			t.Fatal("got nil, want 7.3")
		}
		if *got != 7.3 {
			t.Errorf("got %v, want 7.3", *got)
		}
	})
}

func TestMapDate(t *testing.T) {
	t.Parallel()
	t.Run("all present", func(t *testing.T) {
		y, m, d := mapDate(ptrTo(1999), ptrTo(9), ptrTo(21))
		assertIntPtr(t, y, 1999, "year")
		assertIntPtr(t, m, 9, "month")
		assertIntPtr(t, d, 21, "day")
	})

	t.Run("year only", func(t *testing.T) {
		y, m, d := mapDate(ptrTo(2020), nil, nil)
		assertIntPtr(t, y, 2020, "year")
		if m != nil {
			t.Error("month should be nil")
		}
		if d != nil {
			t.Error("day should be nil")
		}
	})

	t.Run("all nil", func(t *testing.T) {
		y, m, d := mapDate(nil, nil, nil)
		if y != nil || m != nil || d != nil {
			t.Error("all should be nil")
		}
	})

	t.Run("partial month+day without year", func(t *testing.T) {
		y, m, d := mapDate(nil, ptrTo(3), ptrTo(15))
		if y != nil {
			t.Error("year should be nil")
		}
		assertIntPtr(t, m, 3, "month")
		assertIntPtr(t, d, 15, "day")
	})
}

func assertIntPtr(t *testing.T, got *int, want int, label string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: got nil, want %d", label, want)
	}
	if *got != want {
		t.Errorf("%s: got %d, want %d", label, *got, want)
	}
}

func TestMapLanguage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  *string
	}{
		{"JP", ptrTo("ja")},
		{"KR", ptrTo("ko")},
		{"CN", ptrTo("zh")},
		{"TW", ptrTo("zh-TW")},
		{"US", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapLanguage(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("got %q, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("got nil, want %q", *tt.want)
			}
			if *got != *tt.want {
				t.Errorf("got %q, want %q", *got, *tt.want)
			}
		})
	}
}

func TestMapAgeRating(t *testing.T) {
	t.Parallel()
	t.Run("adult true returns 18", func(t *testing.T) {
		got := mapAgeRating(true)
		if got == nil {
			t.Fatal("got nil, want 18")
		}
		if *got != 18 {
			t.Errorf("got %d, want 18", *got)
		}
	})

	t.Run("adult false returns nil", func(t *testing.T) {
		got := mapAgeRating(false)
		if got != nil {
			t.Errorf("got %d, want nil", *got)
		}
	})
}

func TestMapLinksFromMedia(t *testing.T) {
	t.Parallel()
	t.Run("maps external links and appends AniList", func(t *testing.T) {
		var m anilist.MediaFull
		m.SiteURL = "https://anilist.co/manga/20"
		m.ExternalLinks = append(m.ExternalLinks, anilist.MediaExternalLink{URL: "https://mal.net/123", Site: "MyAnimeList", Type: "INFO"})
		m.ExternalLinks = append(m.ExternalLinks, anilist.MediaExternalLink{URL: "https://mangadex.org/title/abc", Site: "MangaDex", Type: "READING"})

		links := mapLinksFromMedia(m)
		if len(links) != 3 {
			t.Fatalf("got %d links, want 3", len(links))
		}
		if links[0].Label != "MyAnimeList" || links[0].URL != "https://mal.net/123" {
			t.Errorf("link 0: got {%q, %q}", links[0].Label, links[0].URL)
		}
		if links[1].Label != "MangaDex" || links[1].URL != "https://mangadex.org/title/abc" {
			t.Errorf("link 1: got {%q, %q}", links[1].Label, links[1].URL)
		}
		if links[2].Label != "AniList" || links[2].URL != "https://anilist.co/manga/20" {
			t.Errorf("link 2: got {%q, %q}", links[2].Label, links[2].URL)
		}
	})

	t.Run("no external links still appends AniList", func(t *testing.T) {
		var m anilist.MediaFull
		m.SiteURL = "https://anilist.co/manga/20"
		links := mapLinksFromMedia(m)
		if len(links) != 1 {
			t.Fatalf("got %d links, want 1", len(links))
		}
		if links[0].Label != "AniList" {
			t.Errorf("got label %q, want AniList", links[0].Label)
		}
	})
}

func TestMapDescription(t *testing.T) {
	t.Parallel()
	t.Run("strips HTML tags", func(t *testing.T) {
		got := mapDescription("<br>A ninja story<br/>")
		if got == nil {
			t.Fatal("got nil")
		}
		if *got != "A ninja story" {
			t.Errorf("got %q, want %q", *got, "A ninja story")
		}
	})

	t.Run("unescapes HTML entities", func(t *testing.T) {
		got := mapDescription("Tom &amp; Jerry &lt;3")
		if got == nil {
			t.Fatal("got nil")
		}
		if *got != "Tom & Jerry <3" {
			t.Errorf("got %q, want %q", *got, "Tom & Jerry <3")
		}
	})

	t.Run("empty after strip returns nil", func(t *testing.T) {
		got := mapDescription("<br>  <br/>")
		if got != nil {
			t.Errorf("got %q, want nil", *got)
		}
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		got := mapDescription("")
		if got != nil {
			t.Errorf("got %q, want nil", *got)
		}
	})

	t.Run("complex HTML", func(t *testing.T) {
		got := mapDescription("<p>Line 1</p><br><i>italic</i> text")
		if got == nil {
			t.Fatal("got nil")
		}
		if *got != "Line 1\nitalic text" {
			t.Errorf("got %q, want %q", *got, "Line 1\nitalic text")
		}
	})
}

func TestMapCover(t *testing.T) {
	t.Parallel()
	t.Run("non-empty returns pointer", func(t *testing.T) {
		got := mapCover("https://img.anilist.co/cover.jpg")
		if got == nil {
			t.Fatal("got nil")
		}
		if *got != "https://img.anilist.co/cover.jpg" {
			t.Errorf("got %q", *got)
		}
	})

	t.Run("empty returns nil", func(t *testing.T) {
		got := mapCover("")
		if got != nil {
			t.Errorf("got %q, want nil", *got)
		}
	})
}

func TestMapMediaToSeriesMetadata(t *testing.T) {
	t.Parallel()
	media := anilist.MediaFull{
		ID:              20,
		Status:          "FINISHED",
		Description:     "<br>A ninja story",
		Genres:          []string{"Action", "Adventure"},
		IsAdult:         false,
		CountryOfOrigin: "JP",
		SiteURL:         "https://anilist.co/manga/20",
		Volumes:         ptrTo(72),
		AverageScore:    ptrTo(84),
	}
	media.Title = anilist.MediaTitle{Romaji: "NARUTO", English: "Naruto", Native: "ナルト"}
	media.Tags = []anilist.MediaTag{{Name: "Shounen", Rank: ptrTo(85), IsMediaSpoiler: false}}
	media.Staff = anilist.MediaStaff{Edges: []anilist.MediaStaffEdge{{
		Role: "Story & Art",
		Node: anilist.MediaStaffNode{Name: anilist.MediaStaffName{Full: "Masashi Kishimoto"}},
	}}}
	media.StartDate = anilist.MediaDate{Year: ptrTo(1999), Month: ptrTo(9), Day: ptrTo(21)}
	media.CoverImage = anilist.MediaCoverImage{Large: "https://img.anilist.co/naruto.jpg"}
	media.ExternalLinks = []anilist.MediaExternalLink{{URL: "https://mal.net/20", Site: "MyAnimeList", Type: "INFO"}}

	result := MapMediaToSeriesMetadata(media, true)
	if result == nil {
		t.Fatal("got nil result")
	}

	if result.Status == nil || *result.Status != metadata.SeriesStatusCompleted {
		t.Errorf("status: got %v, want COMPLETED", result.Status)
	}

	if result.Title == nil || *result.Title != "Naruto" {
		t.Errorf("title: got %v, want Naruto", result.Title)
	}

	if len(result.Titles) != 3 {
		t.Fatalf("titles: got %d, want 3", len(result.Titles))
	}

	// Summary (HTML stripped)
	if result.Summary == nil || *result.Summary != "A ninja story" {
		t.Errorf("summary: got %v, want 'A ninja story'", result.Summary)
	}

	if len(result.Genres) != 2 || result.Genres[0] != "Action" {
		t.Errorf("genres: got %v", result.Genres)
	}

	if len(result.Tags) != 1 || result.Tags[0] != "Shounen" {
		t.Errorf("tags: got %v", result.Tags)
	}

	// Authors (Story & Art = 2 entries)
	if len(result.Authors) != 2 {
		t.Fatalf("authors: got %d, want 2", len(result.Authors))
	}

	if result.TotalBookCount == nil || *result.TotalBookCount != 72 {
		t.Errorf("totalBookCount: got %v, want 72", result.TotalBookCount)
	}

	if result.CommunityScore == nil || *result.CommunityScore != 8.4 {
		t.Errorf("communityScore: got %v, want 8.4", result.CommunityScore)
	}

	assertIntPtr(t, result.ReleaseYear, 1999, "releaseYear")
	assertIntPtr(t, result.ReleaseMonth, 9, "releaseMonth")
	assertIntPtr(t, result.ReleaseDay, 21, "releaseDay")

	if result.Language == nil || *result.Language != "ja" {
		t.Errorf("language: got %v, want ja", result.Language)
	}

	// AgeRating (not adult)
	if result.AgeRating != nil {
		t.Errorf("ageRating: got %d, want nil", *result.AgeRating)
	}

	// Links (external + AniList)
	if len(result.Links) != 2 {
		t.Fatalf("links: got %d, want 2", len(result.Links))
	}
	if result.Links[len(result.Links)-1].Label != "AniList" {
		t.Error("last link should be AniList")
	}

	if result.ThumbnailURL == nil || *result.ThumbnailURL != "https://img.anilist.co/naruto.jpg" {
		t.Errorf("thumbnailURL: got %v", result.ThumbnailURL)
	}
}

func TestMapMediaToSearchResult(t *testing.T) {
	t.Parallel()
	t.Run("uses English title when available", func(t *testing.T) {
		media := anilist.MediaFull{ID: 20, SiteURL: "https://anilist.co/manga/20"}
		media.Title.English = "Naruto"
		media.Title.Romaji = "NARUTO"
		media.CoverImage.Large = "https://img.anilist.co/naruto.jpg"

		r := MapMediaToSearchResult(media)
		if r.Title != "Naruto" {
			t.Errorf("title: got %q, want Naruto", r.Title)
		}
		if r.ProviderName != "anilist" {
			t.Errorf("providerName: got %q, want anilist", r.ProviderName)
		}
		if r.ResultID != "20" {
			t.Errorf("resultID: got %q, want 20", r.ResultID)
		}
		if r.ImageURL != "https://img.anilist.co/naruto.jpg" {
			t.Errorf("imageURL: got %q", r.ImageURL)
		}
		if r.URL != "https://anilist.co/manga/20" {
			t.Errorf("url: got %q", r.URL)
		}
	})

	t.Run("falls back to Romaji when English empty", func(t *testing.T) {
		media := anilist.MediaFull{ID: 30}
		media.Title.Romaji = "Solo Leveling"
		r := MapMediaToSearchResult(media)
		if r.Title != "Solo Leveling" {
			t.Errorf("title: got %q, want Solo Leveling", r.Title)
		}
	})

	t.Run("falls back to Native when English and Romaji empty", func(t *testing.T) {
		media := anilist.MediaFull{ID: 40}
		media.Title.Native = "ナルト"
		r := MapMediaToSearchResult(media)
		if r.Title != "ナルト" {
			t.Errorf("title: got %q, want ナルト", r.Title)
		}
		if r.ResultID != "40" {
			t.Errorf("resultID: got %q, want 40", r.ResultID)
		}
	})
}
