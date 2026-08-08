package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestFiles создаёт временные файлы для теста и возвращает их пути.
func createTestFiles(t *testing.T) (calendarPath, activityPath, outPath string) {
	tmpDir := t.TempDir()

	// Календарь
	calendarContent := `{
		"version": 1,
		"timezone": "+03:00",
		"calendars": [
			{
				"name": "five-day",
				"users": ["ivanov"],
				"enabled": true,
				"week": [
					{"weekday": "mon", "day_off": false, "begin": "09:00:00", "end": "18:00:00", "breaks": [{"name": "lunch", "begin": "13:00:00", "end": "14:00:00"}]},
					{"weekday": "tue", "day_off": false, "begin": "09:00:00", "end": "18:00:00", "breaks": [{"name": "lunch", "begin": "13:00:00", "end": "14:00:00"}]},
					{"weekday": "wed", "day_off": false, "begin": "09:00:00", "end": "18:00:00", "breaks": [{"name": "lunch", "begin": "13:00:00", "end": "14:00:00"}]},
					{"weekday": "thu", "day_off": false, "begin": "09:00:00", "end": "18:00:00", "breaks": [{"name": "lunch", "begin": "13:00:00", "end": "14:00:00"}]},
					{"weekday": "fri", "day_off": false, "begin": "09:00:00", "end": "17:00:00"},
					{"weekday": "sat", "day_off": true},
					{"weekday": "sun", "day_off": true}
				],
				"holidays": [{"name": "Новый год", "month": 1, "day": 1}],
				"exceptions": [],
				"vacations": []
			}
		]
	}`
	calendarPath = filepath.Join(tmpDir, "calendar.json")
	err := os.WriteFile(calendarPath, []byte(calendarContent), 0644)
	require.NoError(t, err)

	// Активность
	activityContent := `{"rec_id":1,"user":"ivanov","comp":"pc-014","start":"2026-03-02T09:12:41+03:00","stop":"2026-03-02T09:48:03+03:00"}
{"rec_id":2,"user":"ivanov","comp":"pc-014","start":"2026-03-02T10:00:00+03:00","stop":"2026-03-02T10:30:00+03:00"}
{"rec_id":3,"user":"petrova","comp":"pc-002","start":"2026-03-02T11:00:00+03:00","stop":"2026-03-02T11:30:00+03:00"}`
	activityPath = filepath.Join(tmpDir, "activity.jsonl")
	err = os.WriteFile(activityPath, []byte(activityContent), 0644)
	require.NoError(t, err)

	outPath = filepath.Join(tmpDir, "output.jsonl")
	return
}

func TestRunBuild(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		calendarPath, activityPath, outPath := createTestFiles(t)

		args := []string{
			"--activity", activityPath,
			"--calendar", calendarPath,
			"--from", "2026-03-01",
			"--to", "2026-03-31",
			"--out", outPath,
		}
		exitCode := RunBuild(args)
		assert.Equal(t, 0, exitCode)

		// Проверяем, что файл создан
		_, err := os.Stat(outPath)
		require.NoError(t, err)

		// Читаем строки
		file, err := os.Open(outPath)
		require.NoError(t, err)
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var rows []model.DailyRow
		for scanner.Scan() {
			var row model.DailyRow
			err := json.Unmarshal(scanner.Bytes(), &row)
			require.NoError(t, err)
			rows = append(rows, row)
		}
		require.NoError(t, scanner.Err())

		// Ожидаем 2 строки: ivanov и petrova
		assert.Len(t, rows, 2)

		// Проверяем ivanov
		var ivanovRow *model.DailyRow
		for _, r := range rows {
			if r.User == "ivanov" {
				ivanovRow = &r
				break
			}
		}
		require.NotNil(t, ivanovRow)
		assert.Equal(t, "pc-014", ivanovRow.Comp)
		assert.Equal(t, "2026-03-02", ivanovRow.Day)
		assert.Equal(t, "work_week", ivanovRow.DayType)
		assert.True(t, ivanovRow.IsWorkDay)
		assert.Equal(t, int64(28800000), ivanovRow.NormTimeMS) // 8 часов
		assert.Equal(t, int64(3922000), ivanovRow.ActiveTimeMS) // 1:05:22
		assert.Equal(t, "underwork", ivanovRow.DeviationClass)

		// Проверяем petrova (без календаря)
		var petrovaRow *model.DailyRow
		for _, r := range rows {
			if r.User == "petrova" {
				petrovaRow = &r
				break
			}
		}
		require.NotNil(t, petrovaRow)
		assert.Equal(t, "pc-002", petrovaRow.Comp)
		assert.Equal(t, "2026-03-02", petrovaRow.Day)
		assert.Equal(t, "calendar_disabled", petrovaRow.DayType)
		assert.False(t, petrovaRow.IsWorkDay)
		assert.Equal(t, int64(0), petrovaRow.NormTimeMS)
		assert.Equal(t, int64(1800000), petrovaRow.ActiveTimeMS) // 30 минут
		assert.Equal(t, "no_calendar", petrovaRow.DeviationClass)
	})

	t.Run("missing required flags", func(t *testing.T) {
		args := []string{}
		exitCode := RunBuild(args)
		assert.Equal(t, 2, exitCode)
	})

	t.Run("invalid from date", func(t *testing.T) {
			calendarPath, activityPath, outPath := createTestFiles(t)

		args := []string{
			"--activity", activityPath,
			"--calendar", calendarPath,
			"--from", "2026-13-01",
			"--to", "2026-03-31",
			"--out", outPath,
		}
		exitCode := RunBuild(args)
		assert.Equal(t, 2, exitCode)

	})

	t.Run("no valid intervals", func(t *testing.T) {
		calendarPath, _, outPath := createTestFiles(t) // activity не создаём – пустой файл
		// Создаём пустой activity
		tmpDir := filepath.Dir(calendarPath)
		activityPath := filepath.Join(tmpDir, "empty.jsonl")
		err := os.WriteFile(activityPath, []byte(""), 0644)
		require.NoError(t, err)

		args := []string{
			"--activity", activityPath,
			"--calendar", calendarPath,
			"--from", "2026-03-01",
			"--to", "2026-03-31",
			"--out", outPath,
		}
		exitCode := RunBuild(args)
		assert.Equal(t, 1, exitCode)
	})

	t.Run("invalid calendar", func(t *testing.T) {
		_, activityPath, outPath := createTestFiles(t)
		tmpDir := filepath.Dir(activityPath)
		calendarPath := filepath.Join(tmpDir, "invalid.jsonl")
		err := os.WriteFile(calendarPath, []byte(""), 0644)
		require.NoError(t, err)

		args := []string{
			"--activity", activityPath,
			"--calendar", calendarPath,
			"--from", "2026-03-01",
			"--to", "2026-03-31",
			"--out", outPath,
		}
		exitCode := RunBuild(args)
		assert.Equal(t, 2, exitCode)
	})
}