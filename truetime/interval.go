package truetime

import "time"

// Interval is a [Earliest, Latest] range guaranteed (by whatever source produced it) to contain the
// current true global time. The width: Latest - Earliest is the uncertainty: a narrow interval means
// the source is confident; a wide one means it is not.
type Interval struct {
	Earliest time.Time
	Latest   time.Time
}

// Returns the latest - earliest. The wider the interval, the more uncertain the underlying source.
func (i Interval) Width() time.Duration {
	return i.Latest.Sub(i.Earliest)
}

// Reports whether t lies within the interval (inclusive on both ends). "Could t be the current time?":
// yes if Contains is true, no otherwise.
func (i Interval) Contains(t time.Time) bool {
	return !t.Before(i.Earliest) && !t.After(i.Latest)
}

// Reports whether i ends strictly before other begins. If true, every instant in i is provably earlier than
// every instant in other.
//
// This is the predicate Spanner uses to compare two TrueTime intervals for safe causal ordering.
func (i Interval) Before(other Interval) bool {
	return i.Latest.Before(other.Earliest)
}

// Definitely reports whether the interval is entirely at or past t. In other words: Earliest >= t.
// If true, we are certain time has moved past t.
//
// This is the predicate the commit-wait loop polls on: "wait until clock.Now().Definitely(commitAt)."
func (i Interval) Definitely(t time.Time) bool {
	return !i.Earliest.Before(t)
}
