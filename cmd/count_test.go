package cmd

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/abicky/pd-shift/internal/pd"
	"github.com/abicky/pd-shift/testing/mock"
	"go.uber.org/mock/gomock"
)

func Test_runCount(t *testing.T) {
	tests := []struct {
		name           string
		tz             *time.Location
		since          string
		until          string
		handoffTimes   []string
		include        []string
		nonWorkingDays []string
		scheduleIDs    []string
		wantOutput     string
	}{
		{
			name:           "example",
			tz:             time.UTC,
			since:          "2025-07-01",
			until:          "2025-07-08",
			handoffTimes:   []string{"05:00", "17:00"},
			include:        []string{"working-days:17:00-05:00", "non-working-days"},
			nonWorkingDays: []string{"JP holidays", "Sat", "Sun", "Dec 29", "Dec 30", "Dec 31", "Jan 1", "Jan 2", "Jan 3"},
			scheduleIDs:    []string{"P4DRALL"},
			wantOutput: `# Summary

- John Smith: 6.75
- Takeshi Arabiki: 1.50
- Total: 8.25
- Expected total: 9

# Details

- Tue, 2025-07-01 17:00+0000 - Wed, 2025-07-02 05:00+0000
    - Weekly Rotation
        - John Smith: 1.00 (17:00 - 05:00)
- Wed, 2025-07-02 17:00+0000 - Thu, 2025-07-03 05:00+0000
    - Weekly Rotation
        - John Smith: 1.00 (17:00 - 05:00)
- Thu, 2025-07-03 17:00+0000 - Fri, 2025-07-04 05:00+0000
    - Weekly Rotation
        - John Smith: 1.00 (17:00 - 05:00)
- Fri, 2025-07-04 17:00+0000 - Sat, 2025-07-05 05:00+0000
    - Weekly Rotation
        - John Smith: 0.58 (17:00 - 09:00)
        - Takeshi Arabiki: 0.42 (09:00 - 05:00)
- Sat, 2025-07-05 05:00+0000 - Sat, 2025-07-05 17:00+0000
    - Weekly Rotation
        - Takeshi Arabiki: 0.08 (05:00 - 15:00)
        - John Smith: 0.92 (15:00 - 17:00)
- Sat, 2025-07-05 17:00+0000 - Sun, 2025-07-06 05:00+0000
    - Weekly Rotation
        - John Smith: 1.00 (17:00 - 05:00)
- Sun, 2025-07-06 05:00+0000 - Sun, 2025-07-06 17:00+0000
    - Weekly Rotation
        - John Smith: 1.00 (05:00 - 17:00)
- Sun, 2025-07-06 17:00+0000 - Mon, 2025-07-07 05:00+0000
    - Weekly Rotation
        - John Smith: 0.25 (17:00 - 05:00)
        - Takeshi Arabiki: 0.75 (05:00 - 05:00)
- Mon, 2025-07-07 17:00+0000 - Tue, 2025-07-08 05:00+0000
    - Weekly Rotation
        - Takeshi Arabiki: 0.25 (17:00 - 05:00)

# PagerDuty schedules

## Weekly Rotation

- 2025-07-01T05:00:00+09:00 - 2025-07-05T09:00:00+09:00: John Smith
- 2025-07-05T09:00:00+09:00 - 2025-07-05T15:00:00+09:00: Takeshi Arabiki
- 2025-07-05T15:00:00+09:00 - 2025-07-07T05:00:00+09:00: John Smith
- 2025-07-07T05:00:00+09:00 - 2025-07-08T05:00:00+09:00: Takeshi Arabiki
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := mock.NewMockClient(ctrl)
			for _, id := range tt.scheduleIDs {
				client.EXPECT().GetScheduleWithContext(t.Context(), id, pagerduty.GetScheduleOptions{
					TimeZone: tt.tz.String(),
					Since:    tt.since + " " + tt.handoffTimes[0],
					Until:    tt.until + " " + tt.handoffTimes[0],
				}).DoAndReturn(func(_ context.Context, id string, o pagerduty.GetScheduleOptions) (*pagerduty.Schedule, error) {
					return &pagerduty.Schedule{
						Name: "Weekly Rotation",
						FinalSchedule: pagerduty.ScheduleLayer{
							RenderedScheduleEntries: []pagerduty.RenderedScheduleEntry{
								{
									Start: "2025-07-01T05:00:00+09:00",
									End:   "2025-07-05T09:00:00+09:00",
									User: pagerduty.APIObject{
										Summary: "John Smith",
									},
								},
								{
									Start: "2025-07-05T09:00:00+09:00",
									End:   "2025-07-05T15:00:00+09:00",
									User: pagerduty.APIObject{
										Summary: "Takeshi Arabiki",
									},
								},
								{
									Start: "2025-07-05T15:00:00+09:00",
									End:   "2025-07-07T05:00:00+09:00",
									User: pagerduty.APIObject{
										Summary: "John Smith",
									},
								},
								{
									Start: "2025-07-07T05:00:00+09:00",
									End:   "2025-07-08T05:00:00+09:00",
									User: pagerduty.APIObject{
										Summary: "Takeshi Arabiki",
									},
								},
							},
						},
					}, nil
				})
			}

			var b bytes.Buffer

			sg, err := pd.NewShiftGenerator(tt.tz, tt.since, tt.until, tt.handoffTimes, tt.include, tt.nonWorkingDays)
			if err != nil {
				t.Fatal(err)
			}

			if err := runCount(t.Context(), &b, client, tt.tz, tt.since+" "+tt.handoffTimes[0], tt.until+" "+tt.handoffTimes[0], tt.scheduleIDs, sg); err != nil {
				t.Errorf("runCount() = %v, want nil", err)
			}
			if b.String() != tt.wantOutput {
				t.Errorf("b.String() = %v, want %v", b.String(), tt.wantOutput)
			}
		})
	}
}
