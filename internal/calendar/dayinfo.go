package calendar

import (
	"fmt"
	"strconv"
	"time"
)

type DayType string

const (
	DayWorkWeek         DayType = "work_week"
	DayOffWeek          DayType = "day_off_week"
	DayHoliday          DayType = "holiday"
	DayVacation         DayType = "vacation"
	DayException        DayType = "exception"
	DayOffException     DayType = "day_off_exception"
	DayCalendarDisabled DayType = "calendar_disabled"
)

type DayInfo struct {
	DayType   DayType       `json:"day_type"`
	Begin     string        `json:"begin,omitempty"`
	End       string        `json:"end,omitempty"`
	Breaks    []Break       `json:"breaks,omitempty"`
	NormTime  time.Duration `json:"norm_time"`
	IsWorkDay bool 			`json:"is_work_day"`
}

// GetUserCalendar возвращает календарь для пользователя.
func (calendarFile *CalendarFile) GetUserCalendar(user string) *Calendar {
	for _, calendar := range calendarFile.Calendars {
		for _, u := range calendar.Users {
			if u == user {
				return &calendar
			}
		}
	}
	return nil
}

// parseTimezone, чтобы превратить строку часового пояса (например, "+03:00" или "-05:00") в объект *time.Location
func parseTimezone(tz string) (*time.Location, error) {
    if len(tz) == 0 {
        return nil, fmt.Errorf("timezone is empty")
    }
    sign := 1
	switch tz[0] {
	case '-':
		sign = -1
        tz = tz[1:]
	case '+':
		tz = tz[1:]
	}

    if len(tz) != 5 || tz[2] != ':' {
        return nil, fmt.Errorf("invalid timezone format: %s, expected ±HH:MM", tz)
    }

	// Извлечение часов и минут
    h, err := strconv.Atoi(tz[0:2])
    if err != nil {
        return nil, fmt.Errorf("invalid hour: %w", err)
    }
    m, err := strconv.Atoi(tz[3:5])
    if err != nil {
        return nil, fmt.Errorf("invalid minute: %w", err)
    }

	// Вычисляем смещение в секундах
    offset := sign * (h*3600 + m*60)
    return time.FixedZone(tz, offset), nil
}


// GetDayInfo определяет тип дня и норму для указанного пользователя и даты.
func GetDayInfo(calendarFile *CalendarFile, user string, date time.Time) (*DayInfo, error) {
	// Находим календарь пользователя
	userCalendar := calendarFile.GetUserCalendar(user)

	// Если нет календаря или disabled -> calendar_disabled
	if userCalendar == nil || userCalendar.Enabled == false {
		return &DayInfo{
			DayType: DayCalendarDisabled,
			IsWorkDay: false,
			NormTime: 0,
		}, nil
	}

	tz := calendarFile.Timezone
	if tz == "" {
		return nil, fmt.Errorf("timezone is empty")
	}

	loc, err := parseTimezone(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	date = date.In(loc)
	

	// Проверить исключения
	for _, exception := range userCalendar.Exceptions {
		excDate, err := time.Parse("2006-01-02", exception.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid exception date %s: %w", exception.Date, err)
		}

		if excDate.Year() == date.Year() && excDate.Month() == date.Month() && excDate.Day() == date.Day() {
			if exception.DayOff {
				return &DayInfo{
					DayType: DayOffException,
					IsWorkDay: false,
					NormTime: 0,
				}, nil
			} else {
				begin, err := time.ParseInLocation("15:04:05", exception.Begin, loc)
				if err != nil {
					return nil, fmt.Errorf("invalid exception begin: %w", err)
				}
				end, err := time.ParseInLocation("15:04:05", exception.End, loc)
				if err != nil {
					return nil, fmt.Errorf("invalid exception end: %w", err)
				}

				beginTime := time.Date(date.Year(), date.Month(), date.Day(), begin.Hour(), begin.Minute(), begin.Second(), 0, loc)
				endTime := time.Date(date.Year(), date.Month(), date.Day(), end.Hour(), end.Minute(), end.Second(), 0, loc)				
				norm := endTime.Sub(beginTime)

				return &DayInfo{
					DayType: DayException,
					Begin: beginTime.Format(time.RFC3339),
					End: endTime.Format(time.RFC3339),
					Breaks: nil,
					NormTime: norm,
					IsWorkDay: true,
				}, nil
			}
		}
	}

	// Проверка праздников 

	for _, holiday := range userCalendar.Holidays {
		if holiday.Month == int(date.Month()) && holiday.Day == date.Day() {
			return &DayInfo{
				DayType: DayHoliday,
				IsWorkDay: false,
				NormTime: 0,
			}, nil 
		}
	}

	// Проверка отпусков 
	for _, vacation := range userCalendar.Vacations {
		from, err := time.Parse("2006-01-02", vacation.From)
		if err != nil {
			return nil, fmt.Errorf("invalid vacation from %s: %w", vacation.From, err)
		}
		to, err := time.Parse("2006-01-02", vacation.To)
		if err != nil {
			return nil, fmt.Errorf("invalid vacation to %s: %w", vacation.To, err)
		}

		// Проверяем попадает ли дата в интервал c from до to 
		if (date.Equal(from) || date.After(from)) && (date.Equal(to) || date.Before(to)) {
			return &DayInfo{
				DayType: DayVacation,
				IsWorkDay: false,
				NormTime: 0,
			}, nil
		}
	}

	// Проверка дня недели 

	weekdayMap := map[string]string{
		"Monday": "mon", "Tuesday": "tue", "Wednesday": "wed",
		"Thursday": "thu", "Friday": "fri", "Saturday": "sat", "Sunday": "sun",
	}

	weekdayKey := weekdayMap[date.Weekday().String()]
	var wd *WeekDay
	for i := range userCalendar.Week {
		if userCalendar.Week[i].WeekDay == weekdayKey {
			wd = &userCalendar.Week[i]
			break
		}
	}

	if wd == nil || wd.DayOff {
		return &DayInfo{
			DayType: DayOffWeek,
			IsWorkDay: false,
			NormTime: 0,
		}, nil
	}

	// Рабочий день недели
	beginTime, err := time.ParseInLocation("15:04:05", wd.Begin, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid weekday begin %s: %w", wd.Begin, err)
	}
	endTime, err := time.ParseInLocation("15:04:05", wd.End, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid weekday end %s: %w", wd.End, err)
	}
	begin := time.Date(date.Year(), date.Month(), date.Day(), beginTime.Hour(), beginTime.Minute(), beginTime.Second(), 0, loc)
	end := time.Date(date.Year(), date.Month(), date.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), 0, loc)
	norm := end.Sub(begin)

	// Вычитаем перерывы
	var totalBreaks time.Duration

	for _, br := range wd.Breaks {
		brStart, err := time.ParseInLocation("15:04:05", br.Begin, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid break begin %s: %w", br.Begin, err)
		}
		brEnd, err := time.ParseInLocation("15:04:05", br.End, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid break end %s: %w", br.End, err)
		}
		brStartTime := time.Date(date.Year(), date.Month(), date.Day(), brStart.Hour(), brStart.Minute(), brStart.Second(), 0, loc)
		brEndTime := time.Date(date.Year(), date.Month(), date.Day(), brEnd.Hour(), brEnd.Minute(), brEnd.Second(), 0, loc)

		totalBreaks += brEndTime.Sub(brStartTime)
	}
	norm -= totalBreaks

	// копируем перерывы для возврата 
	breaksCopy := make([]Break, len(wd.Breaks))
	copy(breaksCopy, wd.Breaks)

	return &DayInfo{
		DayType: DayWorkWeek,
		Begin: begin.Format(time.RFC3339),
		End: end.Format(time.RFC3339),
		Breaks: breaksCopy,
		NormTime:  norm,
		IsWorkDay: true,
	}, nil
}