package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/abicky/pd-shift/internal/pd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const dateTimeLayout = "Mon, 2006-01-02 15:04-0700"

var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Count PagerDuty on-call shifts",
	Long: `This command counts PagerDuty on-call shifts based on the specified configuration.
For full configuration details, refer to https://github.com/abicky/pd-shift#configurations`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Prevent showing usage after validation
		cmd.SilenceUsage = true

		v := vipers[cmd]

		tz, err := time.LoadLocation(v.GetString("time-zone"))
		if err != nil {
			return err
		}

		handoffTimes := v.GetStringSlice("handoff-times")
		slices.Sort(handoffTimes)

		since := v.GetString("since")
		until := v.GetString("until")

		sg, err := pd.NewShiftGenerator(tz, since, until, handoffTimes, v.GetStringSlice("include"), v.GetStringSlice("non-working-days"))
		if err != nil {
			return err
		}

		client := pagerduty.NewClient(viper.GetString("api-key"))

		return runCount(
			cmd.Context(),
			os.Stdout,
			client,
			tz,
			// Align time to the first handoff time to include entire shifts
			since+" "+handoffTimes[0],
			until+" "+handoffTimes[0],
			v.GetStringSlice("schedule-ids"),
			sg,
		)
	},
}

func init() {
	rootCmd.AddCommand(countCmd)

	countCmd.Flags().String("time-zone", "UTC", "Time zone used for handoff-times, since, and until")
	countCmd.Flags().StringSlice("schedule-ids", []string{}, "List of scheduled IDs to include in the count")
	countCmd.MarkFlagRequired("schedule-ids")
	countCmd.Flags().StringSlice("handoff-times", []string{}, "List of handoff times")
	countCmd.MarkFlagRequired("handoff-times")
	countCmd.Flags().StringSlice("include", []string{}, "List of shifts to count")
	countCmd.Flags().StringSlice("non-working-days", []string{}, "List of non-working days used by include")
	countCmd.Flags().String("since", "", "Start of the date range for counting on-call shifts")
	countCmd.MarkFlagRequired("since")
	countCmd.Flags().String("until", "", "End of the date range for counting on-call shifts")
	countCmd.MarkFlagRequired("until")
}

func runCount(ctx context.Context, out io.Writer, client pd.Client, tz *time.Location, since, until string, scheduleIDs []string, sg *pd.ShiftGenerator) error {
	rsEntries := make(map[string][]pagerduty.RenderedScheduleEntry, len(scheduleIDs))
	scheduleNames := make([]string, len(scheduleIDs))
	iters := make([]*pd.ScheduleEntryIter, len(scheduleIDs))
	for i, id := range scheduleIDs {
		schedule, err := client.GetScheduleWithContext(ctx, id, pagerduty.GetScheduleOptions{
			TimeZone: tz.String(),
			Since:    since,
			Until:    until,
		})
		if err != nil {
			var pdErr pagerduty.APIError
			if errors.As(err, &pdErr) && pdErr.StatusCode == http.StatusUnauthorized {
				return errors.New("failed to get PagerDuty schedule: unauthorized")
			} else {
				return fmt.Errorf("failed to get PagerDuty schedule: %w", err)
			}
		}

		rsEntries[schedule.Name] = schedule.FinalSchedule.RenderedScheduleEntries
		scheduleNames[i] = schedule.Name
		iters[i], err = pd.NewScheduleEntryIter(schedule.Name, tz, schedule.FinalSchedule.RenderedScheduleEntries)
		if err != nil {
			return err
		}
	}

	shifts := make([]*pd.Shift, 0)
	shiftCounts := make(map[string]float64)
	for shift := range sg.Shifts() {
		for _, iter := range iters {
			shift.AddDetails(iter)
		}
		shifts = append(shifts, shift)
		for _, details := range shift.Details {
			for _, detail := range details {
				shiftCounts[detail.User] += detail.Proportion
			}
		}
	}

	fmt.Fprintf(out, "# Summary\n\n")
	total := 0.0
	users := slices.Collect(maps.Keys(shiftCounts))
	slices.Sort(users)
	for _, user := range users {
		total += shiftCounts[user]
		fmt.Fprintf(out, "- %s: %0.2f\n", user, shiftCounts[user])
	}
	fmt.Fprintf(out, "- Total: %0.2f\n", total)
	fmt.Fprintf(out, "- Expected total: %v\n\n", len(shifts)*len(iters))
	fmt.Fprintf(out, "# Details\n\n")
	for _, shift := range shifts {
		fmt.Fprintf(out, "- %s - %s\n", shift.Start.Format(dateTimeLayout), shift.End.Format(dateTimeLayout))
		for _, name := range scheduleNames {
			fmt.Fprintf(out, "    - %s\n", name)
			for _, detail := range shift.Details[name] {
				fmt.Fprintf(out, "        - %s: %0.2f (%s - %s)\n", detail.User, detail.Proportion, detail.Start.Format("15:04"), detail.End.Format("15:04"))
			}
		}
	}
	fmt.Fprintln(out, "\n# PagerDuty schedules")
	for _, name := range scheduleNames {
		fmt.Fprintf(out, "\n## %s\n\n", name)
		for _, entry := range rsEntries[name] {
			fmt.Fprintf(out, "- %s - %s: %s\n", entry.Start, entry.End, entry.User.Summary)
		}
	}

	return nil
}
