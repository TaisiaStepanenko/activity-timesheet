package model

import (
	"testing"
	"time"
)

func TestInterval_StartTimeParse(t *testing.T) {
	i := Interval{Start: "2026-03-02T09:12:41+03:00"}
	parseTime, err := i.StartTimeParse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2026, 3, 2, 9, 12, 41, 0, time.FixedZone("", 3*3600))
	if !parseTime.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, parseTime)
	}
}

func TestInterval_StopTimeParse(t *testing.T) {
	i := Interval{Stop: "2026-03-02T09:12:41+03:00"}
	parseTime, err := i.StopTimeParse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2026, 3, 2, 9, 12, 41, 0, time.FixedZone("", 3*3600))
	if !parseTime.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, parseTime)
	}
}

func TestInterval_Duration(t *testing.T) {
	i := Interval{
		Start: "2026-03-02T09:12:41+03:00",
		Stop: "2026-03-02T09:48:03+03:00",
	}

	dur, err := i.Duration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := 35*time.Minute + 22*time.Second
	if dur !=expected {
		t.Errorf("expected %v, got %v", expected, dur)
	}
}

func TestInterval_IsZero(t *testing.T) {
	i := Interval{
		Start: "2026-03-02T09:12:41+03:00",
		Stop: "2026-03-02T09:12:41+03:00",
	}

	zeroDur, err := i.IsZero()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !zeroDur {
		t.Errorf("expected zero interval")
	}
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
			if (err == nil) != test.expected {
				t.Errorf("IsValid() error = %v, want %v", err, test.expected)
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
	if dayInterval.Day != day {
		t.Errorf("expected day %v, got %v", day, dayInterval.Day)
	}
	if dayInterval.Interval != interval {
		t.Error("interval mismatch")
	}
}