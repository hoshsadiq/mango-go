package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vrsandeep/mango-go/internal/metadata"
)

const maxQueryLength = 400

// MediaFull represents the expanded AniList Media fields for full metadata retrieval.
type MediaFull struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Title  struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Tags        []struct {
		Name           string `json:"name"`
		Rank           *int   `json:"rank"`
		IsMediaSpoiler bool   `json:"isMediaSpoiler"`
	} `json:"tags"`
	Staff struct {
		Edges []struct {
			Role string `json:"role"`
			Node struct {
				Name struct {
					Full string `json:"full"`
				} `json:"name"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"staff"`
	StartDate struct {
		Year  *int `json:"year"`
		Month *int `json:"month"`
		Day   *int `json:"day"`
	} `json:"startDate"`
	Volumes         *int   `json:"volumes"`
	AverageScore    *int   `json:"averageScore"`
	IsAdult         bool   `json:"isAdult"`
	CountryOfOrigin string `json:"countryOfOrigin"`
	ExternalLinks   []struct {
		URL  string `json:"url"`
		Site string `json:"site"`
		Type string `json:"type"`
	} `json:"externalLinks"`
	SiteUrl    string `json:"siteUrl"`
	CoverImage struct {
		Large string `json:"large"`
	} `json:"coverImage"`
}

// Client wraps the retry-aware HTTP client for AniList GraphQL API calls.
type Client struct {
	httpClient *metadata.RetryClient
	// BaseURL is the AniList GraphQL endpoint. Exported for test overrides.
	BaseURL string
}

// NewClient creates a new AniList client using the provided retry-aware HTTP client.
func NewClient(httpClient *metadata.RetryClient) *Client {
	return &Client{
		httpClient: httpClient,
		BaseURL:    graphqlURL,
	}
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

// SearchMediaFull searches AniList for manga with full metadata fields.
// Returns an empty slice (not nil) when there are no results.
// The query string is truncated to 400 characters before sending.
func (c *Client) SearchMediaFull(ctx context.Context, query string, limit int) ([]MediaFull, error) {
	if len(query) > maxQueryLength {
		query = query[:maxQueryLength]
	}

	body := map[string]interface{}{
		"query": searchMediaFullQuery,
		"variables": map[string]interface{}{
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

// GetMediaFull fetches a single manga by AniList ID with full metadata fields.
func (c *Client) GetMediaFull(ctx context.Context, id int) (*MediaFull, error) {
	body := map[string]interface{}{
		"query": getMediaFullQuery,
		"variables": map[string]interface{}{
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

// doGraphQL sends a GraphQL POST request and decodes the JSON response into target.
func (c *Client) doGraphQL(ctx context.Context, body map[string]interface{}, target interface{}) error {
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
