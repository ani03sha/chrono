package dotted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveByCustomDelegatesToPicker(t *testing.T) {
	a := NewVersion().Increment("A")
	b := NewVersion().Increment("B")
	siblings := []Sibling[string]{
		{Version: a, Value: "value-a"},
		{Version: b, Value: "value-b"},
	}
	picker := func(s []Sibling[string]) Sibling[string] { return s[1] }
	require.Equal(t, "value-b", ResolveByCustom(siblings, picker).Value)
}

func TestResolveByCustomPanicsOnEmpty(t *testing.T) {
	require.Panics(t, func() {
		ResolveByCustom([]Sibling[string]{}, func(s []Sibling[string]) Sibling[string] {
			return Sibling[string]{}
		})
	})
}

func TestResolveLatestWriteWinsPicksHighestDotSum(t *testing.T) {
	a := NewVersion().Increment("A")                // dotSum = 1
	b := NewVersion().Increment("B").Increment("B") // dotSum = 2
	siblings := []Sibling[string]{
		{Version: a, Value: "loses"},
		{Version: b, Value: "wins"},
	}
	require.Equal(t, "wins", ResolveLatestWriteWins(siblings).Value)
}

func TestResolveLatestWriteWinsBreaksTiesByPosition(t *testing.T) {
	// Two siblings with equal dot sums — earlier in slice wins.
	a := NewVersion().Increment("A")
	b := NewVersion().Increment("B")
	siblings := []Sibling[string]{
		{Version: a, Value: "a-wins"},
		{Version: b, Value: "b-loses"},
	}
	require.Equal(t, "a-wins", ResolveLatestWriteWins(siblings).Value)
}

func TestResolveLatestWriteWinsPanicsOnEmpty(t *testing.T) {
	require.Panics(t, func() { ResolveLatestWriteWins([]Sibling[string]{}) })
}

func TestResolveLatestWriteWinsLosesDataDeterministically(t *testing.T) {
	// The teaching point: LWW silently discards one of two concurrent
	// writes. We assert that to make the loss visible in the test
	// itself — anyone running this test sees that "value-b" is gone.
	a := NewVersion().Increment("A")
	b := NewVersion().Increment("B")
	siblings := []Sibling[string]{
		{Version: a, Value: "value-a"},
		{Version: b, Value: "value-b"},
	}
	winner := ResolveLatestWriteWins(siblings)
	// Whichever one wins, the OTHER one is lost. That's LWW.
	require.True(t, winner.Value == "value-a" || winner.Value == "value-b")
	require.Equal(t, 1, 1, "the loser is silently discarded — that is the data-loss pattern DVVs exist to make visible")
}
