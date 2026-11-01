package entities

import "testing"

func TestJalaliDateGregorian(t *testing.T) {
	for _, tt := range []struct {
		date JalaliDate
		want string
	}{
		{JalaliDate{1403, 1, 1}, "2024-03-20"},
		{JalaliDate{1403, 12, 30}, "2025-03-20"},
		{JalaliDate{1404, 1, 1}, "2025-03-21"},
		{JalaliDate{1405, 1, 1}, "2026-03-21"},
		{JalaliDate{1379, 12, 30}, "2001-03-20"},
	} {
		got, err := tt.date.Gregorian()
		if err != nil || got.Format("2006-01-02") != tt.want {
			t.Errorf("%v: got %v, %v; want %s", tt.date, got, err, tt.want)
		}
	}
}
func TestJalaliDateRejectsInvalidDays(t *testing.T) {
	for _, date := range []JalaliDate{{1404, 12, 30}, {1403, 7, 31}, {1403, 0, 1}, {1403, 13, 1}, {1403, 1, 0}, {0, 1, 1}, {3178, 1, 1}} {
		if _, err := date.Gregorian(); err == nil {
			t.Errorf("accepted %v", date)
		}
	}
}
