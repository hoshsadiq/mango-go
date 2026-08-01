package comicinfo

import "encoding/xml"

// ComicInfo represents the ComicInfo.xml schema used in comic archives.
// All fields are strings; numeric fields (Year, Month, Day, Volume, Number) are
// kept as strings to avoid parse errors on malformed data.
type ComicInfo struct {
	XMLName         xml.Name `xml:"ComicInfo"`
	Title           string   `xml:"Title"`
	Series          string   `xml:"Series"`
	Number          string   `xml:"Number"`
	Volume          string   `xml:"Volume"`
	Summary         string   `xml:"Summary"`
	Notes           string   `xml:"Notes"`
	Year            string   `xml:"Year"`
	Month           string   `xml:"Month"`
	Day             string   `xml:"Day"`
	Writer          string   `xml:"Writer"`
	Penciller       string   `xml:"Penciller"`
	Inker           string   `xml:"Inker"`
	Colorist        string   `xml:"Colorist"`
	Letterer        string   `xml:"Letterer"`
	CoverArtist     string   `xml:"CoverArtist"`
	Editor          string   `xml:"Editor"`
	Translator      string   `xml:"Translator"`
	Genre           string   `xml:"Genre"`
	Tags            string   `xml:"Tags"`
	Web             string   `xml:"Web"`
	LanguageISO     string   `xml:"LanguageISO"`
	Format          string   `xml:"Format"`
	AgeRating       string   `xml:"AgeRating"`
	Characters      string   `xml:"Characters"`
	Teams           string   `xml:"Teams"`
	Locations       string   `xml:"Locations"`
	ScanInformation string   `xml:"ScanInformation"`
	StoryArc        string   `xml:"StoryArc"`
	StoryArcNumber  string   `xml:"StoryArcNumber"`
	Publisher       string   `xml:"Publisher"`
	Count           string   `xml:"Count"`
	Manga           string   `xml:"Manga"`
}
