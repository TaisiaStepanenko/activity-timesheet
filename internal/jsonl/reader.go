package jsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
)

type ReadStats struct {
	TotalLines int
	ValidCount int
	ZeroCount  int
	ErrorCount int
	LateCount  int
	NoCalendar int
}

// ReadActivity читает JSONL из reader и отправляет корректные интервалы в out
func ReadActivity(cont context.Context, filename string, out chan<- model.Interval, errLog func(err error), maxErrors int) (ReadStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return ReadStats{}, fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()
	defer close(out)

	scanner := bufio.NewScanner(file)
	stats := ReadStats{}
	errorCount := 0
	lineNumber := 0

	for scanner.Scan() {
		if cont.Err() != nil {
			return stats, cont.Err()
		}
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		stats.TotalLines++

		var interval model.Interval
		err := json.Unmarshal([]byte(line), &interval)
		if err != nil {
			stats.ErrorCount++
			if maxErrors == 0 || errorCount < maxErrors {
				errLog(fmt.Errorf("%s:%d: JSON unmarshal error: %w", filename, lineNumber, err))
				errorCount++
			}
			continue
		}

		// Нулевой интервал (stop == start)
		if zero, _ := interval.IsZero(); zero {
			stats.ZeroCount++
			continue
		}

		// Валидация 
		err = interval.IsValid()
		if err != nil {
			stats.ErrorCount++
			if (maxErrors == 0 || errorCount < maxErrors) {
				user := interval.User
				if (user == "") {
					user = "<unknown>"
				}
				errLog(fmt.Errorf("%s:%d: user %q: %w", filename, lineNumber, user, err))
				errorCount++
			}
			continue
		}

		
		// Отправляем валидный интервал
		stats.ValidCount++
		select {
		case out <- interval:
		case <-cont.Done():
			return stats, cont.Err()
		}
	}
  
	err = scanner.Err()
	if err != nil {
		return stats, fmt.Errorf("scanner error reading %s: %w", filename, err)
	}
	return stats, nil
}