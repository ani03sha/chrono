package proofs

import (
	"sort"
	"testing"
	"time"

	"github.com/ani03sha/chrono/hlc"
	"github.com/ani03sha/chrono/internal/wallclock"
	"github.com/stretchr/testify/require"
)

// TestNTPStepCausesIncorrectOrderingWithWallClock shows the motivating failure for hybrid logical clocks:
// a system that timestamps events with raw wall time produces *incorrect* causal order whenever the wall
// clock is stepped backward by NTP.
//
// The test then re-runs the same scenario with HLC and demonstrates that the order is preserved.
func TestNTPStepCausesIncorrectOrderingWithWallClock(t *testing.T) {
	type wallEvent struct {
		Order int
		Wall  time.Time
	}

	fake := wallclock.NewFake(time.Unix(1000, 0))
	var events []wallEvent

	// Event 1: wall time 1000.
	events = append(events, wallEvent{Order: 1, Wall: fake.Now()})

	// Event 2: 100ms later, wall time 1000.100.
	fake.Advance(100 * time.Millisecond)
	events = append(events, wallEvent{Order: 2, Wall: fake.Now()})

	// NTP step backward by 200ms — wall clock now reads 999.900,
	// EARLIER than the previous event's recorded time.
	fake.Reverse(200 * time.Millisecond)

	// Event 3 happens AFTER event 2 in real time, but its wall stamp
	// is now smaller than both prior events'.
	events = append(events, wallEvent{Order: 3, Wall: fake.Now()})

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Wall.Before(events[j].Wall)
	})

	// Naive wall-time sort places Event 3 first — this is the failure.
	require.Equal(t, 3, events[0].Order,
		"wall-time sort puts the most-recent event first because of the backward step")
	require.Equal(t, 1, events[1].Order)
	require.Equal(t, 2, events[2].Order)

	// Now repeat with HLC. HLC absorbs the backward step into its
	// logical counter, so the total order matches real time.
	type hlcEvent struct {
		Order int
		Stamp hlc.Timestamp
	}

	hlcFake := wallclock.NewFake(time.Unix(1000, 0))
	clock := hlc.New(hlcFake)
	var hlcEvents []hlcEvent

	hlcEvents = append(hlcEvents, hlcEvent{1, clock.Now()})
	hlcFake.Advance(100 * time.Millisecond)
	hlcEvents = append(hlcEvents, hlcEvent{2, clock.Now()})
	hlcFake.Reverse(200 * time.Millisecond)
	hlcEvents = append(hlcEvents, hlcEvent{3, clock.Now()})

	sort.SliceStable(hlcEvents, func(i, j int) bool {
		return hlcEvents[i].Stamp.Less(hlcEvents[j].Stamp)
	})

	// HLC sort produces real-time order [1, 2, 3].
	require.Equal(t, 1, hlcEvents[0].Order,
		"HLC absorbs the backward step into its logical counter")
	require.Equal(t, 2, hlcEvents[1].Order)
	require.Equal(t, 3, hlcEvents[2].Order)

	// The third HLC stamp shares its WallTime with the second (since
	// the wall clock was behind), but its Logical is higher.
	require.Equal(t, hlcEvents[1].Stamp.WallTime, hlcEvents[2].Stamp.WallTime,
		"after backward step, l stays put — only c advances")
	require.Greater(t, hlcEvents[2].Stamp.Logical, hlcEvents[1].Stamp.Logical)
}
