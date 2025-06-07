package pd

import (
	"time"
)

type Shift struct {
	Start   time.Time
	End     time.Time
	Details map[string][]ShiftDetail

	duration time.Duration
}

type ShiftDetail struct {
	User       string
	Start      time.Time
	End        time.Time
	Proportion float64
}

func NewShift(start, end time.Time) *Shift {
	return &Shift{
		Start:    start,
		End:      end,
		Details:  make(map[string][]ShiftDetail),
		duration: end.Sub(start),
	}
}

func (s *Shift) AddDetails(iter *ScheduleEntryIter) {
	entry := iter.Peek()
	for entry != nil {
		//                  s.Start     s.End
		//                  |-----------|
		// 1.                              entry.Start entry.End
		//                                 |-----------|
		// 2. entry.Start entry.End
		//    |-----------|
		// 3. entry.Start          entry.End
		//    |--------------------|
		// 4. entry.Start                              entry.End
		//    |----------------------------------------|
		if !s.End.After(entry.Start) { // 1
			break
		} else if !entry.End.After(s.Start) { // 2
			entry = iter.Next()
		} else {
			s.addDetail(iter.scheduleName, entry)
			if !entry.End.After(s.End) { // 3
				entry = iter.Next()
			} else { // 4
				break
			}
		}
	}
}

func (s *Shift) addDetail(name string, e *ScheduleEntry) {
	// s.Start          s.End
	// |----------------|
	//      e.Start        e.End
	//      |--------------|
	// -> start = e.Start
	start := s.Start
	if e.Start.After(start) {
		start = e.Start
	}

	//      s.Start          s.End
	//      |----------------|
	// e.Start        e.End
	// |--------------|
	// -> end = e.End
	end := s.End
	if e.End.Before(end) {
		end = e.End
	}

	s.Details[name] = append(s.Details[name], ShiftDetail{
		User:       e.User,
		Start:      start,
		End:        end,
		Proportion: float64(end.Sub(start)) / float64(s.duration),
	})
}
