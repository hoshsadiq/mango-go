package chapterparse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptrStr(s string) *string   { return &s }
func ptrF64(f float64) *float64 { return &f }

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantNumber *string
		wantSort   *float64
		wantTitle  *string
		wantVolume *string
	}{
		// Pattern 1: Vol+Ch combined
		{
			name:       "vol_ch_combined",
			filename:   "My Series Vol.2 Ch.015",
			wantNumber: ptrStr("015"),
			wantSort:   ptrF64(15.0),
			wantVolume: ptrStr("2"),
		},
		{
			name:       "vol_ch_with_title",
			filename:   "My Series Vol.2 Ch.015 - The Beginning",
			wantNumber: ptrStr("015"),
			wantSort:   ptrF64(15.0),
			wantTitle:  ptrStr("The Beginning"),
			wantVolume: ptrStr("2"),
		},
		{
			name:       "vol_ch_fractional_volume",
			filename:   "Series Vol.01 Ch.003",
			wantNumber: ptrStr("003"),
			wantSort:   ptrF64(3.0),
			wantVolume: ptrStr("1"),
		},

		// Pattern 2: Ch. prefix
		{
			name:       "ch_prefix_dot",
			filename:   "Ch.001",
			wantNumber: ptrStr("001"),
			wantSort:   ptrF64(1.0),
		},
		{
			name:       "chapter_word_with_title",
			filename:   "Chapter 5 - The Beginning",
			wantNumber: ptrStr("5"),
			wantSort:   ptrF64(5.0),
			wantTitle:  ptrStr("The Beginning"),
		},
		{
			name:       "ch_no_dot",
			filename:   "Ch 10",
			wantNumber: ptrStr("10"),
			wantSort:   ptrF64(10.0),
		},

		// Pattern 3: Series-c001
		{
			name:       "series_c_format",
			filename:   "My-Series-c042",
			wantNumber: ptrStr("042"),
			wantSort:   ptrF64(42.0),
		},
		{
			name:       "series_c_with_title",
			filename:   "My-Series-c042 - Extra",
			wantNumber: ptrStr("042"),
			wantSort:   ptrF64(42.0),
			wantTitle:  ptrStr("Extra"),
		},

		// Pattern 4: [Group] prefix
		{
			name:       "group_prefix_ch",
			filename:   "[Scanlation Group] My Series Ch.005",
			wantNumber: ptrStr("005"),
			wantSort:   ptrF64(5.0),
		},
		{
			name:       "group_prefix_vol_ch",
			filename:   "[Group] Series Vol.3 Ch.012 - Finale",
			wantNumber: ptrStr("012"),
			wantSort:   ptrF64(12.0),
			wantTitle:  ptrStr("Finale"),
			wantVolume: ptrStr("3"),
		},

		// Pattern 5: v01 c001
		{
			name:       "v_c_format",
			filename:   "My Series v02 c010",
			wantNumber: ptrStr("010"),
			wantSort:   ptrF64(10.0),
			wantVolume: ptrStr("2"),
		},
		{
			name:       "v_c_uppercase",
			filename:   "Series V01 C005",
			wantNumber: ptrStr("005"),
			wantSort:   ptrF64(5.0),
			wantVolume: ptrStr("1"),
		},

		// Pattern 6: Fractional chapter (handled by patterns 1-5)
		{
			name:       "fractional_ch_prefix",
			filename:   "Ch.12.5",
			wantNumber: ptrStr("12.5"),
			wantSort:   ptrF64(12.5),
		},
		{
			name:       "fractional_vol_ch",
			filename:   "Vol.1 Ch.3.5 - Half Chapter",
			wantNumber: ptrStr("3.5"),
			wantSort:   ptrF64(3.5),
			wantTitle:  ptrStr("Half Chapter"),
			wantVolume: ptrStr("1"),
		},

		// Pattern 7: Number - Title
		{
			name:       "number_dash_title",
			filename:   "042 - The Final Battle",
			wantNumber: ptrStr("042"),
			wantSort:   ptrF64(42.0),
			wantTitle:  ptrStr("The Final Battle"),
		},
		{
			name:       "fractional_number_dash_title",
			filename:   "12.5 - Side Story",
			wantNumber: ptrStr("12.5"),
			wantSort:   ptrF64(12.5),
			wantTitle:  ptrStr("Side Story"),
		},

		// Pattern 8: Bare number
		{
			name:       "bare_number_padded",
			filename:   "001",
			wantNumber: ptrStr("001"),
			wantSort:   ptrF64(1.0),
		},
		{
			name:       "bare_number_unpadded",
			filename:   "42",
			wantNumber: ptrStr("42"),
			wantSort:   ptrF64(42.0),
		},
		{
			name:       "bare_fractional",
			filename:   "12.5",
			wantNumber: ptrStr("12.5"),
			wantSort:   ptrF64(12.5),
		},

		// Edge cases
		{
			name:       "no_number",
			filename:   "Extras",
			wantNumber: nil,
		},
		{
			name:       "unicode_filename",
			filename:   "第001話",
			wantNumber: nil,
		},
		{
			name:       "empty_string",
			filename:   "",
			wantNumber: nil,
		},
		{
			name:       "whitespace_only",
			filename:   "   ",
			wantNumber: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFilename(tt.filename)
			assert.NotNil(t, got, "ParseFilename should never return nil")

			if tt.wantNumber == nil {
				assert.Nil(t, got.Number, "Number should be nil")
			} else {
				if assert.NotNil(t, got.Number, "Number should not be nil") {
					assert.Equal(t, *tt.wantNumber, *got.Number)
				}
			}

			if tt.wantSort == nil {
				assert.Nil(t, got.SortNumber, "SortNumber should be nil")
			} else {
				if assert.NotNil(t, got.SortNumber, "SortNumber should not be nil") {
					assert.InDelta(t, *tt.wantSort, *got.SortNumber, 0.001)
				}
			}

			if tt.wantTitle == nil {
				assert.Nil(t, got.Title, "Title should be nil")
			} else {
				if assert.NotNil(t, got.Title, "Title should not be nil") {
					assert.Equal(t, *tt.wantTitle, *got.Title)
				}
			}

			if tt.wantVolume == nil {
				assert.Nil(t, got.Volume, "Volume should be nil")
			} else {
				if assert.NotNil(t, got.Volume, "Volume should not be nil") {
					assert.Equal(t, *tt.wantVolume, *got.Volume)
				}
			}
		})
	}
}
