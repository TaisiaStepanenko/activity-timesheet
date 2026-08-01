package interval

import (
	"testing"
	"time"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		input    []model.Interval
		expected []model.Interval
	}{
		{
			name:  "empty",
			input: []model.Interval{},
			expected: nil,
		},
		{
			name: "single",
			input: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
			},
			expected: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
			},
		},
		{
			name: "overlap",
			input: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
				{User: "u", Comp: "c", Start: "2026-03-02T09:30:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
			expected: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
		},
		{
			name: "touch (adjacent)",
			input: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
				{User: "u", Comp: "c", Start: "2026-03-02T10:00:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
			expected: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
		},
		{
			name: "gap one second",
			input: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
				{User: "u", Comp: "c", Start: "2026-03-02T10:00:01+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
			expected: []model.Interval{
				{User: "u", Comp: "c", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
				{User: "u", Comp: "c", Start: "2026-03-02T10:00:01+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Merge(test.input)
			assert.Equal(t, len(test.expected), len(got))
			for i := range got {
				assert.Equal(t, test.expected[i].Start, got[i].Start)
				assert.Equal(t, test.expected[i].Stop, got[i].Stop)
			}
		})
	}
}

func TestSplitByDay(t *testing.T) {
	location := time.FixedZone("MSK", 3*3600)
	tests := []struct {
		name     string
		interval model.Interval
		expected int // количество частей
	}{
		{
			name: "within one day",
			interval: model.Interval{
				Start: "2026-03-02T09:00:00+03:00",
				Stop:  "2026-03-02T10:00:00+03:00",
			},
			expected: 1,
		},
		{
			name: "cross midnight",
			interval: model.Interval{
				Start: "2026-03-02T23:00:00+03:00",
				Stop:  "2026-03-03T01:00:00+03:00",
			},
			expected: 2,
		},
		{
			name: "two days",
			interval: model.Interval{
				Start: "2026-03-02T23:00:00+03:00",
				Stop:  "2026-03-04T01:00:00+03:00",
			},
			expected: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parts, err := SplitByDay(test.interval, location)
			require.NoError(t, err)
			assert.Len(t, parts, test.expected)

			// Проверяем, что дни разных частей не равны
			for i := 1; i < len(parts); i++ {
				assert.NotEqual(t, parts[i-1].Day, parts[i].Day)
			}
		})
	}
}