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

// BDS 2,2 antenna positions: four antennas, each with a type and an X (from
// the nose) and Z (above ground) offset in metres; a zero offset is invalid.
// Vector built from the MB bit ranges of ICAO Doc 9871 Table A-2-34.
func TestAntennaPositions(t *testing.T) {
	ap, err := mustVelMsg(t, "A000000025151856410000000000").AntennaPositions()
	if err != nil {
		t.Fatalf("AntennaPositions: %v", err)
	}

	wantAntenna(t, "Antenna1", ap.Antennas[0], adsbtype.AntennaType1, 10, 5)
	wantAntenna(t, "Antenna2", ap.Antennas[1], adsbtype.AntennaType2, 12, 5)
	wantAntenna(t, "Antenna3", ap.Antennas[2], adsbtype.AntennaType3, 8, 4)

	// The fourth antenna is entirely invalid (type 0, zero offsets).
	if ap.Antennas[3].Type != adsbtype.AntennaType0 {
		t.Errorf("Antenna4.Type = %v, want %v", ap.Antennas[3].Type, adsbtype.AntennaType0)
	}

	wantNil(t, "Antenna4.X", ap.Antennas[3].X == nil)
	wantNil(t, "Antenna4.Z", ap.Antennas[3].Z == nil)
}

// wantAntenna asserts an antenna's type and non-nil X/Z offsets.
func wantAntenna(t *testing.T, name string, a adsb.Antenna, typ adsbtype.AntennaType, x, z int) {
	t.Helper()

	if a.Type != typ {
		t.Errorf("%s.Type = %v, want %v", name, a.Type, typ)
	}

	wantInt(t, name+".X", a.X, x)
	wantInt(t, name+".Z", a.Z, z)
}

// The antenna position report requires a DF20/21 reply.
func TestAntennaPositionsRejectNonReply(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").AntennaPositions()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// BDS 2,5 aircraft type: the ICAO Doc 8643 aircraft description (landplane, two
// engines, jet, medium wake) and the type designator "B738". Vector built from
// the MB bit ranges of ICAO Doc 9871 Table A-2-37.
func TestAircraftType(t *testing.T) {
	at, err := mustVelMsg(t, "A0000000311416F9F001A0000000").AircraftType()
	if err != nil {
		t.Fatalf("AircraftType: %v", err)
	}

	if at.Class != "L" {
		t.Errorf("Class = %q, want %q", at.Class, "L")
	}

	if at.NumberOfEngines != 2 {
		t.Errorf("NumberOfEngines = %d, want 2", at.NumberOfEngines)
	}

	if at.EngineType != "J" {
		t.Errorf("EngineType = %q, want %q", at.EngineType, "J")
	}

	if at.ModelDesignation != "B738" {
		t.Errorf("ModelDesignation = %q, want %q", at.ModelDesignation, "B738")
	}

	if at.WakeTurbulenceCategory != "M" {
		t.Errorf("WakeTurbulenceCategory = %q, want %q", at.WakeTurbulenceCategory, "M")
	}
}

// The model designation is decoded from a DF21 reply, its reserved fifth
// character (here a non-zero 'X') is ignored, and the four-character sentinel
// "2222" reported as not specified is returned as an empty string (ICAO Doc
// 9871 Table A-2-37).
func TestAircraftTypeModelDesignation(t *testing.T) {
	// DF21 reply with a non-zero fifth model character.
	at, err := mustVelMsg(t, "A8000000311416F9F0C1A0000000").AircraftType()
	if err != nil {
		t.Fatalf("AircraftType (DF21): %v", err)
	}

	if at.ModelDesignation != "B738" {
		t.Errorf("ModelDesignation = %q, want %q (reserved 5th char ignored)",
			at.ModelDesignation, "B738")
	}

	// The four-character "2222" sentinel means the designator is not specified.
	at, err = mustVelMsg(t, "A0000000311596596401A0000000").AircraftType()
	if err != nil {
		t.Fatalf("AircraftType (2222): %v", err)
	}

	if at.ModelDesignation != "" {
		t.Errorf("ModelDesignation = %q, want empty (2222 = not specified)", at.ModelDesignation)
	}
}

// The aircraft type report requires a DF20/21 reply.
func TestAircraftTypeRejectNonReply(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").AircraftType()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
