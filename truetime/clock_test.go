package truetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClockDelegatesToSource(t *testing.T) {
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(7 * time.Millisecond)})
	c := New(src)

	got := c.Now()
	require.Equal(t, base, got.Earliest)
	require.Equal(t, base.Add(7*time.Millisecond), got.Latest)
}

func TestClockAfter(t *testing.T) {
	src := NewFakeSource(Interval{Earliest: base.Add(5 * time.Millisecond), Latest: base.Add(10 * time.Millisecond)})
	c := New(src)

	require.True(t, c.After(base), "earliest is past base")
	require.False(t, c.After(base.Add(5*time.Millisecond)), "After is strictly greater, not ≥")
	require.False(t, c.After(base.Add(8*time.Millisecond)), "earliest not past 8ms")
}

func TestClockBefore(t *testing.T) {
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(5 * time.Millisecond)})
	c := New(src)

	require.True(t, c.Before(base.Add(6*time.Millisecond)), "latest precedes 6ms")
	require.False(t, c.Before(base.Add(5*time.Millisecond)), "Before is strictly less, not ≤")
}

func TestFakeSourceSetReplaces(t *testing.T) {
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(time.Millisecond)})
	newInterval := Interval{Earliest: base.Add(time.Second), Latest: base.Add(2 * time.Second)}
	src.Set(newInterval)
	require.Equal(t, newInterval, src.Now())
}

func TestFakeSourceAdvancePreservesWidth(t *testing.T) {
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(7 * time.Millisecond)})
	src.Advance(100 * time.Millisecond)

	got := src.Now()
	require.Equal(t, base.Add(100*time.Millisecond), got.Earliest)
	require.Equal(t, base.Add(107*time.Millisecond), got.Latest)
	require.Equal(t, 7*time.Millisecond, got.Width(), "Advance keeps width")
}
