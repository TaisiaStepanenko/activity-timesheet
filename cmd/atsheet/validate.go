package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/calendar"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/jsonl"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
)

func RunValidate(args []string) int {

	validateCommand := flag.NewFlagSet("validate", flag.ExitOnError)
	calendarPath := validateCommand.String("calendar", "", "path to calendar JSON file")
	activity := validateCommand.String("activity", "", "path to activity JSONL file")
	maxErrors := validateCommand.Int("max-errors", 20, "maximum number of errors to print")

	err := validateCommand.Parse(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse flags: %v\n", err)
		return 2
	}

	if *calendarPath == "" || *activity == "" {
		fmt.Fprintf(os.Stderr, "both --calendar and --activity are required")
		validateCommand.Usage()
		return 2
	}

	// Валидация календаря
	calendarFile, err := calendar.LoadCalendar(*calendarPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "calendar validation failed: %v\n", err)
		return 2
	}
	fmt.Println("calendar: valid")
	
	// Валидация потока активности
	ctx := context.Background()
	out := make(chan model.Interval, 100) // буферезированный канал
	var errorCount int 
	var userSet = make(map[string]bool) // собираем пользователей из валидных записей
	var wg sync.WaitGroup

	errLog := func(err error)  {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		errorCount++
	}

    wg.Add(1)
	
	go func() {
		defer wg.Done()
		for interval := range out {
			userSet[interval.User] = true
		}
	}()

	stats, err := jsonl.ReadActivity(ctx, *activity, out, errLog, *maxErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read activity: %v\n", err)
		return 2
	}

	wg.Wait()
	// Подсчёт пользователей без календаря
	withoutCalendarUsers := 0
	for user := range userSet {
		if calendarFile.GetUserCalendar(user) == nil {
			withoutCalendarUsers++
		}
 	}

	// Вывод статистики
	fmt.Printf("activity: total lines %d, valid %d, zero %d, errors %d\n",
        stats.TotalLines, stats.ValidCount, stats.ZeroCount, stats.ErrorCount)
    fmt.Printf("users without calendar: %d\n", withoutCalendarUsers)

	// есть ошибки или предупреждения
	if stats.ErrorCount > 0 || withoutCalendarUsers > 0 {
		return 1
	} 

	return 0
}