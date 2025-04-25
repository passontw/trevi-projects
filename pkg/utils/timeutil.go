package utils

import (
	"time"
)

const (
	TimeFormat    = "2006-01-02 15:04:05"
	DateFormat    = "2006-01-02"
	CompactFormat = "20060102150405"
)

func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

func ParseTime(str string) (time.Time, error) {
	return time.Parse(TimeFormat, str)
}

func ParseDate(str string) (time.Time, error) {
	return time.Parse(DateFormat, str)
}

func StartOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func EndOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
}

func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

func DaysBetween(start, end time.Time) int {
	hours := end.Sub(start).Hours()
	return int(hours / 24)
}
