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
	"testing"

	"kreklow.us/go/go-adsb/adsb"
)

// Target State and Status vectors were constructed per DO-260B Figure 2-10
// and Tables 2-46/2-47/2-48/2-49/2-52, and cross-checked against pyModeS
// v3.3.0 (selected altitude, barometric pressure, selected heading and the
// mode flags).
func TestTargetState(t *testing.T) {
	t.Run("Populated", testTargetStatePopulated)
	t.Run("ModeInvalid", testTargetStateModeInvalid)
	t.Run("NoData", testTargetStateNoData)
	t.Run("RejectNonTargetState", testTargetStateRejectType)
	t.Run("RejectReservedSubtype", testTargetStateRejectSubtype)
}

func testTargetStatePopulated(t *testing.T) {
	ts, err := mustVelMsg(t, "8D40621DEB3E985D015F4C000000").TargetState()
	if err != nil {
		t.Fatalf("TargetState: %v", err)
	}

	assertEq(t, "SILSupplement", ts.SILSupplement, 1)
	assertFalse(t, "SelectedAltitudeFMS", ts.SelectedAltitudeFMS)
	wantInt(t, "SelectedAltitude", ts.SelectedAltitude, 32000)
	wantFloat(t, "BarometricPressureSetting", ts.BarometricPressureSetting, 1012.8, 0.01)
	wantFloat(t, "SelectedHeading", ts.SelectedHeading, 90.0, 0.001)
	assertEq(t, "NACp", ts.NACp, 10)
	assertTrue(t, "NICBaro", ts.NICBaro)
	assertEq(t, "SIL", ts.SIL, 3)
	assertTrue(t, "ModeBitsValid", ts.ModeBitsValid)
	assertTrue(t, "AutopilotEngaged", ts.AutopilotEngaged)
	assertFalse(t, "VNAVEngaged", ts.VNAVEngaged)
	assertTrue(t, "AltitudeHold", ts.AltitudeHold)
	assertFalse(t, "ApproachMode", ts.ApproachMode)
	assertTrue(t, "TCASOperational", ts.TCASOperational)
	assertTrue(t, "LNAVEngaged", ts.LNAVEngaged)
}

// The MCP/FCU mode flags are decoded from their raw bits regardless of the
// mode-bits status; ModeBitsValid reports whether they are meaningful. Here
// the autopilot bit is set while the status bit is clear.
func testTargetStateModeInvalid(t *testing.T) {
	ts, err := mustVelMsg(t, "8D40621DEA000000000100000000").TargetState()
	if err != nil {
		t.Fatalf("TargetState: %v", err)
	}

	if ts.ModeBitsValid {
		t.Error("ModeBitsValid = true, want false")
	}

	if !ts.AutopilotEngaged {
		t.Error("AutopilotEngaged = false, want true (raw bit set)")
	}
}

func testTargetStateNoData(t *testing.T) {
	ts, err := mustVelMsg(t, "8D40621DEA000000000000000000").TargetState()
	if err != nil {
		t.Fatalf("TargetState: %v", err)
	}

	wantNil(t, "SelectedAltitude", ts.SelectedAltitude == nil)
	wantNil(t, "BarometricPressureSetting", ts.BarometricPressureSetting == nil)
	wantNil(t, "SelectedHeading", ts.SelectedHeading == nil)
}

func testTargetStateRejectType(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").TargetState()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// Subtype 0 is the reserved DO-260A trajectory-change format and must be
// rejected.
func testTargetStateRejectSubtype(t *testing.T) {
	_, err := mustVelMsg(t, "8D40621DE8000000000000000000").TargetState()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
