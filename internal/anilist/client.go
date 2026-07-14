package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/vrsandeep/mango-go/internal/metadata"
)

const (
	graphqlURL     = "https://graphql.anilist.co"
	maxQueryLength = 400
)

// Media represents minimal AniList fields for the legacy search handler.
// Deprecated: use MediaFull with Client.SearchMediaFull instead.
type Media struct {
	ID         int64       `json:"id"`
	SiteURL    string      `json:"siteUrl"`
	Title      *Title      `json:"title"`
	CoverImage *CoverImage `json:"coverImage"`
}

type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
}

type CoverImage struct {
	Large string `json:"large"`
}

type MediaTitle struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

type MediaTag struct {
	Name           string `json:"name"`
	Rank           *int   `json:"rank"`
	IsMediaSpoiler bool   `json:"isMediaSpoiler"`
}

type MediaStaffName struct {
	Full string `json:"full"`
}

type MediaStaffNode struct {
	Name MediaStaffName `json:"name"`
}

type MediaStaffEdge struct {
	Role string         `json:"role"`
	Node MediaStaffNode `json:"node"`
}

type MediaStaff struct {
	Edges []MediaStaffEdge `json:"edges"`
}

type MediaDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

type MediaExternalLink struct {
	URL  string `json:"url"`
	Site string `json:"site"`
	Type string `json:"type"`
}

type MediaCoverImage struct {
	Large string `json:"large"`
}

type MediaFull struct {
	ID              int64               `json:"id"`
	Status          string              `json:"status"`
	Title           MediaTitle          `json:"title"`
	Description     string              `json:"description"`
	Genres          []string            `json:"genres"`
	Tags            []MediaTag          `json:"tags"`
	Staff           MediaStaff          `json:"staff"`
	StartDate       MediaDate           `json:"startDate"`
	Volumes         *int                `json:"volumes"`
	AverageScore    *int                `json:"averageScore"`
	IsAdult         bool                `json:"isAdult"`
	CountryOfOrigin string              `json:"countryOfOrigin"`
	ExternalLinks   []MediaExternalLink `json:"externalLinks"`
	SiteURL         string              `json:"siteUrl"`
	CoverImage      MediaCoverImage     `json:"coverImage"`
}

type Client struct {
	httpClient *metadata.RetryClient
	BaseURL    string
}

func NewClient(httpClient *metadata.RetryClient) *Client {
	return &Client{
		httpClient: httpClient,
		BaseURL:    graphqlURL,
	}
}

type graphqlResponse struct {
	Data *struct {
		Media *Media `json:"Media"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type searchMediaFullResponse struct {
	Data *struct {
		Page *struct {
			Media []MediaFull `json:"media"`
		} `json:"Page"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type getMediaFullResponse struct {
	Data *struct {
		Media *MediaFull `json:"Media"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

const searchMediaFullQuery = `query ($search: String, $perPage: Int) {
  Page(perPage: $perPage) {
    media(search: $search, type: MANGA) {
      id status
      title { romaji english native }
      description(asHtml: false)
      genres
      tags { name rank isMediaSpoiler }
      staff { edges { role node { name { full } } } }
      startDate { year month day }
      volumes averageScore isAdult countryOfOrigin
      externalLinks { url site type }
      siteUrl
      coverImage { large }
    }
  }
}`

const getMediaFullQuery = `query ($id: Int) {
  Media(id: $id, type: MANGA) {
    id status
    title { romaji english native }
    description(asHtml: false)
    genres
    tags { name rank isMediaSpoiler }
    staff { edges { role node { name { full } } } }
    startDate { year month day }
    volumes averageScore isAdult countryOfOrigin
    externalLinks { url site type }
    siteUrl
    coverImage { large }
  }
}`

// SearchManga searches AniList for a manga by title and returns the first match.
// Deprecated: use Client.SearchMediaFull for retry-aware searches with full metadata.
func SearchManga(title string) (*Media, error) {
	cleanTitle := cleanTitleForSearch(title)
	if cleanTitle == "" {
		return nil, nil
	}

	query := `query ($search: String) {
  Media(search: $search, type: MANGA) {
    id
    title { romaji english }
    coverImage { large }
    siteUrl
  }
}`
	body := map[string]any{
		"query":     query,
		"variables": map[string]string{"search": cleanTitle},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, graphqlURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("AniList API error: %s", resp.Status)
		return nil, fmt.Errorf("anilist API returned %s", resp.Status)
	}

	var out graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		log.Printf("AniList API errors: %v", out.Errors)
		return nil, nil
	}
	if out.Data == nil || out.Data.Media == nil {
		return nil, nil
	}
	return out.Data.Media, nil
}

var nonWordRegexp = regexp.MustCompile(`[^\w\s-]`)

func cleanTitleForSearch(title string) string {
	s := nonWordRegexp.ReplaceAllString(title, "")
	s = strings.TrimSpace(s)
	return s
}

// SearchMediaFull searches AniList for manga matching the query.
// Returns an empty slice when there are no results.
// Queries longer than 400 runes are truncated on rune boundaries.
func (c *Client) SearchMediaFull(ctx context.Context, query string, limit int) ([]MediaFull, error) {
	if utf8.RuneCountInString(query) > maxQueryLength {
		runes := []rune(query)
		query = string(runes[:maxQueryLength])
	}

	body := map[string]any{
		"query": searchMediaFullQuery,
		"variables": map[string]any{
			"search":  query,
			"perPage": limit,
		},
	}

	var out searchMediaFullResponse
	if err := c.doGraphQL(ctx, body, &out); err != nil {
		return nil, err
	}

	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("anilist graphql error: %s", out.Errors[0].Message)
	}

	if out.Data == nil || out.Data.Page == nil || out.Data.Page.Media == nil {
		return []MediaFull{}, nil
	}

	return out.Data.Page.Media, nil
}

// GetMediaFull fetches a single manga by AniList ID with all metadata fields.
func (c *Client) GetMediaFull(ctx context.Context, id int) (*MediaFull, error) {
	body := map[string]any{
		"query": getMediaFullQuery,
		"variables": map[string]any{
			"id": id,
		},
	}

	var out getMediaFullResponse
	if err := c.doGraphQL(ctx, body, &out); err != nil {
		return nil, err
	}

	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("anilist graphql error: %s", out.Errors[0].Message)
	}

	if out.Data == nil || out.Data.Media == nil {
		return nil, fmt.Errorf("anilist: media with id %d not found", id)
	}

	return out.Data.Media, nil
}

func (c *Client) doGraphQL(ctx context.Context, body map[string]any, target any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anilist API returned %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode graphql response: %w", err)
	}

	return nil
}
