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
	"bytes"
	"errors"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/adsbtype"
)

// Aircraft status vectors were constructed per DO-260B Figure 2-13 (subtype 1,
// emergency/priority + Mode A) and Figure 2-14 (subtype 2, TCAS RA broadcast),
// and cross-checked against pyModeS v3.3.0.
func TestAircraftStatus(t *testing.T) {
	t.Run("Emergency", testAircraftStatusEmergency)
	t.Run("ModeADisabled", testAircraftStatusModeADisabled)
	t.Run("ACASRA", testAircraftStatusACASRA)
	t.Run("ACASThreatPosition", testAircraftStatusACASThreatPosition)
	t.Run("RejectNonStatus", testAircraftStatusRejectType)
	t.Run("RejectReservedSubtype", testAircraftStatusRejectSubtype)
}

func testAircraftStatusEmergency(t *testing.T) {
	as, err := mustVelMsg(t, "8D40621DE1AAA200000000000000").AircraftStatus()
	if err != nil {
		t.Fatalf("AircraftStatus: %v", err)
	}

	assertEq(t, "Subtype", as.Subtype, 1)

	if as.ACASRA != nil {
		t.Error("ACASRA: expected nil for emergency message")
	}

	e := as.Emergency
	if e == nil {
		t.Fatal("Emergency: expected non-nil")
	}

	if e.State != adsbtype.EPS5 {
		t.Errorf("State = %v, want Unlawful interference", e.State)
	}

	if !bytes.Equal(e.Squawk, []byte{7, 5, 0, 0}) {
		t.Errorf("Squawk = %v, want [7 5 0 0]", e.Squawk)
	}
}

// A Mode A code of 3000 disables the broadcast; Squawk must be nil.
func testAircraftStatusModeADisabled(t *testing.T) {
	as, err := mustVelMsg(t, "8D40621DE1028000000000000000").AircraftStatus()
	if err != nil {
		t.Fatalf("AircraftStatus: %v", err)
	}

	if as.Emergency == nil {
		t.Fatal("Emergency: expected non-nil")
	}

	if as.Emergency.Squawk != nil {
		t.Errorf("Squawk = %v, want nil (broadcast disabled)", as.Emergency.Squawk)
	}
}

func testAircraftStatusACASRA(t *testing.T) {
	as, err := mustVelMsg(t, "8D40621DE2800165018874000000").AircraftStatus()
	if err != nil {
		t.Fatalf("AircraftStatus: %v", err)
	}

	assertEq(t, "Subtype", as.Subtype, 2)

	if as.Emergency != nil {
		t.Error("Emergency: expected nil for ACAS RA message")
	}

	ra := as.ACASRA
	if ra == nil {
		t.Fatal("ACASRA: expected non-nil")
	}

	assertEq(t, "ARA", ra.ARA, 0x2000)
	assertEq(t, "RAC", ra.RAC, 0x5)
	assertTrue(t, "RATerminated", ra.RATerminated)
	assertFalse(t, "MultipleThreat", ra.MultipleThreat)
	assertEq(t, "ThreatTypeIndicator", ra.ThreatTypeIndicator, 1)

	if ra.ThreatICAO != 0x40621D {
		t.Errorf("ThreatICAO = %06X, want 40621D", ra.ThreatICAO)
	}

	// ARA = 0x2000: bit 41 (single threat) set, all other ARA flags clear.
	assertTrue(t, "SingleThreat", ra.SingleThreat)
	assertFalse(t, "Corrective", ra.Corrective)
	assertFalse(t, "Positive", ra.Positive)

	// RAC = 0x5 (0101): do-not-pass-above and do-not-turn-right.
	assertFalse(t, "DoNotPassBelow", ra.DoNotPassBelow)
	assertTrue(t, "DoNotPassAbove", ra.DoNotPassAbove)
	assertFalse(t, "DoNotTurnLeft", ra.DoNotTurnLeft)
	assertTrue(t, "DoNotTurnRight", ra.DoNotTurnRight)
}

// A subtype 2 message with threat type indicator 2 carries the threat's
// altitude, range and bearing instead of its Mode-S address.
func testAircraftStatusACASThreatPosition(t *testing.T) {
	as, err := mustVelMsg(t, "8D40621DE280000A824CD0000000").AircraftStatus()
	if err != nil {
		t.Fatalf("AircraftStatus: %v", err)
	}

	ra := as.ACASRA
	if ra == nil {
		t.Fatal("ACASRA: expected non-nil")
	}

	assertEq(t, "ThreatTypeIndicator", ra.ThreatTypeIndicator, 2)

	if ra.ThreatAltitude == nil || *ra.ThreatAltitude != 31050 {
		t.Errorf("ThreatAltitude = %v, want 31050", ra.ThreatAltitude)
	}

	wantFloat(t, "ThreatRange", ra.ThreatRange, 5.0, 0.001)
	wantFloat(t, "ThreatBearing", ra.ThreatBearing, 93, 0.001)
}

func testAircraftStatusRejectType(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").AircraftStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

func testAircraftStatusRejectSubtype(t *testing.T) {
	// Subtype 0 is reserved.
	_, err := mustVelMsg(t, "8D40621DE0000000000000000000").AircraftStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
