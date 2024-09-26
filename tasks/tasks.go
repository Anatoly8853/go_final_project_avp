package tasks

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const TimeFormat = "20060102"
const DisplayDateFormat = "02.01.2006"

type Task struct {
	//ID      int64  `db:"id,omitempty"`
	Id      string `json:"id,omitempty"`
	Date    string `db:"date,omitempty" json:"date,omitempty"`
	Title   string `db:"title,omitempty" json:"title" binding:"required"`
	Comment string `db:"comment,omitempty" json:"comment,omitempty"`
	Repeat  string `db:"repeat,omitempty" json:"repeat,omitempty"`
}

// parseDate парсинг даты в формате 20060102.
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse(TimeFormat, dateStr)
}

// nextDayDate следующая дата по дням.
func nextDayDate(start time.Time, days int, now time.Time) time.Time {
	next := start.AddDate(0, 0, days)
	for next.Before(now) {
		next = next.AddDate(0, 0, days)
	}
	return next
}

// nextMonthDate следующая дата по месяцам.
func nextMonthDate(start time.Time, days []int, months []int, now time.Time) time.Time {
	year, month := start.Year(), int(start.Month())

	// Если указаны месяцы, выбираем ближайший подходящий месяц
	if len(months) > 0 {
		sort.Ints(months)
		found := false

		// Ищем ближайший подходящий месяц в текущем году
		for _, m := range months {
			if m >= month {
				month = m
				found = true
				break
			}
		}

		// Если месяц не найден в текущем году, сдвигаемся на следующий год
		if !found {
			month = months[0]
			year++
		}
	}

	lastDay := getLastDayOfMonth(year, month)
	var closestDate time.Time

	// Перебираем дни, чтобы найти ближайший подходящий день
	for _, day := range days {
		var next time.Time

		if day == -1 {
			// Последний день месяца
			next = time.Date(year, time.Month(month), lastDay, 0, 0, 0, 0, time.UTC)
		} else if day == -2 {
			// Предпоследний день месяца
			next = time.Date(year, time.Month(month), lastDay-1, 0, 0, 0, 0, time.UTC)
		} else if day > 0 && day <= lastDay {
			// Обычный день месяца
			next = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		}

		// Проверяем ближайший день, который больше текущей даты (now)
		if !next.IsZero() && next.After(now) {
			if closestDate.IsZero() || next.Before(closestDate) {
				closestDate = next
			}
		} else if !next.IsZero() && next.Equal(now) {
			// Если день совпадает с текущей датой
			if closestDate.IsZero() || next.Before(closestDate) {
				closestDate = next
			}
		}
	}

	// Если дата не найдена в текущем месяце, переходим к следующему
	if closestDate.IsZero() {
		startNextMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		return nextMonthDate(startNextMonth.AddDate(0, 1, 0), days, months, now)
	}

	return closestDate
}

// NextDate обработка правила повторения.
func NextDate(now time.Time, dateStr string, repeat string) (string, error) {
	date, err := parseDate(dateStr)
	if err != nil {
		return "", fmt.Errorf("некорректная дата: %v", err)
	}

	if repeat == "" {
		return date.Format(TimeFormat), nil
		//return "", errors.New("пустое правило повторения")
	}

	switch {
	case strings.HasPrefix(repeat, "d "):
		daysStr := strings.TrimPrefix(repeat, "d ")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days < 1 || days > 400 {
			return "", errors.New("интервал дней должен быть от 1 до 400")
		}
		next := nextDayDate(date, days, now)
		return next.Format(TimeFormat), nil

	case repeat == "y":
		next := date.AddDate(1, 0, 0)
		if date.Month() == 2 && date.Day() == 29 && !isLeapYear(next.Year()) {
			next = time.Date(next.Year(), time.Month(3), 1, 0, 0, 0, 0, time.UTC)
		}
		for next.Before(now) {
			next = next.AddDate(1, 0, 0)
		}
		return next.Format(TimeFormat), nil

	case strings.HasPrefix(repeat, "w "):
		weekdays, err := parseDaysOrMonths(strings.Split(strings.TrimPrefix(repeat, "w "), ","))
		if err != nil {
			return "", err
		}
		if invalidWeekdays(weekdays) {
			return "", errors.New("недопустимый день недели")
		}
		nextDate := nextWeekdayDate(date, weekdays)
		for nextDate.Before(now) {
			nextDate = nextWeekdayDate(nextDate.AddDate(0, 0, 7), weekdays)
		}
		return nextDate.Format(TimeFormat), nil

	case strings.HasPrefix(repeat, "m "):
		parts := strings.Split(strings.TrimPrefix(repeat, "m "), " ")
		days, err := parseDaysOrMonths(strings.Split(parts[0], ","))
		if err != nil {
			return "", err
		}
		if invalidMonthDays(days) {
			return "", errors.New("недопустимый день месяца")
		}
		var months []int
		if len(parts) > 1 {
			months, err = parseDaysOrMonths(strings.Split(parts[1], ","))
			if err != nil {
				return "", err
			}
			if invalidMonths(months) {
				return "", errors.New("недопустимый месяц")
			}
		}
		nextDate := nextMonthDate(date, days, months, now)
		for !nextDate.After(now) {
			nextDate = nextMonthDate(nextDate.AddDate(0, 1, 0), days, months, now)
		}
		return nextDate.Format(TimeFormat), nil

	default:
		return "", errors.New("неподдерживаемый формат правила повторения")
	}
}

// nextWeekdayDate следующая ближайшая дата по дням недели.
func nextWeekdayDate(start time.Time, weekdays []int) time.Time {
	sort.Ints(weekdays)
	weekday := int(start.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	for _, wd := range weekdays {
		if wd >= weekday {
			return start.AddDate(0, 0, wd-weekday)
		}
	}

	// Если не нашли в текущей неделе, возвращаем в следующую
	return start.AddDate(0, 0, 7-weekday+weekdays[0])
}

// parseDaysOrMonths парсинг дней/месяцев из строки.
func parseDaysOrMonths(parts []string) ([]int, error) {
	var result []int
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < -2 || num > 31 {
			return nil, fmt.Errorf("недопустимое значение: %s", part)
		}
		result = append(result, num)
	}

	return result, nil
}

// invalidWeekdays проверка на недопустимые дни недели.
func invalidWeekdays(weekdays []int) bool {
	for _, wd := range weekdays {
		if wd < 1 || wd > 7 {
			return true
		}
	}
	return false
}

// invalidMonthDays проверка на недопустимые дни месяца.
func invalidMonthDays(days []int) bool {
	for _, day := range days {
		if day < -2 || day > 31 {
			return true
		}
	}
	return false
}

// invalidMonths проверка на недопустимые месяцы.
func invalidMonths(months []int) bool {
	for _, month := range months {
		if month < 1 || month > 12 {
			return true
		}
	}
	return false
}

// isLeapYear проверка на високосный год.
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// getLastDayOfMonth получение последнего дня месяца.
func getLastDayOfMonth(year int, month int) int {
	nextMonth := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	lastDay := nextMonth.AddDate(0, 0, -1).Day()
	return lastDay
}
