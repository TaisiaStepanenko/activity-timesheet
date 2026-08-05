package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCalendar создаёт тестовый календарь для проверки GetDayInfo.
func createTestCalendar() *CalendarFile {

	return &CalendarFile{
		Version:  1,
		Timezone: "+03:00",
		Calendars: []Calendar{
			{
				Name:    "five-day",
				Users:   []string{"ivanov", "petrova"},
				Enabled: true,
				Week: []WeekDay{
					{
						WeekDay: "mon",
						DayOff:  false,
						Begin:   "09:00:00",
						End:     "18:00:00",
						Breaks: []Break{
							{Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
						},
					},
					{
						WeekDay: "tue",
						DayOff:  false,
						Begin:   "09:00:00",
						End:     "18:00:00",
						Breaks: []Break{
							{Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
						},
					},
					{
						WeekDay: "wed",
						DayOff:  false,
						Begin:   "09:00:00",
						End:     "18:00:00",
						Breaks: []Break{
							{Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
						},
					},
					{
						WeekDay: "thu",
						DayOff:  false,
						Begin:   "09:00:00",
						End:     "18:00:00",
						Breaks: []Break{
							{Name: "lunch", Begin: "13:00:00", End: "14:00:00"},
						},
					},
					{
						WeekDay: "fri",
						DayOff:  false,
						Begin:   "09:00:00",
						End:     "17:00:00",
						Breaks:  []Break{}, // без перерывов
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
				Holidays: []Holiday{
					{Name: "Новый год", Month: 1, Day: 1},
					{Name: "8 Марта", Month: 3, Day: 8},
				},
				Exceptions: []Exception{
					{Date: "2026-03-07", DayOff: false, Begin: "09:00:00", End: "16:00:00"}, // рабочая суббота
					{Date: "2026-03-09", DayOff: true},                                       // дополнительный выходной
					{Date: "2026-01-02", DayOff: false, Begin: "10:00:00", End: "15:00:00"}, // рабочий праздник (исключение перекрывает праздник)
				},
				Vacations: []Vacation{
					{From: "2026-03-10", To: "2026-03-23"},
					{From: "2026-07-01", To: "2026-07-15"},
				},
			},
			{
				Name:    "night-shift",
				Users:   []string{"sidorov"},
				Enabled: true,
				Week: []WeekDay{
					{
						WeekDay: "mon",
						DayOff:  false,
						Begin:   "20:00:00",
						End:     "04:00:00",
						Breaks: []Break{
							{Name: "dinner", Begin: "23:00:00", End: "23:30:00"},
						},
					},
				},
				Holidays:  []Holiday{},
				Exceptions: []Exception{},
				Vacations:  []Vacation{},
			},
		},
	}
}

func TestGetDayByInfo_WorkWeek(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 2, 0, 0, 0, 0, loc) // понедельник

	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	assert.Equal(t, DayWorkWeek, info.DayType)
	assert.True(t, info.IsWorkDay)
	assert.Equal(t, 8*time.Hour, info.NormTime)
	assert.NotEmpty(t, info.Begin)
	assert.NotEmpty(t, info.End)
	assert.Len(t, info.Breaks, 1)
}

func TestGetDayByInfo_WorkWeekFriday(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 6, 0, 0, 0, 0, loc) // пятница

	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	assert.Equal(t, DayWorkWeek, info.DayType)
	assert.True(t, info.IsWorkDay)
	assert.Equal(t, 8*time.Hour, info.NormTime)
	assert.Len(t, info.Breaks, 0)
}

func TestGetDayByInfo_DayOffWeek(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 7, 0, 0, 0, 0, loc) // суббота

	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	// Но есть исключение на 2026-03-07 — рабочая суббота!
	// Поэтому проверим, что исключение сработало, а не день недели.
	assert.Equal(t, DayException, info.DayType)
	assert.True(t, info.IsWorkDay)
	assert.Equal(t, 7*time.Hour, info.NormTime)
}

func TestGetDayByInfo_Weekend(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 8, 0, 0, 0, 0, loc) // воскресенье

	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	
	assert.Equal(t, DayHoliday, info.DayType)
	assert.False(t, info.IsWorkDay)
	assert.Equal(t, time.Duration(0), info.NormTime)
}

func TestGetDayByInfo_Holiday(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, loc) // воскресенье

	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	
	assert.Equal(t, DayHoliday, info.DayType)
	assert.False(t, info.IsWorkDay)
	assert.Equal(t, time.Duration(0), info.NormTime)
}

func TestGetDayByInfo_ExceptionOverridesHoliday(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 1, 2, 0, 0, 0, 0, loc) // 2 января — исключение

	// В календаре есть исключение на 2026-01-02: рабочий день с 10:00 до 15:00.
	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	
	assert.Equal(t, DayException, info.DayType)
	assert.True(t, info.IsWorkDay)
	assert.Equal(t, 5*time.Hour, info.NormTime)
}

func TestGetDayByInfo_Vacation(t *testing.T) {
	calendarFile := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, loc) // дата внутри отпуска

	info, err := GetDayInfo(calendarFile, "ivanov", date)
	require.NoError(t, err)

	
	assert.Equal(t, DayVacation, info.DayType)
	assert.False(t, info.IsWorkDay)
	assert.Equal(t, time.Duration(0), info.NormTime)
}

func TestGetDayInfo_VacationBoundary(t *testing.T) {
	cf := createTestCalendar()
	loc := time.FixedZone("", 3*3600)

	// Проверяем границы отпуска
	tests := []struct {
		name     string
		date     time.Time
		expected DayType
	}{
		{
			name:     "первый день отпуска",
			date:     time.Date(2026, 3, 10, 0, 0, 0, 0, loc),
			expected: DayVacation,
		},
		{
			name:     "последний день отпуска",
			date:     time.Date(2026, 3, 23, 0, 0, 0, 0, loc),
			expected: DayVacation,
		},
		{
			name:     "день до отпуска",
			date:     time.Date(2026, 3, 9, 0, 0, 0, 0, loc),
			expected: DayOffException, // 9 марта — исключение (выходной)
		},
		{
			name:     "день после отпуска",
			date:     time.Date(2026, 3, 24, 0, 0, 0, 0, loc),
			expected: DayWorkWeek,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := GetDayInfo(cf, "ivanov", tt.date)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, info.DayType)
		})
	}
}

func TestGetDayInfo_NightShift(t *testing.T) {
	cf := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 2, 0, 0, 0, 0, loc) // понедельник

	info, err := GetDayInfo(cf, "sidorov", date)
	require.NoError(t, err)

	assert.Equal(t, DayWorkWeek, info.DayType)
	assert.True(t, info.IsWorkDay)
	assert.Equal(t, 7*time.Hour+30*time.Minute, info.NormTime) // 20:00–04:00 = 8 часов, минус перерыв 30 мин = 7.5 часов
	assert.Len(t, info.Breaks, 1)
}

func TestGetDayInfo_CalendarDisabled(t *testing.T) {
	cf := createTestCalendar()
	loc := time.FixedZone("", 3*3600)
	date := time.Date(2026, 3, 2, 0, 0, 0, 0, loc)

	// Пользователь, которого нет в календаре
	info, err := GetDayInfo(cf, "unknown", date)
	require.NoError(t, err)

	assert.Equal(t, DayCalendarDisabled, info.DayType)
	assert.False(t, info.IsWorkDay)
	assert.Equal(t, time.Duration(0), info.NormTime)
	assert.Empty(t, info.Begin)
	assert.Empty(t, info.End)
	assert.Nil(t, info.Breaks)
}

func TestGetDayInfo_InvalidTimezone(t *testing.T) {
    cf := &CalendarFile{
        Version: 1,
        Timezone: "invalid",
        Calendars: []Calendar{
            {Name: "cal", Users: []string{"ivanov"}, Enabled: true, Week: []WeekDay{}},
        },
    }
    date := time.Now()
    _, err := GetDayInfo(cf, "ivanov", date)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "invalid timezone")
}

func TestGetDayInfo_EmptyTimezone(t *testing.T) {
    cf := &CalendarFile{
        Version: 1,
        Timezone: "",
        Calendars: []Calendar{
            {Name: "cal", Users: []string{"ivanov"}, Enabled: true, Week: []WeekDay{}},
        },
    }
    date := time.Now()
    _, err := GetDayInfo(cf, "ivanov", date)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "timezone is empty")
}