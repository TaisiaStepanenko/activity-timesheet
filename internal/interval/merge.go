package interval

import (
	"sort"
	"time"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
)

// Merge объединяет пересекающиеся и стыкующиеся интервалы
func Merge(intervals []model.Interval) []model.Interval {
	if (len(intervals) == 0) {
		return nil
	}

	// Копируем, чтобы не мутировать исходный срез
	sorted := make([]model.Interval, len(intervals))
	copy(sorted, intervals)

	// Сортируем по Start, при равенстве по Stop
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Start != sorted[j].Start {
			return sorted[i].Start < sorted[j].Start
		}
		return sorted[i].Stop < sorted[j].Stop
	})

	merged := make([]model.Interval, 0, len(sorted))
	cur := sorted[0]

	for _, next := range sorted[1:] {
		// Если текущий интервал заканчивается раньше, чем начинается следующий (стык – если end == start – объединяем)
		if cur.Stop < next.Start {
			merged = append(merged, cur)
			cur = next
		}else {
			// Пересекаются или стыкуются - объединяем
			if next.Stop > cur.Stop {
				cur.Stop = next.Stop
			}
		}
	}
	merged = append(merged, cur)
	return merged  
}

// SplitByDay разрезает интервал по границе суток в заданной временной зоне.
func SplitByDay(interval model.Interval, loc *time.Location) ([]model.DayInterval, error) {
	start, err := interval.StartTimeParse()
	if err != nil {
		return nil, err
	}
	stop, err := interval.StopTimeParse()
	if err != nil {
		return nil, err
	}

	start = start.In(loc)
	stop = stop.In(loc)

	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	stopDay := time.Date(stop.Year(), stop.Month(), stop.Day(), 0, 0, 0, 0, loc)

	// Если интервал внутри одного дня
	if startDay.Equal(stopDay) {
		return []model.DayInterval{model.NewDayInterval(interval, startDay)}, nil
	}

	var result []model.DayInterval
	currentStart := start
	for currentStart.Before(stop) {
		nextMidnight := time.Date(currentStart.Year(), currentStart.Month(), currentStart.Day()+1, 0, 0, 0, 0, loc)
		if nextMidnight.After(stop) {
			nextMidnight = stop
		}
		part := model.Interval{
			RecID: interval.RecID,
			User:  interval.User,
			Comp:  interval.Comp,
			Start: currentStart.Format(time.RFC3339),
			Stop:  nextMidnight.Format(time.RFC3339),
		}
		day := time.Date(currentStart.Year(), currentStart.Month(), currentStart.Day(), 0, 0, 0, 0, loc)
		result = append(result, model.NewDayInterval(part, day))
		currentStart = nextMidnight
	}
	return result, nil
}