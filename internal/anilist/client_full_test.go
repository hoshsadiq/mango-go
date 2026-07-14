package anilist_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/metadata"
)

// newTestClient creates an anilist.Client backed by a mock HTTP server.
func newTestClient(t *testing.T, handler http.HandlerFunc) *anilist.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	rc := metadata.NewRetryClient(
		metadata.WithMaxRetries(0),
	)

	client := anilist.NewClient(rc)
	client.BaseURL = server.URL
	return client
}

const sampleSearchResponse = `{
  "data": {
    "Page": {
      "media": [
        {
          "id": 30013,
          "status": "FINISHED",
          "title": {"romaji": "One Piece", "english": "One Piece", "native": "ワンピース"},
          "description": "A grand adventure.",
          "genres": ["Action", "Adventure", "Comedy"],
          "tags": [
            {"name": "Shounen", "rank": 92, "isMediaSpoiler": false},
            {"name": "Pirates", "rank": 88, "isMediaSpoiler": true}
          ],
          "staff": {
            "edges": [
              {"role": "Story & Art", "node": {"name": {"full": "Eiichiro Oda"}}}
            ]
          },
          "startDate": {"year": 1997, "month": 7, "day": 22},
          "volumes": null,
          "averageScore": 88,
          "isAdult": false,
          "countryOfOrigin": "JP",
          "externalLinks": [{"url": "https://example.com", "site": "Official", "type": "INFO"}],
          "siteUrl": "https://anilist.co/manga/30013",
          "coverImage": {"large": "https://img.anilist.co/large.jpg"}
        },
        {
          "id": 30014,
          "status": "RELEASING",
          "title": {"romaji": "Naruto", "english": "Naruto", "native": "ナルト"},
          "description": "A ninja story.",
          "genres": ["Action"],
          "tags": [],
          "staff": {"edges": []},
          "startDate": {"year": 1999, "month": 9, "day": null},
          "volumes": 72,
          "averageScore": 79,
          "isAdult": false,
          "countryOfOrigin": "JP",
          "externalLinks": [],
          "siteUrl": "https://anilist.co/manga/30014",
          "coverImage": {"large": "https://img.anilist.co/naruto.jpg"}
        }
      ]
    }
  }
}`

func TestSearchMediaFull_ParsesResults(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleSearchResponse))
	})

	results, err := client.SearchMediaFull(context.Background(), "One Piece", 10)
	if err != nil {
		t.Fatalf("SearchMediaFull: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	m := results[0]
	assertEqual(t, "ID", m.ID, 30013)
	assertEqual(t, "Status", m.Status, "FINISHED")
	assertEqual(t, "Title.Romaji", m.Title.Romaji, "One Piece")
	assertEqual(t, "Title.English", m.Title.English, "One Piece")
	assertEqual(t, "Title.Native", m.Title.Native, "ワンピース")
	assertEqual(t, "Description", m.Description, "A grand adventure.")
	assertEqual(t, "len(Genres)", len(m.Genres), 3)
	assertEqual(t, "Genres[0]", m.Genres[0], "Action")
	assertEqual(t, "len(Tags)", len(m.Tags), 2)
	assertEqual(t, "Tags[0].Name", m.Tags[0].Name, "Shounen")
	assertIntPtr(t, "Tags[0].Rank", m.Tags[0].Rank, 92)
	assertEqual(t, "Tags[0].IsMediaSpoiler", m.Tags[0].IsMediaSpoiler, false)
	assertEqual(t, "Tags[1].IsMediaSpoiler", m.Tags[1].IsMediaSpoiler, true)
	assertEqual(t, "len(Staff.Edges)", len(m.Staff.Edges), 1)
	assertEqual(t, "Staff.Edges[0].Role", m.Staff.Edges[0].Role, "Story & Art")
	assertEqual(t, "Staff.Edges[0].Node.Name.Full", m.Staff.Edges[0].Node.Name.Full, "Eiichiro Oda")
	assertIntPtr(t, "StartDate.Year", m.StartDate.Year, 1997)
	assertIntPtr(t, "StartDate.Month", m.StartDate.Month, 7)
	assertIntPtr(t, "StartDate.Day", m.StartDate.Day, 22)
	if m.Volumes != nil {
		t.Errorf("Volumes = %d, want nil", *m.Volumes)
	}
	assertIntPtr(t, "AverageScore", m.AverageScore, 88)
	assertEqual(t, "IsAdult", m.IsAdult, false)
	assertEqual(t, "CountryOfOrigin", m.CountryOfOrigin, "JP")
	assertEqual(t, "len(ExternalLinks)", len(m.ExternalLinks), 1)
	assertEqual(t, "ExternalLinks[0].URL", m.ExternalLinks[0].URL, "https://example.com")
	assertEqual(t, "ExternalLinks[0].Site", m.ExternalLinks[0].Site, "Official")
	assertEqual(t, "ExternalLinks[0].Type", m.ExternalLinks[0].Type, "INFO")
	assertEqual(t, "SiteURL", m.SiteURL, "https://anilist.co/manga/30013")
	assertEqual(t, "CoverImage.Large", m.CoverImage.Large, "https://img.anilist.co/large.jpg")

	m2 := results[1]
	assertIntPtr(t, "results[1].Volumes", m2.Volumes, 72)
	if m2.StartDate.Day != nil {
		t.Errorf("results[1].StartDate.Day = %d, want nil", *m2.StartDate.Day)
	}
}

func TestSearchMediaFull_EmptyResults(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": {"Page": {"media": []}}}`))
	})

	results, err := client.SearchMediaFull(context.Background(), "nonexistent", 10)
	if err != nil {
		t.Fatalf("SearchMediaFull: %v", err)
	}
	if results == nil {
		t.Fatal("got nil, want empty slice")
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestSearchMediaFull_NullMediaReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": {"Page": {"media": null}}}`))
	})

	results, err := client.SearchMediaFull(context.Background(), "test", 10)
	if err != nil {
		t.Fatalf("SearchMediaFull: %v", err)
	}
	if results == nil {
		t.Fatal("got nil, want empty slice")
	}
}

func TestSearchMediaFull_QueryTruncation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		query string
	}{
		{"ASCII", strings.Repeat("a", 500)},
		{"CJK", strings.Repeat("漫", 500)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedSearch string
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var req struct {
					Variables struct {
						Search string `json:"search"`
					} `json:"variables"`
				}
				json.Unmarshal(body, &req)
				receivedSearch = req.Variables.Search

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"data": {"Page": {"media": []}}}`))
			})

			_, err := client.SearchMediaFull(context.Background(), tt.query, 10)
			if err != nil {
				t.Fatalf("SearchMediaFull: %v", err)
			}
			if got := utf8.RuneCountInString(receivedSearch); got != 400 {
				t.Errorf("received query rune count = %d, want 400", got)
			}
		})
	}
}

func TestSearchMediaFull_ShortQueryNotTruncated(t *testing.T) {
	t.Parallel()
	var receivedSearch string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Variables struct {
				Search string `json:"search"`
			} `json:"variables"`
		}
		json.Unmarshal(body, &req)
		receivedSearch = req.Variables.Search

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": {"Page": {"media": []}}}`))
	})

	_, err := client.SearchMediaFull(context.Background(), "One Piece", 10)
	if err != nil {
		t.Fatalf("SearchMediaFull: %v", err)
	}
	assertEqual(t, "receivedSearch", receivedSearch, "One Piece")
}

func TestSearchMediaFull_GraphQLError(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errors": [{"message": "validation failed"}]}`))
	})

	_, err := client.SearchMediaFull(context.Background(), "test", 10)
	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error = %q, want to contain 'validation failed'", err)
	}
}

func TestSearchMediaFull_HTTPError(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.SearchMediaFull(context.Background(), "test", 10)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestGetMediaFull_ParsesSingleResult(t *testing.T) {
	t.Parallel()
	response := `{
	  "data": {
	    "Media": {
	      "id": 30013,
	      "status": "FINISHED",
	      "title": {"romaji": "One Piece", "english": "One Piece", "native": "ワンピース"},
	      "description": "Adventure on the seas.",
	      "genres": ["Action"],
	      "tags": [{"name": "Shounen", "rank": 92, "isMediaSpoiler": false}],
	      "staff": {"edges": [{"role": "Story & Art", "node": {"name": {"full": "Eiichiro Oda"}}}]},
	      "startDate": {"year": 1997, "month": 7, "day": 22},
	      "volumes": null,
	      "averageScore": 88,
	      "isAdult": false,
	      "countryOfOrigin": "JP",
	      "externalLinks": [],
	      "siteUrl": "https://anilist.co/manga/30013",
	      "coverImage": {"large": "https://img.anilist.co/large.jpg"}
	    }
	  }
	}`

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	m, err := client.GetMediaFull(context.Background(), 30013)
	if err != nil {
		t.Fatalf("GetMediaFull: %v", err)
	}
	if m == nil {
		t.Fatal("got nil MediaFull")
	}
	assertEqual(t, "ID", m.ID, 30013)
	assertEqual(t, "Status", m.Status, "FINISHED")
	assertEqual(t, "Title.English", m.Title.English, "One Piece")
	assertEqual(t, "Title.Native", m.Title.Native, "ワンピース")
	assertEqual(t, "Description", m.Description, "Adventure on the seas.")
	assertEqual(t, "Staff[0].Node.Name.Full", m.Staff.Edges[0].Node.Name.Full, "Eiichiro Oda")
	assertIntPtr(t, "StartDate.Year", m.StartDate.Year, 1997)
	assertEqual(t, "CoverImage.Large", m.CoverImage.Large, "https://img.anilist.co/large.jpg")
}

func TestGetMediaFull_GraphQLError(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errors": [{"message": "not found"}]}`))
	})

	_, err := client.GetMediaFull(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err)
	}
}

func TestGetMediaFull_NullDataReturnsError(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": {"Media": null}}`))
	})

	_, err := client.GetMediaFull(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for null Media response")
	}
}

func TestGetMediaFull_HTTPError(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.GetMediaFull(context.Background(), 30013)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- test helpers ---

func assertEqual[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %d", name, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %d, want %d", name, *got, want)
	}
}
