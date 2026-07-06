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

// Comm-B vectors were constructed from explicit subfield values using the
// bit ranges of ICAO Annex 10 Vol IV Table 3-10 and the LSB scaling / sign
// handling of ICAO Doc 9871 as implemented by dump1090/readsb.

// BDS 5,0 track and turn report: roll +20.04 deg, true track 270 deg, ground
// speed 420 kt, track angle rate +2 deg/s, true airspeed 430 kt.
func TestTrackAndTurn(t *testing.T) {
	tt, err := mustVelMsg(t, "A00000008E5C0134A204D7000000").TrackAndTurn()
	if err != nil {
		t.Fatalf("TrackAndTurn: %v", err)
	}

	wantFloat(t, "RollAngle", tt.RollAngle, 20.039, 0.001)
	wantFloat(t, "TrueTrack", tt.TrueTrack, 270, 0.001)
	wantFloat(t, "GroundSpeed", tt.GroundSpeed, 420, 0.001)
	wantFloat(t, "TrackAngleRate", tt.TrackAngleRate, 2.0, 0.001)
	wantFloat(t, "TrueAirspeed", tt.TrueAirspeed, 430, 0.001)
}

// BDS 5,0 with negative-signed fields: roll -20.04 deg (sign set), track
// angle rate -2 deg/s (sign set), locking in the roll and track-rate sign
// offsets.
func TestTrackAndTurnNegative(t *testing.T) {
	tt, err := mustVelMsg(t, "A0000000F1D40125BE0496000000").TrackAndTurn()
	if err != nil {
		t.Fatalf("TrackAndTurn: %v", err)
	}

	wantFloat(t, "RollAngle", tt.RollAngle, -20.039, 0.001)
	wantFloat(t, "TrueTrack", tt.TrueTrack, 90, 0.001)
	wantFloat(t, "TrackAngleRate", tt.TrackAngleRate, -2.0, 0.001)
}

// BDS 6,0 with a negative barometric altitude rate (-2048 ft/min, sign set),
// locking in the vertical-rate sign offset.
func TestHeadingAndSpeedNegative(t *testing.T) {
	hs, err := mustVelMsg(t, "A00000009009911F7E0000000000").HeadingAndSpeed()
	if err != nil {
		t.Fatalf("HeadingAndSpeed: %v", err)
	}

	wantFloat(t, "MagneticHeading", hs.MagneticHeading, 45, 0.001)
	wantFloat(t, "Mach", hs.Mach, 0.5, 0.001)
	wantInt(t, "BarometricAltitudeRate", hs.BarometricAltitudeRate, -2048)
	wantNil(t, "InertialVerticalVelocity", hs.InertialVerticalVelocity == nil)
}

// BDS 6,0 heading and speed report: magnetic heading 270 deg, IAS 250 kt,
// Mach 0.8, barometric altitude rate +1024 ft/min, inertial vertical
// velocity -1024 ft/min.
func TestHeadingAndSpeed(t *testing.T) {
	hs, err := mustVelMsg(t, "A0000000E009F5322107E0000000").HeadingAndSpeed()
	if err != nil {
		t.Fatalf("HeadingAndSpeed: %v", err)
	}

	wantFloat(t, "MagneticHeading", hs.MagneticHeading, 270, 0.001)
	wantFloat(t, "IndicatedAirspeed", hs.IndicatedAirspeed, 250, 0.001)
	wantFloat(t, "Mach", hs.Mach, 0.8, 0.001)
	wantInt(t, "BarometricAltitudeRate", hs.BarometricAltitudeRate, 1024)
	wantInt(t, "InertialVerticalVelocity", hs.InertialVerticalVelocity, -1024)
}

// BDS 4,0 selected vertical intention: MCP and FMS selected altitude 32000 ft,
// barometric pressure setting 1013.0 mb, VNAV engaged, target altitude source
// present.
func TestSelectedVerticalIntention(t *testing.T) {
	svi, err := mustVelMsg(t, "A0000000BE85F430A40185000000").SelectedVerticalIntention()
	if err != nil {
		t.Fatalf("SelectedVerticalIntention: %v", err)
	}

	wantInt(t, "MCPSelectedAltitude", svi.MCPSelectedAltitude, 32000)
	wantInt(t, "FMSSelectedAltitude", svi.FMSSelectedAltitude, 32000)
	wantFloat(t, "BarometricPressureSetting", svi.BarometricPressureSetting, 1013.0, 0.01)

	if !svi.ModeBitsValid {
		t.Error("ModeBitsValid = false, want true")
	}

	if !svi.VNAVMode {
		t.Error("VNAVMode = false, want true")
	}

	if svi.AltitudeHold || svi.ApproachMode {
		t.Error("AltitudeHold/ApproachMode = true, want false")
	}

	if !svi.TargetAltitudeSourceValid {
		t.Error("TargetAltitudeSourceValid = false, want true")
	}

	if svi.TargetAltitudeSource != 1 {
		t.Errorf("TargetAltitudeSource = %d, want 1", svi.TargetAltitudeSource)
	}
}

// With every status bit clear (an all-zero MB field), each optional field
// decodes to nil.
func TestCommBNoData(t *testing.T) {
	msg := mustVelMsg(t, "A000000000000000000000000000")

	tt, err := msg.TrackAndTurn()
	if err != nil {
		t.Fatalf("TrackAndTurn: %v", err)
	}

	wantNil(t, "RollAngle", tt.RollAngle == nil)
	wantNil(t, "TrueTrack", tt.TrueTrack == nil)
	wantNil(t, "GroundSpeed", tt.GroundSpeed == nil)
	wantNil(t, "TrackAngleRate", tt.TrackAngleRate == nil)
	wantNil(t, "TrueAirspeed", tt.TrueAirspeed == nil)

	hs, err := msg.HeadingAndSpeed()
	if err != nil {
		t.Fatalf("HeadingAndSpeed: %v", err)
	}

	wantNil(t, "MagneticHeading", hs.MagneticHeading == nil)
	wantNil(t, "IndicatedAirspeed", hs.IndicatedAirspeed == nil)
	wantNil(t, "Mach", hs.Mach == nil)
	wantNil(t, "BarometricAltitudeRate", hs.BarometricAltitudeRate == nil)
	wantNil(t, "InertialVerticalVelocity", hs.InertialVerticalVelocity == nil)

	svi, err := msg.SelectedVerticalIntention()
	if err != nil {
		t.Fatalf("SelectedVerticalIntention: %v", err)
	}

	wantNil(t, "MCPSelectedAltitude", svi.MCPSelectedAltitude == nil)
	wantNil(t, "FMSSelectedAltitude", svi.FMSSelectedAltitude == nil)
	wantNil(t, "BarometricPressureSetting", svi.BarometricPressureSetting == nil)

	if svi.ModeBitsValid || svi.TargetAltitudeSourceValid {
		t.Error("expected mode/target-source status false for empty MB")
	}
}

// The Comm-B registers require a DF20/21 reply; an extended squitter (DF17)
// has no MB field.
func TestCommBRejectNonReply(t *testing.T) {
	msg := mustVelMsg(t, "8D485020994409940838175B284F")

	_, err := msg.TrackAndTurn()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("TrackAndTurn err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.HeadingAndSpeed()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("HeadingAndSpeed err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.SelectedVerticalIntention()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("SelectedVerticalIntention err = %v, want ErrNotAvailable", err)
	}
}
