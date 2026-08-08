package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/calendar"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/interval"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/timesheet"
)

func RunBuild(args []string) int {

	buildCommand := flag.NewFlagSet("build", flag.ExitOnError)

	calendarPath := buildCommand.String("calendar", "", "path to calendar JSON file")
	activity := buildCommand.String("activity", "", "path to activity JSONL file")
	fromStr := buildCommand.String("from", "", "period start date (YYYY-MM-DD)")
	toStr := buildCommand.String("to", "", "period end date (YYYY-MM-DD)")
	//workers := buildCommand.Int("workers", 1, "number of workers (not used yet)")
	out := buildCommand.String("out", "", "output JSONL file")
	//lateToleranceStr := buildCommand.String("late-tolerance", "1h", "tolerance for late records (not used yet)")
	toleranceStr := buildCommand.String("tolerance", "5m", "tolerance for deviation classification")
	maxErrors := buildCommand.Int("max-errors", 20, "maximum number of errors to print")


	err := buildCommand.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse flags: %v\n", err)
		return 2
	}

	// Проверяем обязательные флаги 
	if *calendarPath == "" || *activity == "" || *fromStr == "" || *toStr == "" || *out == ""  {
		fmt.Fprintln(os.Stderr, "all flags --activity, --calendar, --from, --to, --out are required")		
		buildCommand.Usage()
		return 2
	}

	// Парсим даты
	from, err := time.Parse("2006-01-02", *fromStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid from date: %v\n", err)
		return 2
	}
	to, err := time.Parse("2006-01-02", *toStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid to date: %v\n", err)
		return 2
	}
	if to.Before(from) {
		fmt.Fprintln(os.Stderr, "to date must be after from date")
		return 2
	}

	// Парсим tolerance
	tolerance, err := time.ParseDuration(*toleranceStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid tolerance: %v\n", err)
		return 2
	}

	// Загружаем календарь
	cf, err := calendar.LoadCalendar(*calendarPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load calendar: %v\n", err)
		return 2
	}
	fmt.Println("calendar loaded successfully")

	// Читаем все интервалы из activity
	intervals, err := ReadAllActivity(*activity, *maxErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read activity: %v\n", err)
		return 2
	} 
	if len(intervals) == 0{
		fmt.Fprintln(os.Stderr, "no valid intervals found")
		return 1
	}
	fmt.Printf("read %d intervals\n", len(intervals))

	// Получаем локацию из календаря
	loc, err := calendar.ParseTimezone(cf.Timezone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid timezone: %v\n", err)
		return 2
	}

	// Группируем интервалы по пользователю и дню
	grouped, err := interval.GroupByUserDay(intervals, loc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "grouping error: %v\n", err)
		return 2
	}
	fmt.Printf("grouped into %d user-days\n", len(grouped))

	// Создаём временный файл для  записи
	tempFile, err := os.CreateTemp(filepath.Dir(*out), "atsheet_*.jsonl")	
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temporary file: %v\n", err)
		return 2
	}

	tempName := tempFile.Name()
	defer os.Remove(tempName)

	writer := bufio.NewWriter(tempFile)
	rowCount := 0

	// Для каждой группы вычисляем табель активности
	for key, intervals := range grouped {
		// Проверяем, чтобы каждый интервал входил в период [from, to]
		if key.Day.Before(from) || key.Day.After(to) {
			continue
		} 
		rows, err := timesheet.CalculateDay(key.User, key.Day, intervals, cf, tolerance)
		if err != nil {
			fmt.Fprintf(os.Stderr, "calculation error for %s on %s: %v\n", key.User, key.Day.Format("2006-01-02"), err)
			return 2
		}
		for _, row := range rows {
			data, err := json.Marshal(row)
			if err != nil {
				fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
				return 2
			}
			if _, err := writer.Write(data); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
				return 2
			}
			if _, err := writer.Write([]byte("\n")); err != nil {
				fmt.Fprintf(os.Stderr, "write newline error: %v\n", err)
			}
			rowCount++
		}
	}

	// Сбрасываем буфер и закрываем временный файл
	err = writer.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
	}

	err = tempFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "close error: %v\n", err)
		return 2
	}

	// Переименование
	err = os.Rename(tempName, *out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rename error: %v\n", err)
		return 2
	}

	fmt.Printf("build completed: %d rows written to %s\n", rowCount, *out)
	return 0
}

// ReadAllActivity читает все интервалы из JSONL-файла в срез (временное решение)
func ReadAllActivity(filename string, maxErrors int) ([]model.Interval, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var intervals []model.Interval
	lineNum := 0
	errorCount := 0

	for scanner.Scan() {
		lineNum++
		str := scanner.Text()
		if str == "" {
			continue
		}
		var interval model.Interval
		err := json.Unmarshal([]byte(str), &interval)
		if err != nil {
			errorCount++
			if errorCount <= maxErrors {
				fmt.Fprintf(os.Stderr, "%s:%d: JSON unmarshal error: %v\n", filename, lineNum, err)
			}
			continue
		}
		err = interval.IsValid()
		if err != nil {
			errorCount++
			if errorCount <= maxErrors {
				fmt.Fprintf(os.Stderr, "%s:%d: validation error: %v\n", filename, lineNum, err)
			}
			continue
		}

		zero, _ := interval.IsZero()
		if zero {
			continue
		}
		intervals = append(intervals, interval)
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	if errorCount > 0 {
		fmt.Fprintf(os.Stderr, "total %d errors while reading activity\n", errorCount)
	}
	return intervals, nil
}