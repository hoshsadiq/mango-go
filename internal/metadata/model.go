package metadata

// AuthorRole represents the role of an author/artist in a series
type AuthorRole string

const (
	AuthorRoleWriter      AuthorRole = "WRITER"
	AuthorRolePenciller   AuthorRole = "PENCILLER"
	AuthorRoleInker       AuthorRole = "INKER"
	AuthorRoleColorist    AuthorRole = "COLORIST"
	AuthorRoleLetterer    AuthorRole = "LETTERER"
	AuthorRoleCoverArtist AuthorRole = "COVER_ARTIST"
	AuthorRoleEditor      AuthorRole = "EDITOR"
	AuthorRoleTranslator  AuthorRole = "TRANSLATOR"
)

// SeriesStatus represents the publication status of a series
type SeriesStatus string

const (
	SeriesStatusOngoing   SeriesStatus = "ONGOING"
	SeriesStatusCompleted SeriesStatus = "COMPLETED"
	SeriesStatusAbandoned SeriesStatus = "ABANDONED"
	SeriesStatusHiatus    SeriesStatus = "HIATUS"
)

// ReadingDirection represents the reading direction of a series
type ReadingDirection string

const (
	ReadingDirectionLeftToRight ReadingDirection = "LEFT_TO_RIGHT"
	ReadingDirectionRightToLeft ReadingDirection = "RIGHT_TO_LEFT"
	ReadingDirectionVertical    ReadingDirection = "VERTICAL"
	ReadingDirectionWebtoon     ReadingDirection = "WEBTOON"
)

// Author represents an author or artist with their role
type Author struct {
	Name string     `json:"name"`
	Role AuthorRole `json:"role"`
}

// WebLink represents an external link to a series
type WebLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// SeriesTitle represents a title in a specific language and type
type SeriesTitle struct {
	Title    string `json:"title"`
	Type     string `json:"type"` // ROMAJI, LOCALIZED, NATIVE
	Language string `json:"language"`
}

// SeriesMetadata contains comprehensive metadata about a manga series
// All scalar fields are pointers: nil = absent, pointer to zero-value = explicitly set
// All collection fields are slices: nil = absent, empty slice = explicitly empty
type SeriesMetadata struct {
	Status           *SeriesStatus     `json:"status"`
	Title            *string           `json:"title"`
	Titles           []SeriesTitle     `json:"titles"`
	Summary          *string           `json:"summary"`
	Publisher        *string           `json:"publisher"`
	ReadingDirection *ReadingDirection `json:"reading_direction"`
	AgeRating        *int              `json:"age_rating"`
	Language         *string           `json:"language"`
	Genres           []string          `json:"genres"`
	Tags             []string          `json:"tags"`
	TotalBookCount   *int              `json:"total_book_count"`
	Authors          []Author          `json:"authors"`
	ReleaseYear      *int              `json:"release_year"`
	ReleaseMonth     *int              `json:"release_month"`
	ReleaseDay       *int              `json:"release_day"`
	Links            []WebLink         `json:"links"`
	CommunityScore   *float64          `json:"community_score"`
	ThumbnailURL     *string           `json:"thumbnail_url"`
}

// BookMetadata contains metadata about a specific book/volume in a series
// All scalar fields are pointers: nil = absent, pointer to zero-value = explicitly set
// All collection fields are slices: nil = absent, empty slice = explicitly empty
type BookMetadata struct {
	Title        *string   `json:"title"`
	Summary      *string   `json:"summary"`
	Number       *string   `json:"number"`
	NumberSort   *float64  `json:"number_sort"`
	ReleaseYear  *int      `json:"release_year"`
	ReleaseMonth *int      `json:"release_month"`
	ReleaseDay   *int      `json:"release_day"`
	Authors      []Author  `json:"authors"`
	Tags         []string  `json:"tags"`
	ISBN         *string   `json:"isbn"`
	Links        []WebLink `json:"links"`
	ThumbnailURL *string   `json:"thumbnail_url"`
}

// SeriesSearchResult represents a search result from a metadata provider
// All fields are non-pointer and always populated by providers
type SeriesSearchResult struct {
	URL          string `json:"url"`
	ImageURL     string `json:"image_url"`
	Title        string `json:"title"`
	ProviderName string `json:"provider_name"`
	ResultID     string `json:"result_id"`
}
