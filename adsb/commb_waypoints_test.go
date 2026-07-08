// Copyright 2026 Collin Kreklow
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
// BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package adsb_test

import (
	"errors"
	"math"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
)

// BDS 5,4 waypoint 1: identity "00CDN" (the three-character CDN zero-padded),
// estimated time of arrival 30 minutes, estimated flight level 350, time to go
// 15 minutes. Vector built from the MB bit ranges of ICAO Doc 9871 Table
// A-2-84.
func TestWaypoint(t *testing.T) {
	w, err := mustVelMsg(t, "A0000000E180621D008D00000000").Waypoint1()
	if err != nil {
		t.Fatalf("Waypoint1: %v", err)
	}

	if !w.Valid {
		t.Fatal("Valid = false, want true")
	}

	if w.Identity != "00CDN" {
		t.Errorf("Identity = %q, want %q", w.Identity, "00CDN")
	}

	if math.Abs(w.EstimatedTimeOfArrival-30) > 0.001 {
		t.Errorf("EstimatedTimeOfArrival = %g, want 30", w.EstimatedTimeOfArrival)
	}

	if w.EstimatedFlightLevel != 350 {
		t.Errorf("EstimatedFlightLevel = %d, want 350", w.EstimatedFlightLevel)
	}

	if math.Abs(w.TimeToGo-15) > 0.001 {
		t.Errorf("TimeToGo = %g, want 15", w.TimeToGo)
	}
}

// The all-ones estimated-time and time-to-go sentinel indicates the waypoint
// is one hour or more away, reported as 60 minutes with the flag set. The
// reserved bit is set here too, to prove it is ignored.
func TestWaypointOneHourSentinel(t *testing.T) {
	w, err := mustVelMsg(t, "A0000000E180621DFF8FFE000000").Waypoint1()
	if err != nil {
		t.Fatalf("Waypoint1: %v", err)
	}

	if !w.EstimatedTimeOfArrivalOneHourOrMore {
		t.Error("EstimatedTimeOfArrivalOneHourOrMore = false, want true")
	}

	if !w.TimeToGoOneHourOrMore {
		t.Error("TimeToGoOneHourOrMore = false, want true")
	}

	if math.Abs(w.EstimatedTimeOfArrival-60) > 0.001 {
		t.Errorf("EstimatedTimeOfArrival = %g, want 60", w.EstimatedTimeOfArrival)
	}

	if math.Abs(w.TimeToGo-60) > 0.001 {
		t.Errorf("TimeToGo = %g, want 60", w.TimeToGo)
	}
}

// The reserved bit (MB 56) is ignored: setting it does not affect the decode.
func TestWaypointReservedBitIgnored(t *testing.T) {
	w, err := mustVelMsg(t, "A0000000E180621D008D01000000").Waypoint1()
	if err != nil {
		t.Fatalf("Waypoint1: %v", err)
	}

	if w.Identity != "00CDN" || w.EstimatedFlightLevel != 350 {
		t.Errorf("reserved bit affected decode: %+v", w)
	}

	if math.Abs(w.EstimatedTimeOfArrival-30) > 0.001 || math.Abs(w.TimeToGo-15) > 0.001 {
		t.Errorf("reserved bit affected times: %+v", w)
	}
}

// Waypoints 2 and 3 (BDS 5,5 and 5,6) share the format and decoder of
// waypoint 1.
func TestWaypointShared(t *testing.T) {
	msg := mustVelMsg(t, "A0000000E180621D008D00000000")

	for _, tc := range []struct {
		name string
		call func() (*adsb.Waypoint, error)
	}{
		{"Waypoint2", msg.Waypoint2},
		{"Waypoint3", msg.Waypoint3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := tc.call()
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if w.Identity != "00CDN" {
				t.Errorf("Identity = %q, want %q", w.Identity, "00CDN")
			}
		})
	}
}

// With the single status bit clear (here with residual payload bits set), the
// waypoint report is not valid.
func TestWaypointStatusClear(t *testing.T) {
	w, err := mustVelMsg(t, "A00000007FFFFFFFFFFFFF000000").Waypoint1()
	if err != nil {
		t.Fatalf("Waypoint1: %v", err)
	}

	if w.Valid {
		t.Error("Valid = true, want false")
	}

	// The full invalid-report contract: every field is zero-valued.
	if w.Identity != "" || w.EstimatedTimeOfArrival != 0 || w.EstimatedFlightLevel != 0 ||
		w.TimeToGo != 0 || w.EstimatedTimeOfArrivalOneHourOrMore || w.TimeToGoOneHourOrMore {
		t.Errorf("invalid report is not zero-valued: %+v", w)
	}
}

// The waypoint registers require a DF20/21 reply.
func TestWaypointRejectNonReply(t *testing.T) {
	msg := mustVelMsg(t, "8D485020994409940838175B284F")

	_, err := msg.Waypoint1()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("Waypoint1 err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.Waypoint2()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("Waypoint2 err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.Waypoint3()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("Waypoint3 err = %v, want ErrNotAvailable", err)
	}
}
