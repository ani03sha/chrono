package truetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var base = time.Unix(1_700_000_000, 0)

func TestWidth(t *testing.T) {
	i := Interval{Earliest: base, Latest: base.Add(7 * time.Millisecond)}
	require.Equal(t, 7*time.Millisecond, i.Width())
}

func TestContains(t *testing.T) {
	i := Interval{Earliest: base, Latest: base.Add(10 * time.Millisecond)}
	require.True(t, i.Contains(base))
	require.True(t, i.Contains(base.Add(5*time.Millisecond)))
	require.True(t, i.Contains(base.Add(10*time.Millisecond)), "Latest is inclusive")
	require.False(t, i.Contains(base.Add(-time.Microsecond)))
	require.False(t, i.Contains(base.Add(11*time.Millisecond)))
}

func TestBefore(t *testing.T) {
	a := Interval{Earliest: base, Latest: base.Add(5 * time.Millisecond)}
	b := Interval{Earliest: base.Add(6 * time.Millisecond), Latest: base.Add(10 * time.Millisecond)}
	overlap := Interval{Earliest: base.Add(3 * time.Millisecond), Latest: base.Add(8 * time.Millisecond)}

	require.True(t, a.Before(b))
	require.False(t, b.Before(a))
	require.False(t, a.Before(overlap), "overlapping intervals are not Before")
}

func TestDefinitely(t *testing.T) {
	i := Interval{Earliest: base.Add(5 * time.Millisecond), Latest: base.Add(10 * time.Millisecond)}
	require.True(t, i.Definitely(base), "interval past base")
	require.True(t, i.Definitely(base.Add(5*time.Millisecond)), "Earliest == t counts as past")
	require.False(t, i.Definitely(base.Add(6*time.Millisecond)), "Earliest before t")
}

func TestZeroWidthInterval(t *testing.T) {
	// A zero-width interval is a perfectly-known time. All predicates
	// must still behave sensibly.
	i := Interval{Earliest: base, Latest: base}
	require.Equal(t, time.Duration(0), i.Width())
	require.True(t, i.Contains(base))
	require.True(t, i.Definitely(base))
	require.False(t, i.Definitely(base.Add(time.Nanosecond)))
}
