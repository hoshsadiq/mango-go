package metadata

import "time"

// ChapterMetadata holds all metadata for a single chapter.
// nil = absent (not set), pointer to zero-value = explicitly set to empty.
type ChapterMetadata struct {
	ID        int64
	ChapterID int64

	// Scalar fields — nil means "not set"
	Title           *string
	Number          *string
	SortNumber      *float64
	Volume          *string
	Summary         *string
	Notes           *string
	ReleaseDate     *string
	Language        *string
	ChapterType     *string
	AgeRating       *string
	Web             *string
	Characters      *string
	Teams           *string
	Locations       *string
	ScanlationGroup *string
	StoryArc        *string
	StoryArcNumber  *string

	// Collection fields
	Authors []ChapterMetadataAuthor
	Genres  []string
	Tags    []string

	// Lock fields — true means "locked, do not overwrite"
	TitleLock           bool
	NumberLock          bool
	SortNumberLock      bool
	VolumeLock          bool
	SummaryLock         bool
	NotesLock           bool
	ReleaseDateLock     bool
	LanguageLock        bool
	ChapterTypeLock     bool
	AgeRatingLock       bool
	WebLock             bool
	CharactersLock      bool
	TeamsLock           bool
	LocationsLock       bool
	ScanlationGroupLock bool
	StoryArcLock        bool
	StoryArcNumberLock  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChapterMetadataAuthor represents an author/creator of a chapter.
type ChapterMetadataAuthor struct {
	ID   int64
	Name string
	Role string
}

// ChapterType constants for the chapter_type field.
const (
	ChapterTypeRegular = "Regular"
	ChapterTypeSpecial = "Special"
	ChapterTypeBonus   = "Bonus"
	ChapterTypeOmake   = "Omake"
	ChapterTypeOneShot = "One-Shot"
	ChapterTypeAnnual  = "Annual"
	ChapterTypeOmnibus = "Omnibus"
)

// ChapterMetadataLocks mirrors the lock fields for use in PATCH /locks endpoint.
type ChapterMetadataLocks struct {
	TitleLock           bool `json:"title_lock"`
	NumberLock          bool `json:"number_lock"`
	SortNumberLock      bool `json:"sort_number_lock"`
	VolumeLock          bool `json:"volume_lock"`
	SummaryLock         bool `json:"summary_lock"`
	NotesLock           bool `json:"notes_lock"`
	ReleaseDateLock     bool `json:"release_date_lock"`
	LanguageLock        bool `json:"language_lock"`
	ChapterTypeLock     bool `json:"chapter_type_lock"`
	AgeRatingLock       bool `json:"age_rating_lock"`
	WebLock             bool `json:"web_lock"`
	CharactersLock      bool `json:"characters_lock"`
	TeamsLock           bool `json:"teams_lock"`
	LocationsLock       bool `json:"locations_lock"`
	ScanlationGroupLock bool `json:"scanlation_group_lock"`
	StoryArcLock        bool `json:"story_arc_lock"`
	StoryArcNumberLock  bool `json:"story_arc_number_lock"`
}
