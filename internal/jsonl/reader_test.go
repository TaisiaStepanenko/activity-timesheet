package jsonl

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadActivity(t *testing.T) {
	tmpDir := t.TempDir()
	fileName := filepath.Join(tmpDir, "test.jsonl")

	data := `{"rec_id":1,"user":"ivanov","comp":"pc-014","start":"2026-03-02T09:12:41+03:00","stop":"2026-03-02T09:48:03+03:00"}
	{"rec_id":2,"user":"petrova","comp":"pc-002","start":"2026-03-02T10:00:00+03:00","stop":"2026-03-02T10:30:00+03:00"}
	{"rec_id":3,"user":"","comp":"pc-014","start":"2026-03-02T11:00:00+03:00","stop":"2026-03-02T11:30:00+03:00"}
	{"rec_id":4,"user":"sidorov","comp":"pc-031","start":"2026-03-02T09:00:00+03:00","stop":"2026-03-02T09:00:00+03:00"}
	{"rec_id":5,"user":"invalid","comp":"pc-001","start":"2026-03-02 09:00:00","stop":"2026-03-02T09:30:00+03:00"}
	{"rec_id":6,"user":"stop_before","comp":"pc-002","start":"2026-03-02T10:00:00+03:00","stop":"2026-03-02T09:30:00+03:00"} 
	`

	err := os.WriteFile(fileName, []byte(data), 0644)
	require.NoError(t, err)

	cont := context.Background()
	out := make(chan model.Interval, 10)
	var errLog []string
	errLogFn := func(err error)  {
		errLog = append(errLog, err.Error())
	}

	stats, err := ReadActivity(cont, fileName, out, errLogFn, 10)
	require.NoError(t, err)

	// СОбираем интервалы из канала
	var intervals []model.Interval
	for inter := range out {
		intervals = append(intervals, inter)
	}

	assert.Equal(t, stats.TotalLines, 6)
	assert.Equal(t, stats.ValidCount, 2)
	assert.Equal(t, stats.ZeroCount, 1)
	assert.Equal(t, stats.ErrorCount, 3)
	assert.Len(t, intervals, 2)
	assert.Len(t, errLog, 3)

	assert.Contains(t, errLog[0], "user \"<unknown>\": empty user")
}

func TestReadActivity_MaxErrors(t *testing.T) {
	tmpDir := t.TempDir()
	fileName := filepath.Join(tmpDir, "test_max.jsonl")

	// Создаём 10 ошибочных записей c пустым user
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, `{"rec_id":`+string(rune('0'+i))+`,"user":"","comp":"pc","start":"2026-03-02T09:00:00+03:00","stop":"2026-03-02T10:00:00+03:00"}`)
	}

	data := []byte{}
	for _, line := range lines {
		data = append(data, line...)
		data = append(data, '\n')
	}

	err := os.WriteFile(fileName, data, 0644)
	require.NoError(t, err)

	cont := context.Background()
	out := make(chan model.Interval, 10)
	var errLog []string
	errLogFn := func(err error)  {
		errLog = append(errLog, err.Error())
	}

	maxErrors := 3
	stats, err := ReadActivity(cont, fileName, out, errLogFn, maxErrors)
	require.NoError(t, err)

	// Читаем из канала (все записи ошибочны, канал должен быть пуст и закрыт)
	for range out {
	}

	assert.Equal(t, 10, stats.TotalLines)
	assert.Equal(t, 10, stats.ErrorCount)
	assert.Len(t, errLog, maxErrors)
}

func TestReadActivity_ContextCancel(t *testing.T) {
	tmpDir := t.TempDir()
	fileName := filepath.Join(tmpDir, "test_context.jsonl")

	// Создаём 100 ошибочных записей, чтобы чтение не завершилось мгновенно
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, `{"rec_id":`+string(rune('0'+i))+`,"user":"u","comp":"c","start":"2026-03-02T09:00:00+03:00","stop":"2026-03-02T10:00:00+03:00"}`)
	}

	data := []byte{}
	for _, line := range lines {
		data = append(data, line...)
		data = append(data, '\n')
	}

	err := os.WriteFile(fileName, data, 0644)
	require.NoError(t, err)

	cont, cancel := context.WithCancel(context.Background())
	out := make(chan model.Interval, 1)
	errLogFn := func(err error)  {
	}

	// Запускаем чтение, затем отменяем контекст через 10 мс
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	stats, err := ReadActivity(cont, fileName, out, errLogFn, 10)
	assert.ErrorIs(t, err, context.Canceled)

	for range out {
	}

	assert.GreaterOrEqual(t, stats.TotalLines, 1)

}