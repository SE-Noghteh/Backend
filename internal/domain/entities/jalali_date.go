package entities

import (
	"fmt"
	"time"
)

// JalaliDate is stored as a Jalali calendar date, never as a Gregorian SQL date.
type JalaliDate struct {
	Year  int `json:"year" example:"1380" validate:"required" minimum:"1" maximum:"3176"`
	Month int `json:"month" example:"1" validate:"required" minimum:"1" maximum:"12"`
	Day   int `json:"day" example:"1" validate:"required" minimum:"1" maximum:"31"`
}

func (d JalaliDate) String() string { return fmt.Sprintf("%04d/%02d/%02d", d.Year, d.Month, d.Day) }

// yearStart calculates Nowruz using the Jalali calendar's astronomical break years.
func yearStart(year int) (time.Time, bool) {
	breaks := [...]int{-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178}
	if year < 1 || year >= 3178 {
		return time.Time{}, false
	}
	gy, leapJ, previous := year+621, -14, breaks[0]
	for _, next := range breaks[1:] {
		jump := next - previous
		if year < next {
			n := year - previous
			leapJ += n/33*8 + (n%33+3)/4
			if jump%33 == 4 && jump-n == 4 {
				leapJ++
			}
			leapG := gy/4 - ((gy/100+1)*3)/4 - 150
			return time.Date(gy, time.March, 20+leapJ-leapG, 0, 0, 0, 0, time.UTC), true
		}
		leapJ += jump/33*8 + jump%33/4
		previous = next
	}
	return time.Time{}, false
}

func (d JalaliDate) Gregorian() (time.Time, error) {
	start, ok := yearStart(d.Year)
	if !ok || d.Month < 1 || d.Month > 12 || d.Day < 1 {
		return time.Time{}, fmt.Errorf("invalid Jalali birth date")
	}
	days := 31
	if d.Month > 6 {
		days = 30
	}
	if d.Month == 12 {
		next, valid := yearStart(d.Year + 1)
		if !valid {
			return time.Time{}, fmt.Errorf("unsupported Jalali year")
		}
		days = int(next.Sub(start).Hours()/24) - 336
	}
	if d.Day > days {
		return time.Time{}, fmt.Errorf("invalid Jalali birth date")
	}
	offset := (d.Month - 1) * 31
	if d.Month > 6 {
		offset = 186 + (d.Month-7)*30
	}
	return start.AddDate(0, 0, offset+d.Day-1), nil
}
