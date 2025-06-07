package pd

import (
	"errors"
	"fmt"
	"iter"
	"regexp"
	"slices"
	"strings"
	"time"

	holidayjp "github.com/holiday-jp/holiday_jp-go"
)

var (
	timeRangeRegexp = regexp.MustCompile(`\A(\d{2}:\d{2})-(\d{2}:\d{2})\z`)
	weekdays        = map[string]time.Weekday{
		"Sunday":    time.Sunday,
		"Sun":       time.Sunday,
		"Monday":    time.Monday,
		"Mon":       time.Monday,
		"Tuesday":   time.Tuesday,
		"Tue":       time.Tuesday,
		"Wednesday": time.Wednesday,
		"Wed":       time.Wednesday,
		"Thursday":  time.Thursday,
		"Thu":       time.Thursday,
		"Friday":    time.Friday,
		"Fri":       time.Friday,
		"Saturday":  time.Saturday,
		"Sat":       time.Saturday,
	}
)

type ShiftGenerator struct {
	since             time.Time
	until             time.Time
	current           time.Time
	index             int
	shiftDurations    []time.Duration
	includeConditions []includeCondition
	nonWorkingDaySet  *nonWorkingDaySet
}

type includeCondition interface {
	match(shift *Shift) bool
}

type workingDaysIncludeCondition struct {
	timeRanges       []*timeRange
	nonWorkingDaySet *nonWorkingDaySet
}

var _ includeCondition = (*workingDaysIncludeCondition)(nil)

type nonWorkingDaysIncludeCondition struct {
	timeRanges       []*timeRange
	nonWorkingDaySet *nonWorkingDaySet
}

var _ includeCondition = (*nonWorkingDaysIncludeCondition)(nil)

type timeRange struct {
	start string
	end   string
}

type nonWorkingDaySet struct {
	nonWorkingDays []nonWorkingDay
}

type nonWorkingDay interface {
	cover(time.Time) bool
}

type jpHoliday struct{}

var _ nonWorkingDay = (*jpHoliday)(nil)

type date struct {
	month time.Month
	day   int
}

var _ nonWorkingDay = date{}

type weekday time.Weekday

var _ nonWorkingDay = weekday(time.Sunday)

func NewShiftGenerator(tz *time.Location, since, until string, handoffTimes, include, nonWorkingDays []string) (*ShiftGenerator, error) {
	if len(handoffTimes) == 0 {
		return nil, errors.New("no handoff times provided")
	}
	if !slices.IsSorted(handoffTimes) {
		return nil, errors.New("handoff times must be sorted")
	}

	nwds, err := newNonWorkingDaySet(nonWorkingDays)
	if err != nil {
		return nil, err
	}

	includeConditions, err := buildIncludeConditions(include, handoffTimes, nwds)
	if err != nil {
		return nil, err
	}

	shiftDurations, err := buildShiftDurations(handoffTimes, tz)
	if err != nil {
		return nil, err
	}

	sinceTime, err := time.ParseInLocation(time.DateOnly, since, tz)
	if err != nil {
		return nil, fmt.Errorf("invalid since value: %w", err)
	}
	untilTime, err := time.ParseInLocation(time.DateOnly, until, tz)
	if err != nil {
		return nil, fmt.Errorf("invalid until value: %w", err)
	}

	current, err := time.ParseInLocation(time.DateTime, since+" "+handoffTimes[0]+":00", tz)
	if err != nil {
		return nil, err
	}

	return &ShiftGenerator{
		since:             sinceTime,
		until:             untilTime,
		current:           current,
		index:             0,
		shiftDurations:    shiftDurations,
		includeConditions: includeConditions,
		nonWorkingDaySet:  nwds,
	}, nil
}

func (s *ShiftGenerator) Shifts() iter.Seq[*Shift] {
	return func(yield func(v *Shift) bool) {
		for s.current.Before(s.until) {
			shift := NewShift(s.current, s.current.Add(s.shiftDurations[s.index]))
			s.current = shift.End
			s.index = (s.index + 1) % len(s.shiftDurations)
			if len(s.includeConditions) > 0 && !slices.ContainsFunc(s.includeConditions, func(c includeCondition) bool {
				return c.match(shift)
			}) {
				continue
			}
			if !yield(shift) {
				return
			}
		}
	}
}

func (c *timeRange) match(shift *Shift) bool {
	return c.start == shift.Start.Format("15:04") && c.end == shift.End.Format("15:04")
}

func (c *workingDaysIncludeCondition) match(shift *Shift) bool {
	if c.nonWorkingDaySet.cover(shift.Start) {
		return false
	}
	return len(c.timeRanges) == 0 || slices.ContainsFunc(c.timeRanges, func(tr *timeRange) bool {
		return tr.match(shift)
	})
}

func (c *nonWorkingDaysIncludeCondition) match(shift *Shift) bool {
	if !c.nonWorkingDaySet.cover(shift.Start) {
		return false
	}
	return len(c.timeRanges) == 0 || slices.ContainsFunc(c.timeRanges, func(tr *timeRange) bool {
		return tr.match(shift)
	})
}

func (hs *nonWorkingDaySet) cover(t time.Time) bool {
	return slices.ContainsFunc(hs.nonWorkingDays, func(h nonWorkingDay) bool {
		return h.cover(t)
	})
}

func (h *jpHoliday) cover(t time.Time) bool {
	return holidayjp.IsHoliday(t)
}

func (d date) cover(t time.Time) bool {
	return t.Month() == d.month && t.Day() == d.day
}

func (w weekday) cover(t time.Time) bool {
	return time.Weekday(w) == t.Weekday()
}

func buildShiftDurations(handoffTimes []string, zone *time.Location) ([]time.Duration, error) {
	times := make([]time.Time, len(handoffTimes), len(handoffTimes)+1)
	for i, t := range handoffTimes {
		var err error
		times[i], err = time.ParseInLocation("15:04", t, zone)
		if err != nil {
			return nil, fmt.Errorf("invalid handoff time: %w", err)
		}
	}
	times = append(times, times[0].Add(24*time.Hour))

	shiftDurations := make([]time.Duration, len(times)-1)
	for i := 1; i <= len(shiftDurations); i++ {
		shiftDurations[i-1] = times[i].Sub(times[i-1])
	}

	return shiftDurations, nil
}

func buildIncludeConditions(include, handoffTimes []string, nwds *nonWorkingDaySet) ([]includeCondition, error) {
	includeConditions := make([]includeCondition, 0)
	for _, c := range include {
		typeAndRange := strings.SplitN(c, ":", 2)
		var timeRanges []*timeRange
		if len(typeAndRange) == 2 {
			if !timeRangeRegexp.MatchString(typeAndRange[1]) {
				return nil, fmt.Errorf("invalid time range format in the include condition %q", c)
			}
			i := slices.IndexFunc(handoffTimes, func(s string) bool {
				return strings.HasPrefix(typeAndRange[1], s)
			})
			if i == -1 {
				return nil, fmt.Errorf("start time in the include condition %q must match one of handoff times", c)
			}

			found := false
			for len(timeRanges) < len(handoffTimes) {
				startTime := handoffTimes[i]
				i = (i + 1) % len(handoffTimes)
				timeRanges = append(timeRanges, &timeRange{start: startTime, end: handoffTimes[i]})
				if strings.HasSuffix(c, typeAndRange[1]) {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("end time in the include condition %q must match one of handoff times", c)
			}
		}

		switch typeAndRange[0] {
		case "working-days":
			includeConditions = append(includeConditions, &workingDaysIncludeCondition{timeRanges: timeRanges, nonWorkingDaySet: nwds})
		case "non-working-days":
			includeConditions = append(includeConditions, &nonWorkingDaysIncludeCondition{timeRanges: timeRanges, nonWorkingDaySet: nwds})
		default:
			return nil, fmt.Errorf("unknown include type %q", typeAndRange[0])
		}
	}

	return includeConditions, nil
}

func newNonWorkingDaySet(nonWorkingDays []string) (*nonWorkingDaySet, error) {
	set := make([]nonWorkingDay, len(nonWorkingDays))
LOOP:
	for i, d := range nonWorkingDays {
		switch d {
		case "JP holidays":
			set[i] = &jpHoliday{}
		default:
			var err error
			if w, ok := weekdays[d]; ok {
				set[i] = weekday(w)
				continue LOOP
			}
			for _, layout := range []string{"Jan 02", "Jan _2", "January 02", "January _2"} {
				t, e := time.Parse(layout, d)
				if e == nil {
					set[i] = date{month: t.Month(), day: t.Day()}
					continue LOOP
				}
				err = errors.Join(err, e)
			}
			return nil, fmt.Errorf("invalid nonWorkingDay: %w", err)
		}
	}
	return &nonWorkingDaySet{nonWorkingDays: set}, nil
}
