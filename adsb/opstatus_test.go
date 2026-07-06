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
	"kreklow.us/go/go-adsb/adsbtype"
)

// Operational status vectors were constructed from explicit subfield values
// per DO-260B Figure 2-11 and Tables 2-59/2-60/2-61/2-68A/2-68B. Version-stable
// fields were cross-checked against pyModeS v3.3.0; the version-dependent
// capability/operational-mode subfields (which pyModeS does not decode) were
// cross-checked against readsb.

func TestOperationalStatus(t *testing.T) {
	t.Run("V2Airborne", testOpStatusV2Airborne)
	t.Run("V1Airborne", testOpStatusV1Airborne)
	t.Run("V0Airborne", testOpStatusV0Airborne)
	t.Run("V2Surface", testOpStatusV2Surface)
	t.Run("V1Surface", testOpStatusV1Surface)
	t.Run("OperationalModeFormat", testOpStatusOMFormat)
	t.Run("RejectNonOpStatus", testOpStatusReject)
	t.Run("RejectReservedSubtype", testOpStatusReservedSubtype)
	t.Run("RejectUnsupportedVersion", testOpStatusUnsupportedVersion)
}

func testOpStatusV2Airborne(t *testing.T) {
	os, err := mustVelMsg(t, "8D40621DF8322026005A7A000000").OperationalStatus()
	if err != nil {
		t.Fatalf("OperationalStatus: %v", err)
	}

	assertEq(t, "Subtype", os.Subtype, 0)
	assertEq(t, "Version", os.Version, 2)
	assertTrue(t, "NICSupplementA", os.NICSupplementA)
	assertEq(t, "NACp", os.NACp, 10)
	assertEq(t, "SIL", os.SIL, 3)
	assertEq(t, "SILSupplement", os.SILSupplement, 1)

	if os.HRD != adsbtype.HRD0 {
		t.Errorf("HRD = %v, want True north", os.HRD)
	}

	if os.Surface != nil {
		t.Error("Surface: expected nil for airborne message")
	}

	a := os.Airborne
	if a == nil {
		t.Fatal("Airborne: expected non-nil")
	}

	assertTrue(t, "ACASOperational", a.ACASOperational)
	assertTrue(t, "Has1090ESIn", a.Has1090ESIn)
	assertTrue(t, "HasUATIn", a.HasUATIn)
	assertTrue(t, "ARVCapable", a.ARVCapable)
	assertFalse(t, "TSCapable", a.TSCapable)
	assertFalse(t, "CDTI", a.CDTI)
	assertEq(t, "GVA", a.GVA, 1)
	assertTrue(t, "NICBaro", a.NICBaro)
	assertTrue(t, "TCASRAActive", a.TCASRAActive)
	assertFalse(t, "IdentActive", a.IdentActive)
	assertTrue(t, "SingleAntenna", a.SingleAntenna)
	assertEq(t, "SDA", a.SDA, 2)
}

func testOpStatusV1Airborne(t *testing.T) {
	os, err := mustVelMsg(t, "8D40621DF8130020003928000000").OperationalStatus()
	if err != nil {
		t.Fatalf("OperationalStatus: %v", err)
	}

	assertEq(t, "Version", os.Version, 1)
	assertEq(t, "NACp", os.NACp, 9)
	assertEq(t, "SIL", os.SIL, 2)

	a := os.Airborne
	if a == nil {
		t.Fatal("Airborne: expected non-nil")
	}

	// v1 bit 11 is "Not-TCAS" (inverted); 0 means ACAS operational.
	assertTrue(t, "ACASOperational", a.ACASOperational)
	// v1 bit 12 is CDTI, not 1090ES-IN.
	assertTrue(t, "CDTI", a.CDTI)
	assertFalse(t, "Has1090ESIn", a.Has1090ESIn)
	assertTrue(t, "ARVCapable", a.ARVCapable)
	assertTrue(t, "TSCapable", a.TSCapable)
	assertTrue(t, "NICBaro", a.NICBaro)
	// GVA, SDA and SIL supplement are v2-only.
	assertEq(t, "GVA", a.GVA, 0)
	assertEq(t, "SILSupplement", os.SILSupplement, 0)
}

func testOpStatusV0Airborne(t *testing.T) {
	os, err := mustVelMsg(t, "8D40621DF8300000000000000000").OperationalStatus()
	if err != nil {
		t.Fatalf("OperationalStatus: %v", err)
	}

	assertEq(t, "Version", os.Version, 0)

	a := os.Airborne
	if a == nil {
		t.Fatal("Airborne: expected non-nil")
	}

	// v0 bit 11 "Not-TCAS" = 1 means ACAS not operational.
	assertFalse(t, "ACASOperational", a.ACASOperational)
	assertTrue(t, "CDTI", a.CDTI)
}

func testOpStatusV2Surface(t *testing.T) {
	os, err := mustVelMsg(t, "8D40621DF931792605483E000000").OperationalStatus()
	if err != nil {
		t.Fatalf("OperationalStatus: %v", err)
	}

	assertEq(t, "Subtype", os.Subtype, 1)
	assertEq(t, "Version", os.Version, 2)
	assertEq(t, "NACp", os.NACp, 8)
	assertEq(t, "SIL", os.SIL, 3)

	if os.HRD != adsbtype.HRD1 {
		t.Errorf("HRD = %v, want Magnetic north", os.HRD)
	}

	if os.Airborne != nil {
		t.Error("Airborne: expected nil for surface message")
	}

	s := os.Surface
	if s == nil {
		t.Fatal("Surface: expected non-nil")
	}

	assertTrue(t, "PositionOffsetApplied", s.PositionOffsetApplied)
	assertTrue(t, "Has1090ESIn", s.Has1090ESIn)
	assertFalse(t, "B2Low", s.B2Low)
	assertTrue(t, "HasUATIn", s.HasUATIn)
	assertEq(t, "NACv", s.NACv, 3)
	assertTrue(t, "NICSupplementC", s.NICSupplementC)
	assertEq(t, "LengthWidthCode", s.LengthWidthCode, 9)
	assertTrue(t, "TrackAngleIsHeading", s.TrackAngleIsHeading)
	assertTrue(t, "TCASRAActive", s.TCASRAActive)
	assertTrue(t, "SingleAntenna", s.SingleAntenna)
	assertEq(t, "SDA", s.SDA, 2)
	assertEq(t, "GPSAntennaOffsetLongitudinal", s.GPSAntennaOffsetLongitudinal, 10)
}

func testOpStatusV1Surface(t *testing.T) {
	os, err := mustVelMsg(t, "8D40621DF9320510002720000000").OperationalStatus()
	if err != nil {
		t.Fatalf("OperationalStatus: %v", err)
	}

	assertEq(t, "Version", os.Version, 1)

	s := os.Surface
	if s == nil {
		t.Fatal("Surface: expected non-nil")
	}

	assertTrue(t, "PositionOffsetApplied", s.PositionOffsetApplied)
	// v1 surface bit 12 is CDTI, not 1090ES-IN.
	assertTrue(t, "CDTI", s.CDTI)
	assertFalse(t, "Has1090ESIn", s.Has1090ESIn)
	assertTrue(t, "B2Low", s.B2Low)
	assertEq(t, "LengthWidthCode", s.LengthWidthCode, 5)
	assertTrue(t, "IdentActive", s.IdentActive)
}

// A nonzero operational-mode format code (ME 25-26) selects an undefined OM
// layout, so the operational-mode subfields are left unset rather than decoded
// from the wrong positions.
func testOpStatusOMFormat(t *testing.T) {
	os, err := mustVelMsg(t, "8D40621DF823206600593A000000").OperationalStatus()
	if err != nil {
		t.Fatalf("OperationalStatus: %v", err)
	}

	a := os.Airborne
	if a == nil {
		t.Fatal("Airborne: expected non-nil")
	}

	if a.TCASRAActive || a.IdentActive || a.SingleAntenna || a.SDA != 0 {
		t.Error("operational-mode fields should be unset when the format code is nonzero")
	}
}

func testOpStatusReject(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").OperationalStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// A reserved subtype (2-7) has no defined layout and must be rejected rather
// than returning a struct with neither Airborne nor Surface populated.
func testOpStatusReservedSubtype(t *testing.T) {
	_, err := mustVelMsg(t, "8D40621DFA322026005A7A000000").OperationalStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// An unsupported ADS-B version (3-7) has unknown subfield layouts and must be
// rejected rather than partially decoded.
func testOpStatusUnsupportedVersion(t *testing.T) {
	_, err := mustVelMsg(t, "8D40621DF8322026007A7A000000").OperationalStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// assertEq compares an unsigned field against an expected int value.
func assertEq[T ~uint8 | ~uint16](t *testing.T, name string, got T, want int) {
	t.Helper()

	if int(got) != want {
		t.Errorf("%s = %d, want %d", name, got, want)
	}
}

func assertTrue(t *testing.T, name string, got bool) {
	t.Helper()

	if !got {
		t.Errorf("%s = false, want true", name)
	}
}

func assertFalse(t *testing.T, name string, got bool) {
	t.Helper()

	if got {
		t.Errorf("%s = true, want false", name)
	}
}
