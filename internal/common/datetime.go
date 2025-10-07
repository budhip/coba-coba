package common

// DateLayout
const (
	DateFormatYYYYMMDD                  = "2006-01-02"
	DateFormatYYYYMM                    = "2006-01"
	DateFormatYYYYMMDDWithoutDash       = "20060102"
	DateFormatYYYYMMDDHHMMSSWithoutDash = "20060102150405"
	DateFormatDDMMYYYYWithoutDash       = "02012006"
	DateFormatDDMMMYYYY                 = "02-Jan-2006"
	DateFormatYYYYMMDDWithTime          = "2006-01-02 15:04:05"
	DateFormatDDMMMMYYYYWithTime        = "02-January-2006/15:04:05"
	DateFormatDDMMMMYYYYWithSpace       = "02 January 2006"
	DateFormatDDMMMYYYYWithSpace        = "02 Jan 2006"
	TimeFormatHHMM                      = "15:04"
	DateFormatHHMMSS                    = "15:04:05"
	DateFormatYYYYMMDDWithTimeAndOffset = "2006-01-02T15:04:05-07:00" // same as RFC3339/ISO8601
)

// HOUR FORMAT
const (
	HourFormat000000 = "00:00:00"
	HourFormat235959 = "23:59:59"
)

// TIMEZONE
const (
	TimezoneJakarta = "Asia/Jakarta"
)

// MAP TIMEZONE
var (
	MapTimezone = map[string]int{
		TimezoneJakarta: 7,
	}
)
