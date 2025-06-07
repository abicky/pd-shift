package pd

import (
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

type ScheduleEntryIter struct {
	scheduleName string
	index        int
	current      *ScheduleEntry
	entries      []*ScheduleEntry
}

type ScheduleEntry struct {
	Start time.Time
	End   time.Time
	User  string
}

func NewScheduleEntryIter(scheduleName string, tz *time.Location, rsEntries []pagerduty.RenderedScheduleEntry) (*ScheduleEntryIter, error) {
	entries := make([]*ScheduleEntry, len(rsEntries))
	for i, entry := range rsEntries {
		s, err := time.ParseInLocation(time.RFC3339, entry.Start, tz)
		if err != nil {
			return nil, err
		}
		e, err := time.ParseInLocation(time.RFC3339, entry.End, tz)
		if err != nil {
			return nil, err
		}
		entries[i] = &ScheduleEntry{
			Start: s,
			End:   e,
			User:  entry.User.Summary,
		}
	}

	iter := &ScheduleEntryIter{
		scheduleName: scheduleName,
		index:        0,
		entries:      entries,
	}
	iter.Next()
	return iter, nil
}

func (s *ScheduleEntryIter) Next() *ScheduleEntry {
	if s.index >= len(s.entries) {
		return nil
	}

	e := s.entries[s.index]
	s.current = e
	s.index++
	return e
}

func (s *ScheduleEntryIter) Peek() *ScheduleEntry {
	return s.current
}
