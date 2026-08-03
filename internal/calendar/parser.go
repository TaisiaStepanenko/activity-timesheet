package calendar

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// LoadCalendar читает и парсит файл календаря, выполняя валидацию
func LoadCalendar(filename string) (*CalendarFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read calendar file %s: %w", filename, err)
	}

	var calendarF CalendarFile 
	err = json.Unmarshal(data, &calendarF)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", filename, err)
	}

	if calendarF.Version != 1 {
		return nil, fmt.Errorf("unsupported calendar version %d, expected 1", calendarF.Version)
	}

	// Валидация календарей
	userToCalendar := make(map[string]string)

	for _, calendar := range calendarF.Calendars {
		// Проверяем пользователей на дублирование
		for _, user := range calendar.Users {
			existing, ok := userToCalendar[user]
			if ok {
				return nil, fmt.Errorf("user %q appears in multiple calendars: %s and %s", user, existing, calendar.Name)
			}
			userToCalendar[user] = calendar.Name
		}

		if !calendar.Enabled {
			continue
		}

		// Валидация week
		for _, weekDay := range calendar.Week {
			if (weekDay.DayOff == true) {
				continue
			}

			if weekDay.Begin == "" || weekDay.End == "" {
				return nil, fmt.Errorf("calendar %s: weekday %s is not day off but missing begin/end", calendar.Name, weekDay.WeekDay)
			}

			// Проверяем, что begin < end
			beginTime, err := parseTimeOnly(weekDay.Begin)
			if err != nil{
				return nil, fmt.Errorf("calendar %s: invalid begin time %q for %s: %w", calendar.Name, weekDay.Begin, weekDay.WeekDay, err)
			}
			endTime, err := parseTimeOnly(weekDay.End)
			if err != nil{
				return nil, fmt.Errorf("calendar %s: invalid end time %q for %s: %w", calendar.Name, weekDay.End, weekDay.WeekDay, err)
			}

			if !endTime.After(beginTime) && !endTime.Equal(beginTime) {
				return nil, fmt.Errorf("calendar %s: end time %s must be after begin %s for %s", calendar.Name, weekDay.End, weekDay.Begin, weekDay.WeekDay)
			}

			// Проверяем перерывы
			err = validateBreaks(weekDay.Breaks, weekDay.Begin, weekDay.End, calendar.Name, weekDay.WeekDay)
			if err != nil {
				return nil, err
			}
		}

		// Валидация exceptions
		for _, exception := range calendar.Exceptions {
			_, err := time.Parse("2006-01-02", exception.Date)
			if err != nil {
				return nil, fmt.Errorf("calendar %s: invalid date %q in exception: %w", calendar.Name, exception.Date, err)
			}
			if !exception.DayOff {
				if exception.Begin == "" || exception.End == "" {
					return nil, fmt.Errorf("calendar %s: exception on %s is not day off but missing begin/end", calendar.Name, exception.Date)
				}
				beginTime, err := parseTimeOnly(exception.Begin)
				if err != nil {
					return nil, fmt.Errorf("calendar %s: invalid begin time %q in exception: %w", calendar.Name, exception.Begin, err)
				}
				endTime, err := parseTimeOnly(exception.End)
				if err != nil {
					return nil, fmt.Errorf("calendar %s: invalid end time %q in exception: %w", calendar.Name, exception.End, err)
				}

				if !endTime.After(beginTime) && !endTime.Equal(beginTime) {
					return nil, fmt.Errorf("calendar %s: end time %s must be after begin %s in exception", calendar.Name, exception.End, exception.Begin)
				}
			}
		}

		// Валидация vacations
		for _, vac := range calendar.Vacations {
			from, err := time.Parse("2006-01-02", vac.From)
			if err != nil {
				return nil, fmt.Errorf("calendar %s: invalid from date %q: %w", calendar.Name, vac.From, err)
			}
			to, err := time.Parse("2006-01-02", vac.To)
			if err != nil {
				return nil, fmt.Errorf("calendar %s: invalid to date %q: %w", calendar.Name, vac.To, err)
			}

			if to.Before(from) {
				return nil, fmt.Errorf("calendar %s: vacation end %s must be after start %s", calendar.Name, vac.To, vac.From)
			}
		}
	}
	return &calendarF, nil
}


// parseTimeOnly парсит время в формате HH:MM:SS.
func parseTimeOnly(s string) (time.Time, error) {
	return time.Parse("15:04:05", s)
}

// validateBreaks проверяет, что перерывы не пересекаются и находятся внутри смены.
func validateBreaks(breaks []Break, shiftBegin, shiftEnd, calendarName, weekDay string) error {
	if len(breaks) == 0 {
		return nil
	}

	// Парсим начало и конец смены
	startShift, err := parseTimeOnly(shiftBegin)
	if err != nil {
		return fmt.Errorf("calendar %s: invalid shift begin %q: %w", calendarName, shiftBegin, err)
	}

	endShift, err := parseTimeOnly(shiftEnd)
	if err != nil {
		return fmt.Errorf("calendar %s: invalid shift end %q: %w", calendarName, shiftEnd, err)
	}

	// Сортируем перерывы по началу
	sortedBreaks := make([]Break, len(breaks))
	copy(sortedBreaks, breaks)
	sort.Slice(sortedBreaks, func(i, j int) bool {
		// для сравнения используем строки, так как формат одинаковый, и лексикографический порядок совпадает с временным для HH:MM:SS
		return sortedBreaks[i].Begin < sortedBreaks[j].Begin
	})

	// Проверяем каждый перерыв
	for _, br := range sortedBreaks {
		brStart, err := parseTimeOnly(br.Begin)
		if err != nil {
			return fmt.Errorf("calendar %s: invalid break begin %q: %w", calendarName, brStart, err)
		}

		brEnd, err := parseTimeOnly(br.End)
		if err != nil {
			return fmt.Errorf("calendar %s: invalid break end %q: %w", calendarName, brEnd, err)
		}

		if brStart.Before(startShift) || brEnd.After(endShift) || brStart.Equal(brEnd) {
			return fmt.Errorf("calendar %s: break %s-%s is outside shift or zero-length", calendarName, br.Begin, br.End)
		}
	}

	//Проверяем пересечения попарно
	for i:=0; i < (len(sortedBreaks)-1); i++ {
		curEnd, _ := parseTimeOnly(sortedBreaks[i].End)
		nextStart, _ := parseTimeOnly(sortedBreaks[i+1].Begin)

		if !curEnd.Before(nextStart) && !curEnd.Equal(nextStart) {
			return fmt.Errorf("calendar %s: breaks %s-%s and %s-%s overlap or touch incorrectly", calendarName, sortedBreaks[i].End, sortedBreaks[i+1].Begin, sortedBreaks[i+1].End)
		}
	}
	return nil
}