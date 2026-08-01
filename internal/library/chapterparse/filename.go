package chapterparse

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedFilename holds the extracted components from a chapter filename.
type ParsedFilename struct {
	Number     *string  // Display number string (e.g., "001", "12.5")
	SortNumber *float64 // Numeric sort value (e.g., 1.0, 12.5)
	Title      *string  // Chapter title if present
	Volume     *string  // Volume number if present
}

var (
	// Pattern 1: Vol+Ch combined, Vol.01 Ch.001 or v01 c001
	volChRe = regexp.MustCompile(`(?i)(?:vol\.?\s*(\d+(?:\.\d+)?)\s+)?ch\.?\s*(\d+(?:\.\d+)?)\s*(?:-\s*(.+))?`)

	// Pattern 2: Ch. prefix, Ch.001 or Chapter 5
	chPrefixRe = regexp.MustCompile(`(?i)ch(?:apter)?\.?\s*(\d+(?:\.\d+)?)\s*(?:-\s*(.+))?`)

	// Pattern 3: Series-c001 format
	seriesCRe = regexp.MustCompile(`(?i)-c(\d+(?:\.\d+)?)\s*(?:-\s*(.+))?$`)

	// Pattern 5: Series v01 c001
	vCRe = regexp.MustCompile(`(?i)v(\d+(?:\.\d+)?)\s+c(\d+(?:\.\d+)?)`)

	// Pattern 7: Number - Title, 001 - Title
	numTitleRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s+-\s+(.+)$`)

	// Pattern 8: Bare number, 001 or 42
	bareNumRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)$`)

	// Pattern 4 helper: strip leading [Group] tag
	groupTagRe = regexp.MustCompile(`^\[.*?]\s*`)
)

// ParseFilename extracts chapter metadata from a filename (without extension).
// Returns a ParsedFilename with nil fields for components not found.
func ParseFilename(filename string) *ParsedFilename {
	result := &ParsedFilename{}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return result
	}

	// Pattern 1: Vol+Ch combined
	if m := volChRe.FindStringSubmatch(filename); m != nil {
		// Only match if we actually have a vol or ch prefix keyword in the match
		// The regex requires "ch" so this is a valid pattern 1/2 match
		if m[1] != "" {
			// Has volume, this is pattern 1
			result.Volume = strPtr(cleanVolume(m[1]))
			result.Number = strPtr(m[2])
			result.SortNumber = parseFloat(m[2])
			if t := strings.TrimSpace(m[3]); t != "" {
				result.Title = strPtr(t)
			}
			return result
		}
	}

	// Pattern 2: Ch. prefix
	if m := chPrefixRe.FindStringSubmatch(filename); m != nil {
		result.Number = strPtr(m[1])
		result.SortNumber = parseFloat(m[1])
		if t := strings.TrimSpace(m[2]); t != "" {
			result.Title = strPtr(t)
		}
		return result
	}

	// Pattern 3: Series-c001
	if m := seriesCRe.FindStringSubmatch(filename); m != nil {
		result.Number = strPtr(m[1])
		result.SortNumber = parseFloat(m[1])
		if t := strings.TrimSpace(m[2]); t != "" {
			result.Title = strPtr(t)
		}
		return result
	}

	// Pattern 4: [Group] prefix, strip tag and retry patterns 1 & 2
	if groupTagRe.MatchString(filename) {
		stripped := groupTagRe.ReplaceAllString(filename, "")
		stripped = strings.TrimSpace(stripped)
		if stripped != "" {
			// Try vol+ch on stripped
			if m := volChRe.FindStringSubmatch(stripped); m != nil {
				if m[1] != "" {
					result.Volume = strPtr(cleanVolume(m[1]))
				}
				result.Number = strPtr(m[2])
				result.SortNumber = parseFloat(m[2])
				if t := strings.TrimSpace(m[3]); t != "" {
					result.Title = strPtr(t)
				}
				return result
			}
			// Try ch prefix on stripped
			if m := chPrefixRe.FindStringSubmatch(stripped); m != nil {
				result.Number = strPtr(m[1])
				result.SortNumber = parseFloat(m[1])
				if t := strings.TrimSpace(m[2]); t != "" {
					result.Title = strPtr(t)
				}
				return result
			}
		}
	}

	// Pattern 5: v01 c001
	if m := vCRe.FindStringSubmatch(filename); m != nil {
		result.Volume = strPtr(cleanVolume(m[1]))
		result.Number = strPtr(m[2])
		result.SortNumber = parseFloat(m[2])
		return result
	}

	// Pattern 7: Number - Title
	if m := numTitleRe.FindStringSubmatch(filename); m != nil {
		result.Number = strPtr(m[1])
		result.SortNumber = parseFloat(m[1])
		if t := strings.TrimSpace(m[2]); t != "" {
			result.Title = strPtr(t)
		}
		return result
	}

	// Pattern 8: Bare number
	if m := bareNumRe.FindStringSubmatch(filename); m != nil {
		result.Number = strPtr(m[1])
		result.SortNumber = parseFloat(m[1])
		return result
	}

	return result
}

func strPtr(s string) *string {
	return &s
}

func parseFloat(s string) *float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}

// cleanVolume normalizes a volume string by stripping unnecessary leading zeros
// while keeping it as a clean display string.
func cleanVolume(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	// Format without trailing zeros for display
	return strconv.FormatFloat(f, 'f', -1, 64)
}
