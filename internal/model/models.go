package model

import (
	"fmt"
	"time"
)

// Непрерывный отрезок активности пользователя на одном компьютере
type Interval struct {
	RecID int64  `json:"rec_id"`
	User  string `json:"user"`
	Comp  string `json:"comp"`
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

// Интервал активности, приведённый к конкретному календарному дню. Используется после разрезания интервалов по границе суток
type DayInterval struct {
	Day time.Time `json:"day"`
	Interval
}

// Суточная запись табеля для одного пользователя и компьютера
type DailyRow struct {
	User    string `json:"user"`       
	Comp    string `json:"comp"`       
	Day     string `json:"day"`       
	DayType string `json:"day_type"`       
	BeginSchedule string `json:"begin_schedule"`
	EndSchedule   string `json:"end_schedule"`
	NormTimeMS int64  `json:"norm_time_ms"`   
	IsWorkDay  bool   `json:"is_work_day"`   
	BeginWork  string `json:"begin_work"`   
	EndWork    string `json:"end_work"`   
	ActiveTimeMS  int64  `json:"active_time_ms"`
	BeginWorkCut  string `json:"begin_work_cut"`
	EndWorkCut    string `json:"end_work_cut"`
	ActiveCutMS   int64  `json:"active_time_cut_ms"`
	DeviationMS   int64  `json:"deviation_ms"`
	DeviationClass string `json:"deviation_class"`
}

// Вспомогательные метода для Interval

// Парсинг начала интервала
func (i Interval) StartTimeParse() (time.Time, error) {
	return time.Parse(time.RFC3339, i.Start)
}

// Парсинг конца интервала
func (i Interval) StopTimeParse() (time.Time, error) {
	return time.Parse(time.RFC3339, i.Stop)
}

// Длительность интервала
func (i Interval) Duration() (time.Duration, error) {
	start, err := i.StartTimeParse()
	if (err != nil) {
		return 0, fmt.Errorf("invalid start time: %w", err)
	}
	stop, err := i.StopTimeParse()
	if (err != nil) {
		return 0, fmt.Errorf("invalid stop time: %w", err)
	}
	return stop.Sub(start), nil
}

// Проверка является ли интервал нулевым
func (i Interval) IsZero() (bool, error) {
	start, err := i.StartTimeParse()
	if (err != nil) {
		return false, fmt.Errorf("invalid start time: %w", err)
	}
	stop, err := i.StopTimeParse()
	if (err != nil) {
		return false, fmt.Errorf("invalid stop time: %w", err)
	}
	return start.Equal(stop), nil
}

// Проверяет корректность данных
func (i Interval) IsValid() error {
	if (i.User == "") {
		return fmt.Errorf("empty user")
	}
	if (i.Comp == "") {
		return fmt.Errorf("empty computer")
	}

	start, err := i.StartTimeParse()
	if (err != nil) {
		return fmt.Errorf("invalid start time: %w", err)
	}
	stop, err := i.StopTimeParse()
	if (err != nil) {
		return fmt.Errorf("invalid stop time: %w", err)
	}
	if (start.After(stop)) {
		return fmt.Errorf("stop is before start")
	}
	return nil
}

// Создаёт DayInterval из Interval и дня
func NewDayInterval(i Interval, day time.Time) DayInterval {
	return DayInterval{
		Interval: i,
		Day: day,
	}
}
