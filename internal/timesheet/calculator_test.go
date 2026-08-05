package timesheet

import (
	"testing"
	"time"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/calendar"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCutByShift(t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600)

	// Смена: 09:00–18:00
	shiftBegin := time.Date(2026, 3, 2, 9, 0, 0, 0, loc)
	shiftEnd := time.Date(2026, 3, 2, 18, 0, 0, 0, loc)

	// Перерывы: обед 13:00–14:00
	lunchBreaks := []calendar.Break{
		{Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
	}

	tests := []struct {
		name          string
		intervals     []model.Interval
		shiftBegin    time.Time
		shiftEnd      time.Time
		breaks        []calendar.Break
		loc           *time.Location
		expectedCut   time.Duration
		expectedBegin time.Time
		expectedEnd   time.Time
		expectError   bool
	}{
		{
			name: "активность полностью внутри смены без перерывов",
			intervals: []model.Interval{
				{Start: "2026-03-02T10:00:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        []calendar.Break{},
			loc:           loc,
			expectedCut:   time.Hour,
			expectedBegin: time.Date(2026, 3, 2, 10, 0, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 11, 0, 0, 0, loc),
			expectError:   false,
		},
		{
			name: "активность внутри смены, но не пересекает перерыв",
			intervals: []model.Interval{
				{Start: "2026-03-02T10:00:00+03:00", Stop: "2026-03-02T12:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        lunchBreaks,
			loc:           loc,
			expectedCut:   2 * time.Hour,
			expectedBegin: time.Date(2026, 3, 2, 10, 0, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 12, 0, 0, 0, loc),
			expectError:   false,
		},
		{
			name: "активность пересекает перерыв",
			intervals: []model.Interval{
				{Start: "2026-03-02T12:30:00+03:00", Stop: "2026-03-02T14:30:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        lunchBreaks,
			loc:           loc,
			expectedCut:   1 * time.Hour, // 30 мин до + 30 мин после = 1 час
			expectedBegin: time.Date(2026, 3, 2, 12, 30, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 14, 30, 0, 0, loc),
			expectError:   false,
		},
		{
			name: "активность полностью в перерыве",
			intervals: []model.Interval{
				{Start: "2026-03-02T13:15:00+03:00", Stop: "2026-03-02T13:45:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        lunchBreaks,
			loc:           loc,
			expectedCut:   0,
			expectedBegin: time.Time{},
			expectedEnd:   time.Time{},
			expectError:   false,
		},
		{
			name: "активность частично вне смены (до начала)",
			intervals: []model.Interval{
				{Start: "2026-03-02T08:30:00+03:00", Stop: "2026-03-02T10:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        []calendar.Break{},
			loc:           loc,
			expectedCut:   time.Hour,
			expectedBegin: time.Date(2026, 3, 2, 9, 0, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 10, 0, 0, 0, loc),
			expectError:   false,
		},
		{
			name: "активность частично вне смены (после окончания)",
			intervals: []model.Interval{
				{Start: "2026-03-02T17:00:00+03:00", Stop: "2026-03-02T19:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        []calendar.Break{},
			loc:           loc,
			expectedCut:   time.Hour,
			expectedBegin: time.Date(2026, 3, 2, 17, 0, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 18, 0, 0, 0, loc),
			expectError:   false,
		},
		{
			name: "активность полностью вне смены",
			intervals: []model.Interval{
				{Start: "2026-03-02T06:00:00+03:00", Stop: "2026-03-02T07:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        []calendar.Break{},
			loc:           loc,
			expectedCut:   0,
			expectedBegin: time.Time{},
			expectedEnd:   time.Time{},
			expectError:   false,
		},
		{
			name: "несколько интервалов активности с перерывами",
			intervals: []model.Interval{
				{Start: "2026-03-02T10:00:00+03:00", Stop: "2026-03-02T12:00:00+03:00"},
				{Start: "2026-03-02T14:00:00+03:00", Stop: "2026-03-02T16:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        lunchBreaks,
			loc:           loc,
			expectedCut:   4 * time.Hour,
			expectedBegin: time.Date(2026, 3, 2, 10, 0, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 16, 0, 0, 0, loc),
			expectError:   false,
		},
		{
			name: "активность покрывает всю смену, перерыв вычитается",
			intervals: []model.Interval{
				{Start: "2026-03-02T08:00:00+03:00", Stop: "2026-03-02T19:00:00+03:00"},
			},
			shiftBegin:    shiftBegin,
			shiftEnd:      shiftEnd,
			breaks:        lunchBreaks,
			loc:           loc,
			expectedCut:   8 * time.Hour,
			expectedBegin: time.Date(2026, 3, 2, 9, 0, 0, 0, loc),
			expectedEnd:   time.Date(2026, 3, 2, 18, 0, 0, 0, loc),
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cut, begin, end, err := CutByShift(tt.intervals, tt.shiftBegin, tt.shiftEnd, tt.breaks, tt.loc)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCut, cut)

			// Сравниваем времена в UTC, чтобы игнорировать имя часового пояса
			if tt.expectedBegin.IsZero() {
				assert.True(t, begin.IsZero())
			} else {
				assert.Equal(t, tt.expectedBegin.UTC(), begin.UTC())
			}

			if tt.expectedEnd.IsZero() {
				assert.True(t, end.IsZero())
			} else {
				assert.Equal(t, tt.expectedEnd.UTC(), end.UTC())
			}
		})
	}
}