package truetime

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// This is the default assumption about local-clock drift between NTP queries. 200 parts-per-million is generous:
// quartz oscillators in commodity hardware are typically 50-200ppm. Choosing a generous value means the interval
// widens slightly faster than necessary but stays trustworthy.
const driftPPM = 200

// Seconds between the NTP epoch (1900-01-01) and the Unix epoch (1970-01-01).
const ntpEpochOffset = 2208988800

// NTPSource queries an NTP server periodically and returns intervals based on the reported offset, root dispersion,
// and elapsed time since the last successful query.
//
// Construction calls Query once synchronously so that the source is usable immediately. Use Start to begin background refresh.
type NTPSource struct {
	server        string
	queryInterval time.Duration

	mu             sync.RWMutex
	lastOffset     time.Duration
	lastDispersion time.Duration
	lastQuery      time.Time

	cancel context.CancelFunc
	done   chan struct{}
}

// NewNTPSource constructs an NTPSource and performs one query synchronously so the first Now() call returns
// a meaningful interval.
//
// Returns an error if the initial query fails (network problem, server unreachable, etc.).
// The caller can either retry or fall back to a FakeSource for offline development.
func NewNTPSource(server string, queryInterval time.Duration) (*NTPSource, error) {
	s := &NTPSource{server: server, queryInterval: queryInterval}
	if err := s.query(); err != nil {
		return nil, fmt.Errorf("truetime: initial NTP query to %s: %w", server, err)
	}
	return s, nil
}

// This method performs a single SNTP request and updates the source's state.
//
// Protocol summary: send a 48-byte client packet (mode 3); the server replies with a 48-byte packet containing
// T2 (server receive time) and T3 (server transmit time).
//
// With T1 (client send) and T4 (client receive), offset = ((T2-T1) + (T3-T4)) / 2 and round-trip delay =
// (T4-T1) - (T3-T2). See RFC 4330 for SNTPv4 details.
func (s *NTPSource) query() error {
	conn, err := net.Dial("udp", net.JoinHostPort(s.server, "123"))
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	req := make([]byte, 48)
	req[0] = 0x1B // LI=0, VN=3, Mode=3 (client)

	t1 := time.Now()
	if _, err := conn.Write(req); err != nil {
		return err
	}

	resp := make([]byte, 48)
	if _, err := conn.Read(resp); err != nil {
		return err
	}
	t4 := time.Now()

	t2 := ntpTimeFromBytes(resp[32:40])
	t3 := ntpTimeFromBytes(resp[40:48])

	offset := (t2.Sub(t1) + t3.Sub(t4)) / 2
	delay := t4.Sub(t1) - t3.Sub(t2)

	// Root dispersion is a 32-bit "NTP short" at bytes 8..12 — 16.16
	// fixed-point seconds.
	rootDispRaw := binary.BigEndian.Uint32(resp[8:12])
	rootDisp := time.Duration(rootDispRaw) * time.Second / (1 << 16)

	s.mu.Lock()
	s.lastOffset = offset
	s.lastDispersion = rootDisp + delay/2
	s.lastQuery = t4
	s.mu.Unlock()
	return nil
}

// Now returns the current interval. The estimate is the wall clock
// adjusted by the last measured offset; the uncertainty is the last
// reported dispersion plus drift accumulated since the last query.
func (s *NTPSource) Now() Interval {
	s.mu.RLock()
	offset := s.lastOffset
	disp := s.lastDispersion
	last := s.lastQuery
	s.mu.RUnlock()

	now := time.Now()
	elapsed := now.Sub(last)
	if elapsed < 0 {
		elapsed = 0
	}
	extra := time.Duration(int64(elapsed) * driftPPM / 1_000_000)

	mid := now.Add(offset)
	un := disp + extra
	return Interval{Earliest: mid.Add(-un), Latest: mid.Add(un)}
}

// Start begins a background refresh loop. Cancel ctx (or call Stop) to
// terminate it.
func (s *NTPSource) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		t := time.NewTicker(s.queryInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = s.query()
			}
		}
	}()
}

// Stop terminates the background refresh loop. It is safe to call Stop when Start was never invoked.
func (s *NTPSource) Stop() {
	if s.cancel != nil {
		s.cancel()
		<-s.done
	}
}

// LastDispersion returns the most recent dispersion estimate. Useful for telemetry — a growing dispersion is a warning sign.
func (s *NTPSource) LastDispersion() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastDispersion
}

// ErrShortNTPResponse is returned when an NTP response is malformed.
var ErrShortNTPResponse = errors.New("truetime: short NTP response")

// Parses an 8-byte NTP timestamp (seconds since 1900 in the high 32 bits, fractional seconds in the low 32 bits) into a
// Go time.Time on the Unix epoch.
func ntpTimeFromBytes(b []byte) time.Time {
	raw := binary.BigEndian.Uint64(b)
	sec := raw >> 32
	frac := raw & 0xffffffff
	nsec := (frac * 1_000_000_000) >> 32
	return time.Unix(int64(sec)-ntpEpochOffset, int64(nsec))
}
