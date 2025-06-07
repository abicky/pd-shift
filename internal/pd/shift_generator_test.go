package pd_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/abicky/pd-shift/internal/pd"
)

func TestShiftGenerator_Shifts(t *testing.T) {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		since          string
		until          string
		handoffTimes   []string
		include        []string
		nonWorkingDays []string

		want []pd.Shift
	}{
		{
			name:           "No include",
			since:          "2025-04-27",
			until:          "2025-04-30",
			handoffTimes:   []string{"10:00", "22:00"},
			include:        []string{},
			nonWorkingDays: []string{},
			want: []pd.Shift{
				*pd.NewShift(
					time.Date(2025, time.April, 27, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 30, 10, 0, 0, 0, jst),
				),
			},
		},
		{
			name:           "With working-days condition",
			since:          "2025-04-27",
			until:          "2025-04-30",
			handoffTimes:   []string{"10:00", "22:00"},
			include:        []string{"non-working-days", "working-days:22:00-10:00"},
			nonWorkingDays: []string{},
			want: []pd.Shift{
				*pd.NewShift(
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 30, 10, 0, 0, 0, jst),
				),
			},
		},
		{
			name:           "With common non-working-days",
			since:          "2025-04-27",
			until:          "2025-04-30",
			handoffTimes:   []string{"10:00", "22:00"},
			include:        []string{"non-working-days", "working-days:22:00-10:00"},
			nonWorkingDays: []string{"Sat", "Sunday"},
			want: []pd.Shift{
				*pd.NewShift(
					time.Date(2025, time.April, 27, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 30, 10, 0, 0, 0, jst),
				),
			},
		},
		{
			name:           "With JP holidays",
			since:          "2025-04-27",
			until:          "2025-04-30",
			handoffTimes:   []string{"10:00", "22:00"},
			include:        []string{"non-working-days", "working-days:22:00-10:00"},
			nonWorkingDays: []string{"Sat", "Sunday", "JP holidays"},
			want: []pd.Shift{
				*pd.NewShift(
					time.Date(2025, time.April, 27, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 30, 10, 0, 0, 0, jst),
				),
			},
		},
		{
			name:           "With JP holidays and custom holidays",
			since:          "2025-04-27",
			until:          "2025-04-30",
			handoffTimes:   []string{"10:00", "22:00"},
			include:        []string{"non-working-days", "working-days:22:00-10:00"},
			nonWorkingDays: []string{"Sat", "Sunday", "JP holidays", "April 28"},
			want: []pd.Shift{
				*pd.NewShift(
					time.Date(2025, time.April, 27, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 27, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 28, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 10, 0, 0, 0, jst),
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
				),
				*pd.NewShift(
					time.Date(2025, time.April, 29, 22, 0, 0, 0, jst),
					time.Date(2025, time.April, 30, 10, 0, 0, 0, jst),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg, err := pd.NewShiftGenerator(
				jst,
				tt.since,
				tt.until,
				tt.handoffTimes,
				tt.include,
				tt.nonWorkingDays,
			)
			if err != nil {
				t.Fatal(err)
			}

			shifts := make([]pd.Shift, 0)
			for shift := range sg.Shifts() {
				shifts = append(shifts, *shift)
			}

			if !reflect.DeepEqual(shifts, tt.want) {
				t.Errorf("shifts = %v, want %v", shifts, tt.want)
			}
		})
	}
}
