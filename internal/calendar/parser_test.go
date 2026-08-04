package calendar

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция для создания временного файла
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "calendar_*.json")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	tmpFile.Close()

	return tmpFile.Name()
}

func TestLoadCalendar_Valid(t *testing.T) {
	// Создаём временный файл с корректным JSON
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "five-day",
				"users": ["ivanov", "petrova"],
				"enabled": true,
				"week": [
					{"weekday": "mon", "day_off": false, "begin": "09:00:00", "end": "18:00:00", "breaks": [{"name": "lunch", "begin": "13:00:00", "end": "14:00:00"}]},
					{"weekday": "sat", "day_off": true},
					{"weekday": "sun", "day_off": true}
				],
				"holidays": [{"name": "Новый год", "month": 1, "day": 1}],
				"exceptions": [{"date": "2026-03-07", "day_off": false, "begin": "09:00:00", "end": "16:00:00"}],
				"vacations": [{"from": "2026-03-10", "to": "2026-03-23"}]
			}
		]
	}`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	cf, err := LoadCalendar(tmpFile)
	require.NoError(t, err)
	assert.NotNil(t, cf)
	assert.Equal(t, 1, cf.Version)
	assert.Equal(t, "+03:00", cf.Timezone)
	assert.Len(t, cf.Calendars, 1)
	cal := cf.Calendars[0]
	assert.Equal(t, "five-day", cal.Name)
	assert.ElementsMatch(t, []string{"ivanov", "petrova"}, cal.Users)
	assert.True(t, cal.Enabled)
	assert.Len(t, cal.Week, 3)
	assert.Len(t, cal.Holidays, 1)
	assert.Len(t, cal.Exceptions, 1)
	assert.Len(t, cal.Vacations, 1)
}

// Некорректная версия календаря
func TestLoadCalendar_InvalidVersion(t *testing.T) {
	content := `{"version": 2, "timezone": "+03:00", "calendars": []}`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported calendar version")
}

// Дублирование пользователя 
func TestLoadCalendar_DuplicateUser(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{"name": "cal1", "users": ["ivanov"], "enabled": true},
			{"name": "cal2", "users": ["ivanov"], "enabled": true}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "appears in multiple calendars:")
}

func TestLoadCalendar_WeekdayMissingBeginEnd(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{"name": "cal", "users": ["u"], "enabled": true, "week": [{"weekday": "mon", "day_off": false}]}
		]
	}`  

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not day off but missing begin/end")

}

func TestLoadCalendar_WeekdayEndBeforeBegin(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{"name": "cal", "users": ["u"], "enabled": true, "week": [{"weekday": "mon", "day_off": false, "begin": "18:00:00", "end": "09:00:00"}]}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end time must be after begin")

}

func TestLoadCalendar_BreakOutsideShift(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"week": [
					{
						"weekday": "mon",
						"day_off": false,
						"begin": "09:00:00",
						"end": "18:00:00",
						"breaks": [{"name": "late", "begin": "18:00:00", "end": "19:00:00"}]
					}
				]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside shift")

}

func TestLoadCalendar_OverlappingBreaks(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"week": [
					{
						"weekday": "mon",
						"day_off": false,
						"begin": "09:00:00",
						"end": "18:00:00",
						"breaks": [
							{"name": "b1", "begin": "12:00:00", "end": "13:00:00"},
							{"name": "b2", "begin": "12:30:00", "end": "13:30:00"}
						]
					}
				]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overlap")
}

func TestLoadCalendar_InvalidExceptionDate(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"exceptions": [{"date": "2026-13-01", "day_off": false, "begin": "09:00:00", "end": "18:00:00"}]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date")

}

func TestLoadCalendar_ExceptionMissingBeginEnd(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"exceptions": [{"date": "2026-03-07", "day_off": false}]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing begin/end")
}

func TestLoadCalendar_ExceptionEndBeforeBegin(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"exceptions": [{"date": "2026-03-07", "day_off": false, "begin": "18:00:00", "end": "09:00:00"}]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end time must be after begin")
}

func TestLoadCalendar_VacationFromAfterTo(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"vacations": [{"from": "2026-03-23", "to": "2026-03-10"}]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be after start")
}

func TestLoadCalendar_InvalidVacationDate(t *testing.T) {
	content := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "cal",
				"users": ["u"],
				"enabled": true,
				"vacations": [{"from": "2026-03-XX", "to": "2026-03-23"}]
			}
		]
	}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	_, err := LoadCalendar(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from date")
}



