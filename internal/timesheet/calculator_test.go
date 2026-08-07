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


// СreateTestCalendarForTimesheet создаёт тестовый календарь для тестов timesheet
func СreateTestCalendarForTimesheet() *calendar.CalendarFile {
    return &calendar.CalendarFile{
        Version:  1,
        Timezone: "+03:00",
        Calendars: []calendar.Calendar{
            {
                Name:    "five-day",
                Users:   []string{"ivanov", "petrova"},
                Enabled: true,
                Week: []calendar.WeekDay{
                    {
                        WeekDay: "mon",
                        DayOff:  false,
                        Begin:   "09:00:00",
                        End:     "18:00:00",
                        Breaks: []calendar.Break{
                            {Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
                        },
                    },
                    {
                        WeekDay: "tue",
                        DayOff:  false,
                        Begin:   "09:00:00",
                        End:     "18:00:00",
                        Breaks: []calendar.Break{
                            {Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
                        },
                    },
                    {
                        WeekDay: "sat",
                        DayOff:  true,
                    },
                    {
                        WeekDay: "sun",
                        DayOff:  true,
                    },
                },
                Holidays: []calendar.Holiday{
                    {Name: "Новый год", Month: 1, Day: 1},
                },
                Exceptions: []calendar.Exception{
                    {Date: "2026-03-07", DayOff: false, Begin: "09:00:00", End: "16:00:00"},
                },
                Vacations: []calendar.Vacation{
                    {From: "2026-03-10", To: "2026-03-23"},
                },
            },
        },
    }
}

// mustParseTime парсит время в формате RFC3339, паникует при ошибке.
func mustParseTime(s string) time.Time {
    t, err := time.Parse(time.RFC3339, s)
    if err != nil {
        panic(err)
    }
    return t
}

func TestDistributeByComputer(t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600)

	// Подготовка интервалов
    // Иванов работает на двух компьютерах с пересекающимися интервалами
    intervals := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
        {User: "ivanov", Comp: "pc2", Start: "2026-03-02T10:00:00+03:00", Stop: "2026-03-02T12:00:00+03:00"},
    }

	// Без обрезки
	rawRes, err := DistributeByComputer(intervals, nil, nil, nil, loc)
	require.NoError(t, err)
	assert.NotNil(t, rawRes)

	pc1Seg := rawRes["pc1"]
	pc2Seg := rawRes["pc2"]

	assert.Len(t, pc1Seg, 1)
	assert.Len(t, pc2Seg, 2)

	// Проверим длительности
	dur1, _ := pc1Seg[0].Duration()
	assert.Equal(t, 1*time.Hour, dur1)

	var totalPc1, totalPc2 time.Duration
	for _, seg := range pc1Seg {
		d, _ := seg.Duration()
		totalPc1 += d
	}
	for _, seg := range pc2Seg {
		d, _ := seg.Duration()
		totalPc2 += d
	}

	assert.Equal(t, 2*time.Hour, totalPc1) 
	assert.Equal(t, 1*time.Hour, totalPc2) 

	// С обрезкой по смене (9:00–18:00)
    shiftBegin := time.Date(2026, 3, 2, 9, 0, 0, 0, loc)
    shiftEnd := time.Date(2026, 3, 2, 18, 0, 0, 0, loc)
    breaks := []calendar.Break{} // без перерывов

	cutRes, err := DistributeByComputer(intervals, &shiftBegin, &shiftEnd, breaks, loc)
	require.NoError(t, err)

	// Интервалы полностью внутри смены, поэтому результат должен совпадать с raw
    var cutTotalPc1, cutTotalPc2 time.Duration
 	for _, seg := range cutRes["pc1"] {
        d, _ := seg.Duration()
        cutTotalPc1 += d
    }
    for _, seg := range cutRes["pc2"] {
        d, _ := seg.Duration()
        cutTotalPc2 += d
    }
    assert.Equal(t, totalPc1, cutTotalPc1)
    assert.Equal(t, totalPc2, cutTotalPc2)

	// С обрезкой и перерывами
	breaksWithLunch := []calendar.Break{
		{Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
	}

	// Добавим интервал, который пересекает перерыв
    intervalsWithBreak := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-03-02T12:30:00+03:00", Stop: "2026-03-02T14:30:00+03:00"},
    }

	cutWithBreak, err := DistributeByComputer(intervalsWithBreak, &shiftBegin, &shiftEnd, breaksWithLunch, loc)
	require.NoError(t, err)

	// Ожидаем, что получится два отрезка: 12:30–13:00 и 14:00–14:30
	segments := cutWithBreak["pc1"]
	assert.Len(t, segments, 2)
	// Проверим длительности 
	d1, _ := segments[0].Duration()
    d2, _ := segments[1].Duration()
    assert.Equal(t, 30*time.Minute, d1)
    assert.Equal(t, 30*time.Minute, d2)
}


func TestCalculateDay (t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600)
	calendarFile := СreateTestCalendarForTimesheet()

	tolerance := 5 * time.Minute


	// 1. Рабочий день, один компьютер, без перерывов (понедельник)
	date := time.Date(2026, 3, 2, 0, 0, 0, 0, loc) // понедельник
    intervals := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T12:00:00+03:00"},
        {User: "ivanov", Comp: "pc1", Start: "2026-03-02T13:00:00+03:00", Stop: "2026-03-02T17:00:00+03:00"},
    }

	rows, err := CalculateDay("ivanov", date, intervals, calendarFile, tolerance)
	require.NoError(t, err)
	assert.Len(t, rows, 1) // один компьютер
	row := rows[0]

	assert.Equal(t, "ivanov", row.User)
	assert.Equal(t, "pc1", row.Comp)
	assert.Equal(t, "2026-03-02", row.Day)
	assert.Equal(t, string(calendar.DayWorkWeek), row.DayType)
	assert.True(t, row.IsWorkDay)

	// Норма: 9:00–18:00 минус обед 1 час = 8 часов
    assert.Equal(t, int64(8*time.Hour/time.Millisecond), row.NormTimeMS)

    // Активность: 3 часа утром + 4 часа днём = 7 часов (без вычета перерывов)
    assert.Equal(t, int64(7*time.Hour/time.Millisecond), row.ActiveTimeMS)

    // Обрезанное: утром 9–12 (3ч), днём 14–17 (3ч) – обед 13–14 вычитается, итого 6ч
    assert.Equal(t, int64(6*time.Hour/time.Millisecond), row.ActiveCutMS)

    // Отклонение: 6 - 8 = -2ч -> underwork
    assert.Equal(t, "underwork", row.DeviationClass)
    assert.Equal(t, int64(-2*time.Hour/time.Millisecond), row.DeviationMS)

	// 2. Рабочий день с перекрытием компьютеров (два компа)
    intervalsMulti := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-03-02T09:00:00+03:00", Stop: "2026-03-02T11:00:00+03:00"},
        {User: "ivanov", Comp: "pc2", Start: "2026-03-02T10:00:00+03:00", Stop: "2026-03-02T12:00:00+03:00"},
    }

	rowsMulti, err := CalculateDay("ivanov", date, intervalsMulti, calendarFile, tolerance)
	require.NoError(t, err)
	assert.Len(t, rowsMulti, 2) //два компьютера

	// Проверим, что сумма active_time по компьютерам равна объединённому времени (3 часа)
    var totalActive time.Duration
    for _, r := range rowsMulti {
        totalActive += time.Duration(r.ActiveTimeMS) * time.Millisecond
    }
    // Ожидаем, что pc1 получит 2 часа (9–11), pc2 получит 1 час (11–12) – итого 3 часа
    assert.Equal(t, 3*time.Hour, totalActive)


	// 3. Нерабочий день (суббота) с активностью
    dateSun := time.Date(2026, 3, 8, 0, 0, 0, 0, loc) // воскресенье
    intervalsSun := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-03-08T10:00:00+03:00", Stop: "2026-03-08T11:00:00+03:00"},
    }

	rowsSun, err := CalculateDay("ivanov", dateSun, intervalsSun, calendarFile, tolerance)
	require.NoError(t, err)
	assert.Len(t, rowsSun, 1)
	rowSun := rowsSun[0]
	assert.Equal(t, string(calendar.DayOffWeek), rowSun.DayType) // воскресенье – выходной
    assert.False(t, rowSun.IsWorkDay)
    assert.Equal(t, int64(0), rowSun.NormTimeMS)
    // Активность есть, но обрезанная = 0 (нерабочий день)
    assert.Equal(t, int64(1*time.Hour/time.Millisecond), rowSun.ActiveTimeMS)
    assert.Equal(t, int64(0), rowSun.ActiveCutMS)
    // Класс отклонения – работа в выходной
    assert.Equal(t, "work_on_day_off", rowSun.DeviationClass)

	// 4. Праздник с активностью
    dateHoliday := time.Date(2026, 1, 1, 0, 0, 0, 0, loc) // 1 января – праздник
    intervalsHoliday := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-01-01T10:00:00+03:00", Stop: "2026-01-01T11:00:00+03:00"},
    }

	rowsHoliday, err := CalculateDay("ivanov", dateHoliday, intervalsHoliday, calendarFile, tolerance)
	require.NoError(t, err)
	rowHoliday := rowsHoliday[0]
	assert.Equal(t, string(calendar.DayHoliday), rowHoliday.DayType)
	assert.False(t, row.IsWorkDay)
	assert.Equal(t, "work_onHoliday", rowHoliday.DeviationClass)

	// 5. Отпуск с активностью
    dateVacation := time.Date(2026, 3, 15, 0, 0, 0, 0, loc) // внутри отпуска
    intervalsVacation := []model.Interval{
        {User: "ivanov", Comp: "pc1", Start: "2026-03-15T10:00:00+03:00", Stop: "2026-03-15T11:00:00+03:00"},
    }
    rowsVacation, err := CalculateDay("ivanov", dateVacation, intervalsVacation, calendarFile, tolerance)
    require.NoError(t, err)
    rowVacation := rowsVacation[0]
    assert.Equal(t, string(calendar.DayVacation), rowVacation.DayType)
    assert.False(t, rowVacation.IsWorkDay)
    assert.Equal(t, "work_on_vacation", rowVacation.DeviationClass)

	// 6. Нет интервалов
	rowsEmpty, err := CalculateDay("ivanov", date, []model.Interval{}, calendarFile, tolerance)
	require.NoError(t, err)
	assert.Len(t, rowsEmpty, 1)
	rowEmpty := rowsEmpty[0]
	assert.Equal(t, "", rowEmpty.Comp)
	assert.Equal(t, int64(0), rowEmpty.ActiveTimeMS)
	assert.Equal(t, int64(0), rowEmpty.ActiveTimeMS)
	assert.Equal(t, "", rowEmpty.DeviationClass)

	// 7. Пользователь без календаря (calendar_disabled)
    rowsDisabled, err := CalculateDay("unknown", date, intervals, calendarFile, tolerance)
    require.NoError(t, err)
    assert.Len(t, rowsDisabled, 1)
    rowDisabled := rowsDisabled[0]
    assert.Equal(t, string(calendar.DayCalendarDisabled), rowDisabled.DayType)
    assert.False(t, rowDisabled.IsWorkDay)
    assert.Equal(t, "no_calendar", rowDisabled.DeviationClass)
    // Для calendar_disabled activeCut = activeTime
    assert.Equal(t, rowDisabled.ActiveTimeMS, rowDisabled.ActiveCutMS)
}