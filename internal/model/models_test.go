package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterval_StartTimeParse(t *testing.T) {
	i := Interval{Start: "2026-03-02T09:12:41+03:00"}
	parseTime, err := i.StartTimeParse()
	require.NoError(t, err)

	expected := time.Date(2026, 3, 2, 9, 12, 41, 0, time.FixedZone("", 3*3600))
	assert.Equal(t, expected.UTC(), parseTime.UTC())
}

func TestInterval_StopTimeParse(t *testing.T) {
	i := Interval{Stop: "2026-03-02T09:12:41+03:00"}
	parseTime, err := i.StopTimeParse()
	require.NoError(t, err)

	expected := time.Date(2026, 3, 2, 9, 12, 41, 0, time.FixedZone("", 3*3600))
	assert.Equal(t, expected.UTC(), parseTime.UTC())
}

func TestInterval_Duration(t *testing.T) {
	i := Interval{
		Start: "2026-03-02T09:12:41+03:00",
		Stop: "2026-03-02T09:48:03+03:00",
	}

	dur, err := i.Duration()
	require.NoError(t, err)

	expected := 35*time.Minute + 22*time.Second
	assert.Equal(t, expected, dur)
}

func TestInterval_IsZero(t *testing.T) {
	i := Interval{
		Start: "2026-03-02T09:12:41+03:00",
		Stop: "2026-03-02T09:12:41+03:00",
	}

	zeroDur, err := i.IsZero()
	require.NoError(t, err)

	assert.True(t, zeroDur)
}

func TestInterval_IsValid(t *testing.T) {
	testsInterval := []struct {
		name string
		interval Interval
		expected bool
	} {
		{
			name: "valid",
			interval: Interval{
				User: "Ivanov",
				Comp: "pc-014",
				Start: "2026-03-02T09:12:41+03:00",
				Stop:  "2026-03-02T09:48:03+03:00",
			},
			expected: true,
		},
		{
			name: "empty user",
			interval: Interval{
				User: "",
				Comp: "pc-014",
				Start: "2026-03-02T09:12:41+03:00",
				Stop:  "2026-03-02T09:48:03+03:00",
			},
			expected: false,
		},
		{
			name: "empty comp",
			interval: Interval{
				User:  "ivanov",
				Comp:  "",
				Start: "2026-03-02T09:12:41+03:00",
				Stop:  "2026-03-02T09:48:03+03:00",
			},
			expected: false,
		},
		{
			name: "stop before start",
			interval: Interval{
				User:  "ivanov",
				Comp:  "pc-014",
				Start: "2026-03-02T10:00:00+03:00",
				Stop:  "2026-03-02T09:48:03+03:00",
			},
			expected: false,
		},
		{
			name: "invalid RFC3339",
			interval: Interval{
				User:  "ivanov",
				Comp:  "pc-014",
				Start: "2026-03-02 09:12:41",
				Stop:  "2026-03-02T09:48:03+03:00",
			},
			expected: false,
		},
	}

	for _, test := range testsInterval {
		t.Run(test.name, func(t *testing.T) {
			err := test.interval.IsValid()
			if test.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewDayInterval(t *testing.T) {
	interval := Interval{
		RecID: 1,
		User: "ivanov",
		Comp: "pc-014",
		Start: "2026-03-02T23:00:00+03:00",
		Stop:  "2026-03-03T01:00:00+03:00",
	}

	day := time.Date(2026, 3, 2, 0, 0, 0, 0, time.FixedZone("", 3*3600))
	dayInterval := NewDayInterval(interval, day)
	assert.Equal(t, day, dayInterval.Day)
	assert.Equal(t, interval, dayInterval.Interval)
}