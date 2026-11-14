package postgres

import (
	"testing"
	"time"
)

func TestAdultBirthCutoffMatchesFullBirthday(t *testing.T) {
	for _, tc := range []struct{ now, want time.Time }{
		{time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), time.Date(2006, 2, 28, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), time.Date(2008, 2, 28, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2008, 3, 1, 0, 0, 0, 0, time.UTC)},
	} {
		cutoff, err := adultBirthCutoff(tc.now)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := cutoff.Gregorian()
		if err != nil || !actual.Equal(tc.want) {
			t.Fatalf("%v: cutoff %v, want %v (%v)", tc.now, actual, tc.want, err)
		}
	}
}
