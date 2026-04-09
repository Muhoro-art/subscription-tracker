package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantErr   bool
	}{
		{"valid july 2025", "07-2025", 2025, 7, false},
		{"january", "01-2025", 2025, 1, false},
		{"december", "12-2025", 2025, 12, false},
		{"no separator", "072025", 0, 0, true},
		{"empty string", "", 0, 0, true},
		{"letters", "ab-cdef", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, month, err := parseDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantYear, year)
				assert.Equal(t, tt.wantMonth, month)
			}
		})
	}
}

func TestDateToInt(t *testing.T) {
	assert.Equal(t, 202501, dateToInt(2025, 1))
	assert.Equal(t, 202512, dateToInt(2025, 12))
	assert.Equal(t, 202407, dateToInt(2024, 7))
}

func TestOverlapMonths(t *testing.T) {
	endDate := func(s string) *string { return &s }

	tests := []struct {
		name     string
		subStart string
		subEnd   *string
		qStart   string
		qEnd     string
		want     int
	}{
		{
			"full overlap - 12 months",
			"01-2025", endDate("12-2025"),
			"01-2025", "12-2025",
			12,
		},
		{
			"partial overlap - subscription starts later",
			"03-2025", endDate("09-2025"),
			"01-2025", "06-2025",
			4, // Mar, Apr, May, Jun
		},
		{
			"partial overlap - subscription ends earlier",
			"01-2025", endDate("06-2025"),
			"03-2025", "09-2025",
			4, // Mar, Apr, May, Jun
		},
		{
			"no overlap - subscription before period",
			"01-2024", endDate("12-2024"),
			"01-2025", "06-2025",
			0,
		},
		{
			"no overlap - subscription after period",
			"07-2025", endDate("12-2025"),
			"01-2025", "06-2025",
			0,
		},
		{
			"ongoing subscription (no end_date)",
			"03-2025", nil,
			"01-2025", "06-2025",
			4, // Mar, Apr, May, Jun
		},
		{
			"ongoing subscription starting before period",
			"01-2025", nil,
			"01-2025", "06-2025",
			6, // Jan-Jun
		},
		{
			"same month subscription",
			"03-2025", endDate("03-2025"),
			"01-2025", "06-2025",
			1,
		},
		{
			"multi-year - 24 months",
			"01-2024", endDate("12-2025"),
			"01-2024", "12-2025",
			24,
		},
		{
			"boundary - subscription starts at period end",
			"06-2025", endDate("12-2025"),
			"01-2025", "06-2025",
			1, // only June
		},
		{
			"boundary - subscription ends at period start",
			"01-2025", endDate("03-2025"),
			"03-2025", "06-2025",
			1, // only March
		},
		{
			"same month period and subscription",
			"05-2025", endDate("05-2025"),
			"05-2025", "05-2025",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := overlapMonths(tt.subStart, tt.subEnd, tt.qStart, tt.qEnd)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsDateBefore(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"before same year", "01-2025", "02-2025", true},
		{"after same year", "12-2025", "01-2025", false},
		{"equal", "06-2025", "06-2025", false},
		{"year before", "12-2024", "01-2025", true},
		{"year after", "01-2026", "12-2025", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isDateBefore(tt.a, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
