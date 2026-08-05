package timesheet

import (
	"sort"
	"time"

	"github.com/TaisiaStepanenko/activity-timesheet/internal/calendar"
	"github.com/TaisiaStepanenko/activity-timesheet/internal/model"
)

// Intersect возвращает пересечение двух интервалов [aStart, aEnd] и [bStart, bEnd].
func Intersect(aStart, aEnd, bStart, bEnd time.Time) (time.Time, time.Time, bool) {
	maxStart := aStart
	if bStart.After(maxStart) {
		maxStart = bStart
	}
	minEnd := aEnd
	if bEnd.Before(minEnd) {
		minEnd = bEnd
	}
	if maxStart.Before(minEnd) || maxStart.Equal(minEnd) {
		return maxStart, minEnd, true
	}
	return time.Time{}, time.Time{}, false
}

// SubtractBreaks вычитает перерывы из отрезка [segmentStart, segmentEnd]
func SubtractBreaks(segmentStart, segmentEnd time.Time, breaks []calendar.Break) []model.Interval {
	if len(breaks) == 0 {
		return []model.Interval{
			{Start: segmentStart.Format(time.RFC3339), Stop: segmentEnd.Format(time.RFC3339)},
		}
	}

	// Сортируем перерывы по времени начала
	sortedBreaks := make([]calendar.Break, len(breaks))
	copy(sortedBreaks, breaks)
	sort.Slice(sortedBreaks, func(i, j int) bool {
		return sortedBreaks[i].Begin < sortedBreaks[j].Begin
	})

	var result []model.Interval
	curStart := segmentStart

	for _, br := range sortedBreaks {
		brStart, _ := time.Parse("15:04:05", br.Begin)
        brEnd, _ := time.Parse("15:04:05", br.End)
        brStartFull := time.Date(segmentStart.Year(), segmentStart.Month(), segmentStart.Day(), brStart.Hour(), brStart.Minute(), brStart.Second(), 0, segmentStart.Location())
        brEndFull := time.Date(segmentStart.Year(), segmentStart.Month(), segmentStart.Day(), brEnd.Hour(), brEnd.Minute(), brEnd.Second(), 0, segmentStart.Location())

		// Проверяем, пересекает ли перерыв текущий интервал [curStart, segmentEnd]
		if brEndFull.Before(curStart) || brEndFull.Equal(curStart) || brStartFull.After(segmentEnd) || brStartFull.Equal(segmentEnd) {
            continue
        }

		// Если перерыв начинается позже curStart, добавляем часть до перерыва
        if brStartFull.After(curStart) {
            result = append(result, model.Interval{
                Start: curStart.Format(time.RFC3339),
                Stop:  brStartFull.Format(time.RFC3339),
            })
        }
        // Сдвигаем curStart на конец перерыва
        if brEndFull.After(curStart) {
            curStart = brEndFull
        }
    }

	// После всех перерывов добавляем оставшийся хвост
	if curStart.Before(segmentEnd) {
		result = append(result, model.Interval{
			Start: curStart.Format(time.RFC3339),
			Stop:  segmentEnd.Format(time.RFC3339),
		})
	}

	return result
}

// CutByShift обрезает интервалы активности по смене и перерывам.
func CutByShift(intervals []model.Interval, shiftBegin, shiftEnd time.Time, breaks []calendar.Break, loc *time.Location) (activeCut time.Duration, beginCut time.Time, endCut time.Time, err error) {
	var pieces []model.Interval

	for _, interval := range intervals {
		start, err := interval.StartTimeParse()
		if err != nil {
			return 0, time.Time{}, time.Time{}, err
		}
		stop, err := interval.StopTimeParse()
		if err != nil {
			return 0, time.Time{}, time.Time{}, err
		}

		start = start.In(loc)
		stop = stop.In(loc)

		// Приводим смену к той же локации
		shiftBeginLocal := shiftBegin.In(loc)
		shiftEndLocal := shiftEnd.In(loc)

		segmentStart, segmentEnd, ok := Intersect(start, stop, shiftBeginLocal, shiftEndLocal)
		if !ok || segmentStart.Equal(segmentEnd) {
			continue
		}

		segments := SubtractBreaks(segmentStart, segmentEnd, breaks)
		pieces = append(pieces, segments...)
	}

	if len(pieces) == 0 {
		return 0, time.Time{}, time.Time{}, nil
	}

	// Сортируем по Start
	sort.Slice(pieces, func(i, j int) bool {
		return pieces[i].Start < pieces[j].Start
	})

	// Суммируем длительности
	var total time.Duration
	for _, p := range pieces {
		d, err := p.Duration()
		if err != nil {
			return 0, time.Time{}, time.Time{}, err
		}
		total += d
	}

	// Начало первого, конец последнего
	begin, err := pieces[0].StartTimeParse()
	if err != nil {
		return 0, time.Time{}, time.Time{}, err
	}
	end, err := pieces[len(pieces)-1].StopTimeParse()
	if err != nil {
		return 0, time.Time{}, time.Time{}, err
	}

	return total, begin, end, nil
}