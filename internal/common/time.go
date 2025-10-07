package common

import (
	"time"
)

const (
	defaultTimeZone = TimezoneJakarta
)

var addedNanoSecond = Now().Nanosecond()

func CurrentTime() *time.Time {
	currentTime := Now()
	return &currentTime
}

func YesterdayTime() *time.Time {
	yesterday := Now().AddDate(0, 0, -1)
	return &yesterday
}

func Now() time.Time {
	loc, err := time.LoadLocation(defaultTimeZone)
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}

func GetLocation() *time.Location {
	loc, err := time.LoadLocation(defaultTimeZone)
	if err != nil {
		return Now().Location()
	}
	return loc
}

func NowZeroTime() (now time.Time, err error) {
	now, err = time.ParseInLocation(DateFormatYYYYMMDD, Now().Format(DateFormatYYYYMMDD), Now().Location())
	return
}

func NowWithTime() (now time.Time, err error) {
	now, err = time.ParseInLocation(DateFormatYYYYMMDDWithTime, Now().Format(DateFormatYYYYMMDDWithTime), Now().Location())
	return
}

func NowWithNanoTime() (now time.Time, err error) {
	now, err = time.ParseInLocation(time.RFC3339Nano, Now().Format(time.RFC3339Nano), Now().Location())
	return
}

func ParseStringToDatetime(layout, dateString string) (time.Time, error) {
	date, err := time.ParseInLocation(layout, dateString, Now().Location())
	return date, err
}

func FormatDatetimeToString(date time.Time, formatLayout string) string {
	return date.Format(formatLayout)
}

func FormatDatetimeToStringInLocalTime(date time.Time, formatLayout string) string {
	return date.In(GetLocation()).Format(formatLayout)
}

func ParseStringDateToDateWithTimeNow(layout, date string) (time.Time, error) {
	timeParam, err := ParseStringToDatetime(layout, date)
	if err != nil {
		return time.Time{}, err
	}
	datetime := time.Date(timeParam.Year(), timeParam.Month(), timeParam.Day(), Now().Hour(), Now().Minute(), Now().Second(), addedNanoSecond, Now().Location())

	return datetime, nil
}

// NowEndOfDay returns the current time adjusted to the end of the same day.
// It sets the time to 23:59:59 and the maximum nanosecond value (999999999),
// effectively representing the last moment of the current day.
func NowEndOfDay() time.Time {
	now := Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	return endOfDay
}

// GetTotalDiffDayBetweenTwoDate ...
func GetTotalDiffDayBetweenTwoDate(dateFrom, dateTo time.Time) float64 {
	t1 := time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, dateFrom.Location())
	t2 := time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 0, 0, 0, 0, dateTo.Location())
	diff := (t2.Sub(t1).Hours() / 24) / 1

	return diff
}

func IsTodayAfterDate(date time.Time) bool {
	now, _ := NowZeroTime()
	if date.Location().String() != defaultTimeZone {
		// convert date to local time
		date = date.In(GetLocation())
	}

	// set date to start of day
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, GetLocation())
	return date.After(now)
}
