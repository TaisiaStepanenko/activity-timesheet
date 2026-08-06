package timesheet

import (
	"fmt"
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

const DateFormat = "2006-01-02"

func DistributeByComputer(intervals []model.Interval, shiftBegin, shiftEnd *time.Time, breaks []calendar.Break, loc *time.Location) (map[string][]model.Interval, error) {

	if len(intervals) == 0 {
		return nil, nil
	}

	user := intervals[0].User

	// Собираем все границы (начало и конец) всех интервалов
	type event struct {
		time time.Time
		isStart bool
		comp string
		interval model.Interval 
	}

	var events []event
	for _, interval := range intervals {
		startTime, err := interval.StartTimeParse()
		if err != nil {
			return  nil, err
		}

		stopTime, err := interval.StopTimeParse()
		if err != nil {
			return  nil, err
		}

		startTime = startTime.In(loc)
		stopTime = stopTime.In(loc)

		events = append(events, event{time: startTime, isStart: true, comp: interval.Comp, interval: interval})
		events = append(events, event{time: stopTime, isStart: false, comp: interval.Comp, interval: interval})
	}
	// Сортируем события по времени
	sort.Slice(events, func(i, j int) bool {
		
		if events[i].time.Equal(events[j].time) {
		
		// Если время равно, сначала конец, потом начало (чтобы не создавать нулевые отрезки)
		return !events[i].isStart && events[j].isStart
		}
		return  events[i].time.Before(events[j].time)
	})
	active  := make(map[string][]model.Interval) // comp -> активные интервалы
	result  := make(map[string][]model.Interval) // comp -> итоговые отрезки
	var lastTime time.Time
	for i, event := range events {
		// Если это не первое событие, и время изменилось, обрабатываем отрезок [lastTime, ev.t)
		if i > 0 && !event.time.Equal(lastTime) {
			
			// Определяем, какие компьютеры активны на отрезке [lastTime, ev.t)
			var activeComps []string
			for comp, intervals := range active {
				// Проверяем, есть ли активный интервал на этом отрезке
				if len(intervals) > 0 {
					activeComps = append(activeComps, comp)
				} 
			}
			// Если активен хотя бы один компьютер, выбираем приоритетный
			if len(activeComps) > 0 {
				// Находим компьютер с наименьшим временем начала среди всех его активных интервалов
				var chosenComp string
				var chosenStart time.Time
				for _, comp := range activeComps {
					for _, interval := range active[comp] {
						start, _ := interval.StartTimeParse()
						start = start.In(loc)
						if chosenComp == "" || start.Before(chosenStart) || (start.Equal(chosenStart) && comp < chosenComp) {
							chosenComp = comp
							chosenStart = start
						}
					}
				}
				// Создаём отрезок [lastTime, ev.t) и добавляем его выбранному компьютеру
				seg := model.Interval{
					User: user,
					Comp: chosenComp,
					Start: lastTime.Format(time.RFC3339),
					Stop: event.time.Format(time.RFC3339),
				}
				// Если требуется обрезка по смене, обрезаем отрезок
				if shiftBegin != nil && shiftEnd != nil {
					segStart, segEnd, ok := Intersect(lastTime, event.time, *shiftBegin, *shiftEnd)
					if  !ok || segStart.Equal(segEnd) {
						// Отрезок вне смены – пропускаем и переходим к следующему событию, но всё равно обработывем оставшиеся события
					} else {
						// Вычитаем перерывы 
						cutSegments := SubtractBreaks(segStart, segEnd, breaks)
						for _, cutSeg := range cutSegments {
							result[chosenComp] = append(result[chosenComp], cutSeg)
						}
					}
				} else {
					// Без обрезки - добавляем как есть
					result[chosenComp] = append(result[chosenComp], seg)
				}
			} 
		}
		if event.isStart {
			active[event.comp] = append(active[event.comp], event.interval)
		} else {
			// Удаляем этот интервал из активных
			for i, interval := range active[event.comp] {
				// Сравниваем по RecID или по указателю? Будем сравнивать по содержанию (start, stop, comp, user)
				if interval.Start == event.interval.Start && interval.Stop == event.interval.Stop && interval.Comp == event.interval.Comp && interval.User == event.interval.User {
					active[event.comp] = append(active[event.comp][:i], active[event.comp][i+1:]...)
					break
				}
			}
		}
		lastTime = event.time
	}
	return result, nil
	
}

// CalculateDay вычисляет суточные записи табеля для одного пользователя и даты.
func CalculateDay(user string, date time.Time, intervals []model.Interval, calendarFile *calendar.CalendarFile, tolerance time.Duration) ([]model.DailyRow, error) {
	// Получаем информацию о дне из календаря
	dayInfo, err := calendar.GetDayInfo(calendarFile, user, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get day info: %w", err)
	}

	// Получаем локацию из календаря
	loc, err := calendar.ParseTimezone(calendarFile.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	date = date.In(loc)

	// Если интервалов нет, возвращаем одну строку с нулевыми значениями
	if len(intervals) == 0 {
		row := model.DailyRow{
			User:          user,
			Comp:          "",
			Day:           date.Format(DateFormat),
			DayType:       string(dayInfo.DayType),
			IsWorkDay:     dayInfo.IsWorkDay,
			NormTimeMS:    int64(dayInfo.NormTime / time.Millisecond),
			BeginSchedule: dayInfo.Begin,
			EndSchedule:   dayInfo.End,
		}
		return []model.DailyRow{row}, nil
	}

	// Распределяем интервалы без обрезки
	rawDist, err := DistributeByComputer(intervals, nil, nil, nil, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to distribute raw intervals: %w", err)
	}

	// Распределяем интервалы с обрезкой, если день рабочий
	var cutDist map[string][]model.Interval
	if dayInfo.IsWorkDay {
		shiftBegin, err := time.Parse(time.RFC3339, dayInfo.Begin)
		if err != nil {
			return nil, fmt.Errorf("invalid shift begin: %w", err)
		}
		shiftEnd, err := time.Parse(time.RFC3339, dayInfo.End)
		if err != nil {
			return nil, fmt.Errorf("invalid shift end: %w", err)
		}
		cutDist, err = DistributeByComputer(intervals, &shiftBegin, &shiftEnd, dayInfo.Breaks, loc)
		if err != nil {
			return nil, fmt.Errorf("failed to distribute cut intervals: %w", err)
		}
	} else {
		cutDist = make(map[string][]model.Interval)
	}

	// Собираем список всех компьютеров
	compSet := make(map[string]bool)
	for comp := range rawDist {
		compSet[comp] = true
	}
	for comp := range cutDist {
		compSet[comp] = true
	}
	if len(compSet) == 0 {
		// Если нет компьютеров, создаём одну запись с пустым comp
		compSet[""] = true
	}

	var rows []model.DailyRow
	for comp := range compSet {
		// Вычисляем факт для этого компьютера из rawDist
		rawSegments := rawDist[comp]
		var activeTime time.Duration
		var beginWork, endWork time.Time
		if len(rawSegments) > 0 {
			for _, seg := range rawSegments {
				d, err := seg.Duration()
				if err != nil {
					return nil, fmt.Errorf("duration error: %w", err)
				}
				activeTime += d
			}
			start, err := rawSegments[0].StartTimeParse()
			if err != nil {
				return nil, err
			}
			stop, err := rawSegments[len(rawSegments)-1].StopTimeParse()
			if err != nil {
				return nil, err
			}
			beginWork = start
			endWork = stop
		}

		// Вычисляем обрезанные значения из cutDist
		cutSegments := cutDist[comp]
		var activeCut time.Duration
		var beginCut, endCut time.Time
		if dayInfo.IsWorkDay && len(cutSegments) > 0 {
			for _, seg := range cutSegments {
				d, err := seg.Duration()
				if err != nil {
					return nil, fmt.Errorf("duration error: %w", err)
				}
				activeCut += d
			}
			start, err := cutSegments[0].StartTimeParse()
			if err != nil {
				return nil, err
			}
			stop, err := cutSegments[len(cutSegments)-1].StopTimeParse()
			if err != nil {
				return nil, err
			}
			beginCut = start
			endCut = stop
		}

		// Для calendar_disabled activeCut = activeTime
		if dayInfo.DayType == calendar.DayCalendarDisabled {
			activeCut = activeTime
		}

		// Определяем класс отклонения
		deviationClass := ""
		deviation := time.Duration(0)

		if dayInfo.DayType == calendar.DayCalendarDisabled {
			deviationClass = "no_calendar"
		} else if !dayInfo.IsWorkDay {
			// Нерабочий день
			if activeCut > 0 {
				switch dayInfo.DayType {
				case calendar.DayHoliday:
					deviationClass = "work_on_holiday"
				case calendar.DayOffWeek, calendar.DayOffException:
					deviationClass = "work_on_day_off"
				case calendar.DayVacation:
					deviationClass = "work_on_vacation"
				default:
					deviationClass = "work_on_day_off"
				}
			} else {
				deviationClass = ""
			}
		} else {
			// Рабочий день
			norm := dayInfo.NormTime
			dev := activeCut - norm
			if dev < -tolerance {
				deviationClass = "underwork"
				deviation = dev
			} else if dev > tolerance {
				deviationClass = "overwork"
				deviation = dev
			} else {
				deviationClass = "ok"
				deviation = 0
			}
		}

		// Формируем строку DailyRow
		row := model.DailyRow{
			User:           user,
			Comp:           comp,
			Day:            date.Format(DateFormat),
			DayType:        string(dayInfo.DayType),
			BeginSchedule:  dayInfo.Begin,
			EndSchedule:    dayInfo.End,
			NormTimeMS:     int64(dayInfo.NormTime / time.Millisecond),
			IsWorkDay:      dayInfo.IsWorkDay,
			BeginWork:      formatTime(beginWork),
			EndWork:        formatTime(endWork),
			ActiveTimeMS:   int64(activeTime / time.Millisecond),
			BeginWorkCut:   formatTime(beginCut),
			EndWorkCut:     formatTime(endCut),
			ActiveCutMS:    int64(activeCut / time.Millisecond),
			DeviationMS:    int64(deviation / time.Millisecond),
			DeviationClass: deviationClass,
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// formatTime возвращает RFC3339 строку или пустую строку для нулевого времени.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

