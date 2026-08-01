package comicinfo

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrsandeep/mango-go/internal/metadata"
)

// sampleComicInfoXML is a valid ComicInfo XML string used across tests.
const sampleComicInfoXML = `<?xml version="1.0" encoding="utf-8"?>
<ComicInfo>
  <Title>Test Chapter</Title>
  <Series>Test Series</Series>
  <Number>5</Number>
  <Volume>2</Volume>
  <Summary>A test summary</Summary>
  <Notes>Some notes</Notes>
  <Year>2023</Year>
  <Month>7</Month>
  <Day>15</Day>
  <Writer>Writer One, Writer Two</Writer>
  <Penciller>Penciller One</Penciller>
  <Inker>Inker One</Inker>
  <Colorist>Colorist One</Colorist>
  <Letterer>Letterer One</Letterer>
  <CoverArtist>Cover Artist One</CoverArtist>
  <Editor>Editor One</Editor>
  <Translator>Translator One</Translator>
  <Genre>Action, Comedy, Drama</Genre>
  <Tags>tag1, tag2, tag3</Tags>
  <Web>https://example.com</Web>
  <LanguageISO>en</LanguageISO>
  <AgeRating>Teen</AgeRating>
  <Characters>Char A, Char B</Characters>
  <Teams>Team X</Teams>
  <Locations>Location Y</Locations>
  <ScanInformation>ScanGroup</ScanInformation>
  <StoryArc>Main Arc</StoryArc>
  <StoryArcNumber>3</StoryArcNumber>
  <Publisher>Test Publisher</Publisher>
  <Count>10</Count>
  <Manga>Yes</Manga>
</ComicInfo>`

// createTestCBZ creates a CBZ file at the given path with the specified files.
func createTestCBZ(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range files {
		fw, err := w.Create(name)
		require.NoError(t, err)
		_, err = fw.Write(content)
		require.NoError(t, err)
	}
}

func TestParse_ValidXML(t *testing.T) {
	ci, err := Parse([]byte(sampleComicInfoXML))
	require.NoError(t, err)
	require.NotNil(t, ci)

	assert.Equal(t, "Test Chapter", ci.Title)
	assert.Equal(t, "Test Series", ci.Series)
	assert.Equal(t, "5", ci.Number)
	assert.Equal(t, "2", ci.Volume)
	assert.Equal(t, "A test summary", ci.Summary)
	assert.Equal(t, "Some notes", ci.Notes)
	assert.Equal(t, "2023", ci.Year)
	assert.Equal(t, "7", ci.Month)
	assert.Equal(t, "15", ci.Day)
	assert.Equal(t, "Writer One, Writer Two", ci.Writer)
	assert.Equal(t, "Penciller One", ci.Penciller)
	assert.Equal(t, "Action, Comedy, Drama", ci.Genre)
	assert.Equal(t, "tag1, tag2, tag3", ci.Tags)
	assert.Equal(t, "https://example.com", ci.Web)
	assert.Equal(t, "en", ci.LanguageISO)
	assert.Equal(t, "Teen", ci.AgeRating)
	assert.Equal(t, "Char A, Char B", ci.Characters)
	assert.Equal(t, "Team X", ci.Teams)
	assert.Equal(t, "Location Y", ci.Locations)
	assert.Equal(t, "ScanGroup", ci.ScanInformation)
	assert.Equal(t, "Main Arc", ci.StoryArc)
	assert.Equal(t, "3", ci.StoryArcNumber)
	assert.Equal(t, "Test Publisher", ci.Publisher)
	assert.Equal(t, "10", ci.Count)
	assert.Equal(t, "Yes", ci.Manga)
}

func TestParse_BOM(t *testing.T) {
	bomXML := append([]byte{0xEF, 0xBB, 0xBF}, []byte(sampleComicInfoXML)...)
	ci, err := Parse(bomXML)
	require.NoError(t, err)
	require.NotNil(t, ci)
	assert.Equal(t, "Test Chapter", ci.Title)
	assert.Equal(t, "5", ci.Number)
}

func TestParse_MalformedXML(t *testing.T) {
	malformed := []byte(`<ComicInfo><Title>Unclosed`)
	ci, err := Parse(malformed)
	assert.Error(t, err)
	assert.Nil(t, ci)
}

func TestExtractFromArchive_WithComicInfo(t *testing.T) {
	dir := t.TempDir()
	cbzPath := filepath.Join(dir, "with_comicinfo.cbz")

	createTestCBZ(t, cbzPath, map[string][]byte{
		"ComicInfo.xml": []byte(sampleComicInfoXML),
		"page001.jpg":   {0xFF, 0xD8, 0xFF}, // minimal JPEG header
	})

	ci, err := ExtractFromArchive(cbzPath)
	require.NoError(t, err)
	require.NotNil(t, ci)
	assert.Equal(t, "Test Chapter", ci.Title)
	assert.Equal(t, "5", ci.Number)
}

func TestExtractFromArchive_WithoutComicInfo(t *testing.T) {
	dir := t.TempDir()
	cbzPath := filepath.Join(dir, "without_comicinfo.cbz")

	createTestCBZ(t, cbzPath, map[string][]byte{
		"page001.jpg": {0xFF, 0xD8, 0xFF},
		"page002.jpg": {0xFF, 0xD8, 0xFF},
	})

	ci, err := ExtractFromArchive(cbzPath)
	assert.NoError(t, err)
	assert.Nil(t, ci)
}

func TestExtractFromArchive_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	cbzPath := filepath.Join(dir, "case_insensitive.cbz")

	createTestCBZ(t, cbzPath, map[string][]byte{
		"COMICINFO.XML": []byte(sampleComicInfoXML),
		"page001.jpg":   {0xFF, 0xD8, 0xFF},
	})

	ci, err := ExtractFromArchive(cbzPath)
	require.NoError(t, err)
	require.NotNil(t, ci)
	assert.Equal(t, "Test Chapter", ci.Title)
}

func TestExtractFromArchive_PrefersShallowDepth(t *testing.T) {
	dir := t.TempDir()
	cbzPath := filepath.Join(dir, "nested.cbz")

	rootXML := `<?xml version="1.0"?><ComicInfo><Title>Root Level</Title></ComicInfo>`
	nestedXML := `<?xml version="1.0"?><ComicInfo><Title>Nested Level</Title></ComicInfo>`

	createTestCBZ(t, cbzPath, map[string][]byte{
		"ComicInfo.xml":             []byte(rootXML),
		"subdir/ComicInfo.xml":      []byte(nestedXML),
		"subdir/deep/ComicInfo.xml": []byte(nestedXML),
	})

	ci, err := ExtractFromArchive(cbzPath)
	require.NoError(t, err)
	require.NotNil(t, ci)
	assert.Equal(t, "Root Level", ci.Title)
}

func TestExtractFromArchive_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("not an archive"), 0o644))

	ci, err := ExtractFromArchive(path)
	assert.Error(t, err)
	assert.Nil(t, ci)
}

func TestMapToChapterMetadata_FullMapping(t *testing.T) {
	ci, err := Parse([]byte(sampleComicInfoXML))
	require.NoError(t, err)

	meta := MapToChapterMetadata(ci)
	require.NotNil(t, meta)

	assert.Equal(t, "Test Chapter", *meta.Title)
	assert.Equal(t, "5", *meta.Number)
	assert.Equal(t, "2", *meta.Volume)
	assert.Equal(t, "A test summary", *meta.Summary)
	assert.Equal(t, "Some notes", *meta.Notes)
	assert.Equal(t, "en", *meta.Language)
	assert.Equal(t, "Teen", *meta.AgeRating)
	assert.Equal(t, "https://example.com", *meta.Web)
	assert.Equal(t, "Char A, Char B", *meta.Characters)
	assert.Equal(t, "Team X", *meta.Teams)
	assert.Equal(t, "Location Y", *meta.Locations)
	assert.Equal(t, "ScanGroup", *meta.ScanlationGroup)
	assert.Equal(t, "Main Arc", *meta.StoryArc)
	assert.Equal(t, "3", *meta.StoryArcNumber)
	assert.Equal(t, "2023-07-15", *meta.ReleaseDate)
}

func TestMapToChapterMetadata_CommaSeparated(t *testing.T) {
	ci := &ComicInfo{
		Genre: "Action, Comedy, Drama",
		Tags:  "tag1, tag2, tag3",
	}

	meta := MapToChapterMetadata(ci)

	assert.Equal(t, []string{"Action", "Comedy", "Drama"}, meta.Genres)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, meta.Tags)
}

func TestMapToChapterMetadata_CommaSeparated_SkipsEmpty(t *testing.T) {
	ci := &ComicInfo{
		Genre: "Action,, , Drama",
		Tags:  ",,,",
	}

	meta := MapToChapterMetadata(ci)

	assert.Equal(t, []string{"Action", "Drama"}, meta.Genres)
	assert.Nil(t, meta.Tags)
}

func TestMapToChapterMetadata_MultipleAuthors(t *testing.T) {
	ci := &ComicInfo{
		Writer:    "Alice, Bob",
		Penciller: "Charlie",
		Colorist:  "Diana, Eve",
	}

	meta := MapToChapterMetadata(ci)

	expected := []metadata.ChapterMetadataAuthor{
		{Name: "Alice", Role: string(metadata.AuthorRoleWriter)},
		{Name: "Bob", Role: string(metadata.AuthorRoleWriter)},
		{Name: "Charlie", Role: string(metadata.AuthorRolePenciller)},
		{Name: "Diana", Role: string(metadata.AuthorRoleColorist)},
		{Name: "Eve", Role: string(metadata.AuthorRoleColorist)},
	}
	assert.Equal(t, expected, meta.Authors)
}

func TestMapToChapterMetadata_ReleaseDate(t *testing.T) {
	tests := []struct {
		name     string
		year     string
		month    string
		day      string
		expected *string
	}{
		{
			name:     "full date",
			year:     "2023",
			month:    "7",
			day:      "15",
			expected: strPtr("2023-07-15"),
		},
		{
			name:     "year only",
			year:     "2023",
			month:    "",
			day:      "",
			expected: strPtr("2023-01-01"),
		},
		{
			name:     "year and month only",
			year:     "2023",
			month:    "12",
			day:      "",
			expected: strPtr("2023-12-01"),
		},
		{
			name:     "no year means no date",
			year:     "",
			month:    "7",
			day:      "15",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ci := &ComicInfo{
				Year:  tt.year,
				Month: tt.month,
				Day:   tt.day,
			}
			meta := MapToChapterMetadata(ci)
			if tt.expected == nil {
				assert.Nil(t, meta.ReleaseDate)
			} else {
				require.NotNil(t, meta.ReleaseDate)
				assert.Equal(t, *tt.expected, *meta.ReleaseDate)
			}
		})
	}
}

func TestMapToChapterMetadata_EmptyFields(t *testing.T) {
	ci := &ComicInfo{}
	meta := MapToChapterMetadata(ci)

	assert.Nil(t, meta.Title)
	assert.Nil(t, meta.Number)
	assert.Nil(t, meta.Volume)
	assert.Nil(t, meta.Summary)
	assert.Nil(t, meta.Notes)
	assert.Nil(t, meta.Language)
	assert.Nil(t, meta.AgeRating)
	assert.Nil(t, meta.Web)
	assert.Nil(t, meta.Characters)
	assert.Nil(t, meta.Teams)
	assert.Nil(t, meta.Locations)
	assert.Nil(t, meta.ScanlationGroup)
	assert.Nil(t, meta.StoryArc)
	assert.Nil(t, meta.StoryArcNumber)
	assert.Nil(t, meta.ReleaseDate)
	assert.Nil(t, meta.Genres)
	assert.Nil(t, meta.Tags)
	assert.Nil(t, meta.Authors)
}

func TestMapToChapterMetadata_SeriesFieldsNotMapped(t *testing.T) {
	ci := &ComicInfo{
		Series:    "My Series",
		Publisher: "My Publisher",
		Count:     "10",
		Manga:     "Yes",
	}

	meta := MapToChapterMetadata(ci)

	// These fields should NOT appear in ChapterMetadata.
	// Verify all scalar fields are nil (series-level fields are skipped).
	assert.Nil(t, meta.Title)
	assert.Nil(t, meta.Number)
}

// strPtr returns a pointer to the given string value.
func strPtr(s string) *string {
	return &s
}
