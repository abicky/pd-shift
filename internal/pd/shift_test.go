package pd_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/abicky/pd-shift/internal/pd"
)

type entry struct {
	start string
	end   string
	user  string
}

func newScheduleEntryIter(t *testing.T, name string, tz *time.Location, entries []entry) *pd.ScheduleEntryIter {
	rsEntries := make([]pagerduty.RenderedScheduleEntry, len(entries))
	for i, e := range entries {
		rsEntries[i] = pagerduty.RenderedScheduleEntry{
			Start: e.start,
			End:   e.end,
			User: pagerduty.APIObject{
				Summary: e.user,
			},
		}
	}

	iter, err := pd.NewScheduleEntryIter(name, tz, rsEntries)
	if err != nil {
		t.Fatal(err)
	}
	return iter
}

func TestShift_AddDetails(t *testing.T) {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2025, time.April, 27, 10, 0, 0, 0, jst)
	end1 := time.Date(2025, time.April, 27, 22, 0, 0, 0, jst)
	end2 := time.Date(2025, time.April, 28, 10, 0, 0, 0, jst)
	scheduleName := "primary"

	tests := []struct {
		name  string
		iter  *pd.ScheduleEntryIter
		want1 []pd.ShiftDetail
		want2 []pd.ShiftDetail
	}{
		{
			name: "Shift has only one user",
			iter: newScheduleEntryIter(t, scheduleName, jst, []entry{
				{
					start: "2025-04-21T10:00:00+09:00",
					end:   "2025-04-28T10:00:00+09:00",
					user:  "user1",
				},
			}),
			want1: []pd.ShiftDetail{
				{
					User:       "user1",
					Start:      start,
					End:        end1,
					Proportion: 1,
				},
			},
			want2: []pd.ShiftDetail{
				{
					User:       "user1",
					Start:      end1,
					End:        end2,
					Proportion: 1,
				},
			},
		},
		{
			name: "Shift has two users",
			iter: newScheduleEntryIter(t, scheduleName, jst, []entry{
				{
					start: "2025-04-21T10:00:00+09:00",
					end:   "2025-04-27T13:00:00+09:00",
					user:  "user1",
				},
				{
					start: "2025-04-27T13:00:00+09:00",
					end:   "2025-04-28T10:00:00+09:00",
					user:  "user2",
				},
			}),
			want1: []pd.ShiftDetail{
				{
					User:       "user1",
					Start:      start,
					End:        time.Date(2025, time.April, 27, 13, 0, 0, 0, jst),
					Proportion: 0.25,
				},
				{
					User:       "user2",
					Start:      time.Date(2025, time.April, 27, 13, 0, 0, 0, jst),
					End:        end1,
					Proportion: 0.75,
				},
			},
			want2: []pd.ShiftDetail{
				{
					User:       "user2",
					Start:      end1,
					End:        end2,
					Proportion: 1,
				},
			},
		},
		{
			name: "Shift has three users",
			iter: newScheduleEntryIter(t, scheduleName, jst, []entry{
				{
					start: "2025-04-21T10:00:00+09:00",
					end:   "2025-04-27T13:00:00+09:00",
					user:  "user1",
				},
				{
					start: "2025-04-27T13:00:00+09:00",
					end:   "2025-04-27T19:00:00+09:00",
					user:  "user2",
				},
				{
					start: "2025-04-27T19:00:00+09:00",
					end:   "2025-04-28T10:00:00+09:00",
					user:  "user3",
				},
			}),
			want1: []pd.ShiftDetail{
				{
					User:       "user1",
					Start:      start,
					End:        time.Date(2025, time.April, 27, 13, 0, 0, 0, jst),
					Proportion: 0.25,
				},
				{
					User:       "user2",
					Start:      time.Date(2025, time.April, 27, 13, 0, 0, 0, jst),
					End:        time.Date(2025, time.April, 27, 19, 0, 0, 0, jst),
					Proportion: 0.5,
				},
				{
					User:       "user3",
					Start:      time.Date(2025, time.April, 27, 19, 0, 0, 0, jst),
					End:        end1,
					Proportion: 0.25,
				},
			},
			want2: []pd.ShiftDetail{
				{
					User:       "user3",
					Start:      end1,
					End:        end2,
					Proportion: 1,
				},
			},
		},
		{
			name: "Boundaries",
			iter: newScheduleEntryIter(t, scheduleName, jst, []entry{
				{
					start: "2025-04-21T10:00:00+09:00",
					end:   "2025-04-27T09:59:59+09:00",
					user:  "user1",
				},
				{
					start: "2025-04-27T09:59:59+09:00",
					end:   "2025-04-27T10:00:00+09:00",
					user:  "user2",
				},
				{
					start: "2025-04-27T10:00:00+09:00",
					end:   "2025-04-27T22:00:00+09:00",
					user:  "user3",
				},
			}),
			want1: []pd.ShiftDetail{
				{
					User:       "user3",
					Start:      start,
					End:        end1,
					Proportion: 1,
				},
			},
			want2: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1 := pd.NewShift(start, end1)
			s1.AddDetails(tt.iter)
			if !reflect.DeepEqual(s1.Details[scheduleName], tt.want1) {
				t.Errorf("s1.Details[%q] = %v, want %v", scheduleName, s1.Details[scheduleName], tt.want1)
			}

			s2 := pd.NewShift(end1, end2)
			s2.AddDetails(tt.iter)
			if !reflect.DeepEqual(s2.Details[scheduleName], tt.want2) {
				t.Errorf("s2.Details[%q] = %v, want %v", scheduleName, s2.Details[scheduleName], tt.want2)
			}
		})
	}
}
