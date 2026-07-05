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
	"encoding/hex"
	"errors"
	"math"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/adsbtype"
)

// All synthetic vectors below were constructed from explicit BDS 0,9
// subfield values and cross-checked against pyModeS v3.3.0. The canonical
// subtype-1 vector is the worked example from Sun, "The 1090MHz Riddle".

// mustVelMsg builds a Message from a hex string, failing on any error.
func mustVelMsg(t *testing.T, s string) *adsb.Message {
	t.Helper()

	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex.DecodeString(%q): %v", s, err)
	}

	m := new(adsb.Message)

	err = m.UnmarshalBinary(b)
	if err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	return m
}

// wantFloat asserts a *float64 is non-nil and within tol of want.
func wantFloat(t *testing.T, name string, got *float64, want, tol float64) {
	t.Helper()

	if got == nil {
		t.Errorf("%s = nil, want ~%g", name, want)

		return
	}

	if math.Abs(*got-want) > tol {
		t.Errorf("%s = %g, want ~%g", name, *got, want)
	}
}

// wantInt asserts a *int is non-nil and equal to want.
func wantInt(t *testing.T, name string, got *int, want int) {
	t.Helper()

	if got == nil {
		t.Errorf("%s = nil, want %d", name, want)

		return
	}

	if *got != want {
		t.Errorf("%s = %d, want %d", name, *got, want)
	}
}

// wantNil asserts a pointer field is nil (field carried "no data").
func wantNil(t *testing.T, name string, isNil bool) {
	t.Helper()

	if !isNil {
		t.Errorf("%s: expected nil (no data)", name)
	}
}

func TestVelocity(t *testing.T) {
	t.Run("GroundSpeed", testVelGroundSpeed)
	t.Run("SupersonicGroundSpeed", testVelSupersonicGS)
	t.Run("AirspeedIAS", testVelAirspeedIAS)
	t.Run("SupersonicAirspeedTAS", testVelSupersonicTAS)
	t.Run("NoData", testVelNoData)
	t.Run("WestSouthDown", testVelWestSouthDown)
	t.Run("HeadingUnavailable", testVelHeadingUnavail)
	t.Run("AirspeedUnavailable", testVelAirspeedUnavail)
	t.Run("DF18", testVelDF18)
	t.Run("RejectNonVelocity", testVelRejectNonVelocity)
	t.Run("RejectShortFrame", testVelRejectShortFrame)
	t.Run("RejectReservedSubtype0", testVelRejectSubtype0)
	t.Run("RejectReservedSubtype5", testVelRejectSubtype5)
}

// Canonical subtype-1 ground-speed vector: GS 159.2 kt, track 182.88 deg
// (true north), vertical rate -832 ft/min from a GNSS source, GNSS-baro
// difference +550 ft, NACv 0. Intent change false, IFR capable true.
func testVelGroundSpeed(t *testing.T) {
	v, err := mustVelMsg(t, "8D485020994409940838175B284F").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	if v.Subtype != 1 {
		t.Errorf("Subtype = %d, want 1", v.Subtype)
	}

	wantFloat(t, "GroundSpeed", v.GroundSpeed, 159.20, 0.05)
	wantFloat(t, "Track", v.Track, 182.88, 0.01)
	wantNil(t, "Airspeed", v.Airspeed == nil)
	wantNil(t, "Heading", v.Heading == nil)
	wantInt(t, "VerticalRate", v.VerticalRate, -832)

	if v.VerticalRateSource != adsbtype.VRS0 {
		t.Errorf("VerticalRateSource = %v, want GNSS (VRS0)", v.VerticalRateSource)
	}

	wantInt(t, "GNSSBaroDiff", v.GNSSBaroDiff, 550)

	if v.NACv != 0 {
		t.Errorf("NACv = %d, want 0", v.NACv)
	}

	if v.IntentChange {
		t.Error("IntentChange = true, want false")
	}

	if !v.IFRCapability {
		t.Error("IFRCapability = false, want true")
	}
}

// Subtype 2 (supersonic ground speed): east 400 kt, north 300 kt -> GS 500,
// track 53.13 deg; vertical rate +6400 ft/min barometric; GNSS-baro +2500 ft;
// NACv 1. Exercises the x4 velocity scaling.
func testVelSupersonicGS(t *testing.T) {
	v, err := mustVelMsg(t, "8D4840409A486509919465000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	if v.Subtype != 2 {
		t.Errorf("Subtype = %d, want 2", v.Subtype)
	}

	wantFloat(t, "GroundSpeed", v.GroundSpeed, 500, 0.05)
	wantFloat(t, "Track", v.Track, 53.13, 0.01)
	wantInt(t, "VerticalRate", v.VerticalRate, 6400)

	if v.VerticalRateSource != adsbtype.VRS1 {
		t.Errorf("VerticalRateSource = %v, want Barometric (VRS1)", v.VerticalRateSource)
	}

	wantInt(t, "GNSSBaroDiff", v.GNSSBaroDiff, 2500)

	if v.NACv != 1 {
		t.Errorf("NACv = %d, want 1", v.NACv)
	}
}

// Subtype 3 (airspeed): IAS 250 kt, magnetic heading 90 deg, vertical rate
// -1024 ft/min GNSS, GNSS-baro -1000 ft, NACv 2.
func testVelAirspeedIAS(t *testing.T) {
	v, err := mustVelMsg(t, "8D4840409B95001F6844A9000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	if v.Subtype != 3 {
		t.Errorf("Subtype = %d, want 3", v.Subtype)
	}

	wantNil(t, "GroundSpeed", v.GroundSpeed == nil)
	wantFloat(t, "Airspeed", v.Airspeed, 250, 0.05)

	if v.AirspeedType != adsbtype.AST0 {
		t.Errorf("AirspeedType = %v, want IAS (AST0)", v.AirspeedType)
	}

	wantFloat(t, "Heading", v.Heading, 90, 0.2)
	wantInt(t, "VerticalRate", v.VerticalRate, -1024)
	wantInt(t, "GNSSBaroDiff", v.GNSSBaroDiff, -1000)

	if v.NACv != 2 {
		t.Errorf("NACv = %d, want 2", v.NACv)
	}
}

// Subtype 4 (supersonic airspeed): TAS 600 kt, magnetic heading 270 deg,
// vertical rate and GNSS-baro both "no data". Exercises x4 airspeed scaling
// and the TAS airspeed type.
func testVelSupersonicTAS(t *testing.T) {
	v, err := mustVelMsg(t, "8D4840409C470092F00000000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	if v.Subtype != 4 {
		t.Errorf("Subtype = %d, want 4", v.Subtype)
	}

	wantFloat(t, "Airspeed", v.Airspeed, 600, 0.05)

	if v.AirspeedType != adsbtype.AST1 {
		t.Errorf("AirspeedType = %v, want TAS (AST1)", v.AirspeedType)
	}

	wantFloat(t, "Heading", v.Heading, 270, 0.2)
	wantNil(t, "VerticalRate", v.VerticalRate == nil)
	wantNil(t, "GNSSBaroDiff", v.GNSSBaroDiff == nil)
}

// All velocity, vertical-rate and GNSS-baro fields carry the all-zero
// "no data" sentinel, but the header fields are populated (subtype 1,
// IFR capable, NACv 5). The header must decode while every optional pointer
// stays nil.
func testVelNoData(t *testing.T) {
	v, err := mustVelMsg(t, "8D48404099680000000000000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	if v.Subtype != 1 {
		t.Errorf("Subtype = %d, want 1", v.Subtype)
	}

	if !v.IFRCapability {
		t.Error("IFRCapability = false, want true")
	}

	if v.NACv != 5 {
		t.Errorf("NACv = %d, want 5", v.NACv)
	}

	wantNil(t, "GroundSpeed", v.GroundSpeed == nil)
	wantNil(t, "Track", v.Track == nil)
	wantNil(t, "Airspeed", v.Airspeed == nil)
	wantNil(t, "Heading", v.Heading == nil)
	wantNil(t, "VerticalRate", v.VerticalRate == nil)
	wantNil(t, "GNSSBaroDiff", v.GNSSBaroDiff == nil)
}

// Subtype 1 with west/south direction bits, a downward vertical rate, and a
// GNSS-below-baro difference: west 100 kt, south 200 kt -> GS 223.6,
// track 206.57 deg; VR -1600 ft/min baro; GNSS-baro -500 ft; NACv 3.
func testVelWestSouthDown(t *testing.T) {
	v, err := mustVelMsg(t, "8D484040995C6599386895000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	wantFloat(t, "GroundSpeed", v.GroundSpeed, 223.61, 0.05)
	wantFloat(t, "Track", v.Track, 206.57, 0.01)
	wantInt(t, "VerticalRate", v.VerticalRate, -1600)

	if v.VerticalRateSource != adsbtype.VRS1 {
		t.Errorf("VerticalRateSource = %v, want Barometric (VRS1)", v.VerticalRateSource)
	}

	wantInt(t, "GNSSBaroDiff", v.GNSSBaroDiff, -500)

	if v.NACv != 3 {
		t.Errorf("NACv = %d, want 3", v.NACv)
	}
}

// Subtype 3 with the heading-status bit clear (heading unavailable) but a
// valid airspeed, and a GNSS-baro magnitude of all-ones (127) which also
// means "no data".
func testVelHeadingUnavail(t *testing.T) {
	v, err := mustVelMsg(t, "8D4840409B000016A0007F000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	wantFloat(t, "Airspeed", v.Airspeed, 180, 0.05)
	wantNil(t, "Heading", v.Heading == nil)
	wantNil(t, "GNSSBaroDiff", v.GNSSBaroDiff == nil)
}

// Subtype 3 with a valid magnetic heading (45 deg) but an all-zero airspeed
// field: Heading is set while Airspeed stays nil.
func testVelAirspeedUnavail(t *testing.T) {
	v, err := mustVelMsg(t, "8D4840409B048000101000000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	wantFloat(t, "Heading", v.Heading, 45, 0.2)
	wantNil(t, "Airspeed", v.Airspeed == nil)
}

// A DF18 (TIS-B / non-transponder) extended squitter with CF 0 carries the
// same BDS 0,9 velocity payload as DF17 and must decode identically:
// east 120 kt, north 160 kt -> GS 200, track 36.87 deg; VR +64 ft/min GNSS;
// GNSS-baro +100 ft; NACv 1.
func testVelDF18(t *testing.T) {
	v, err := mustVelMsg(t, "9048404099487914200805000000").Velocity()
	if err != nil {
		t.Fatalf("Velocity: %v", err)
	}

	if v.Subtype != 1 {
		t.Errorf("Subtype = %d, want 1", v.Subtype)
	}

	wantFloat(t, "GroundSpeed", v.GroundSpeed, 200, 0.05)
	wantFloat(t, "Track", v.Track, 36.87, 0.01)
	wantInt(t, "VerticalRate", v.VerticalRate, 64)

	if v.VerticalRateSource != adsbtype.VRS0 {
		t.Errorf("VerticalRateSource = %v, want GNSS (VRS0)", v.VerticalRateSource)
	}

	wantInt(t, "GNSSBaroDiff", v.GNSSBaroDiff, 100)
}

// A DF17 airborne-position message (ES type 11) is not a velocity message.
func testVelRejectNonVelocity(t *testing.T) {
	_, err := mustVelMsg(t, "8D40621D58C382D690C8AC2863A7").Velocity()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// A short-frame surveillance message (DF4) has no extended squitter and must
// be rejected.
func testVelRejectShortFrame(t *testing.T) {
	_, err := mustVelMsg(t, "20001910bc45e9").Velocity()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// Type code 19 with reserved subtype 0 is not a defined velocity format and
// must be rejected.
func testVelRejectSubtype0(t *testing.T) {
	_, err := mustVelMsg(t, "8D48404098000100200401000000").Velocity()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// Type code 19 with reserved subtype 5 is not a defined velocity format and
// must be rejected.
func testVelRejectSubtype5(t *testing.T) {
	_, err := mustVelMsg(t, "8D4840409D000100200401000000").Velocity()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
