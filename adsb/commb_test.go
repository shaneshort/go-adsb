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
	"slices"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/adsbtype"
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

// A true track with the sign bit set and a zero magnitude sits exactly on the
// 180-degree wrap boundary.
func TestTrackAndTurnWrap(t *testing.T) {
	tt, err := mustVelMsg(t, "A00000008E580134A204D7000000").TrackAndTurn()
	if err != nil {
		t.Fatalf("TrackAndTurn: %v", err)
	}

	wantFloat(t, "TrueTrack", tt.TrueTrack, 180, 0.001)
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

	_, err = msg.DataLinkCapability()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("DataLinkCapability err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.CommonUsageGICB()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("CommonUsageGICB err = %v, want ErrNotAvailable", err)
	}
}

// BDS 1,7 common usage GICB capability report: registers 0,5, 0,9, 2,0, 4,0,
// 5,0, 6,0 and F,1 marked available. Vector built from the MB bit assignments
// of ICAO Doc 9871 Table A-2-23.
func TestCommonUsageGICB(t *testing.T) {
	got, err := mustVelMsg(t, "A00000008A810108000000000000").CommonUsageGICB()
	if err != nil {
		t.Fatalf("CommonUsageGICB: %v", err)
	}

	want := []adsbtype.BDS{
		adsbtype.BDS05, adsbtype.BDS09, adsbtype.BDS20,
		adsbtype.BDS40, adsbtype.BDS50, adsbtype.BDS60, adsbtype.BDSF1,
	}
	if !slices.Equal(got, want) {
		t.Errorf("CommonUsageGICB = %v, want %v", got, want)
	}
}

// BDS 1,7 bits 27 and 28 map to the E,1 and E,2 built-in test equipment
// registers (ICAO Doc 9871 Table A-2-23).
func TestCommonUsageGICBBITE(t *testing.T) {
	got, err := mustVelMsg(t, "A000000000000030000000000000").CommonUsageGICB()
	if err != nil {
		t.Fatalf("CommonUsageGICB: %v", err)
	}

	want := []adsbtype.BDS{adsbtype.BDSE1, adsbtype.BDSE2}
	if !slices.Equal(got, want) {
		t.Errorf("CommonUsageGICB = %v, want %v", got, want)
	}
}

// BDS 1,0 data link capability report: overlay command and ACAS capable,
// Mode S subnetwork version 4, enhanced protocol and specific services,
// uplink ELM 3, downlink ELM 5, aircraft ID and squitter capable, no SI code,
// common-usage GICB present, ACAS additional capability 10, DTE sub-address
// status 0xACE1. Vector built from the MB bit ranges of ICAO Annex 10 Vol IV
// Table 3-6.
func TestDataLinkCapability(t *testing.T) {
	dlc, err := mustVelMsg(t, "A0000000100309B5DAACE1000000").DataLinkCapability()
	if err != nil {
		t.Fatalf("DataLinkCapability: %v", err)
	}

	wantBool(t, "ContinuationFlag", dlc.ContinuationFlag, false)
	wantBool(t, "OverlayCommandCapability", dlc.OverlayCommandCapability, true)
	wantBool(t, "ACASCapability", dlc.ACASCapability, true)
	wantU8(t, "ModeSSubnetworkVersion", dlc.ModeSSubnetworkVersion, 4)
	wantBool(t, "TransponderEnhancedProtocol", dlc.TransponderEnhancedProtocol, true)
	wantBool(t, "SpecificServicesCapability", dlc.SpecificServicesCapability, true)
	wantU8(t, "UplinkELMCapability", dlc.UplinkELMCapability, 3)
	wantU8(t, "DownlinkELMCapability", dlc.DownlinkELMCapability, 5)
	wantBool(t, "AircraftIdentificationCapable", dlc.AircraftIdentificationCapable, true)
	wantBool(t, "SquitterCapability", dlc.SquitterCapability, true)
	wantBool(t, "SurveillanceIdentifierCode", dlc.SurveillanceIdentifierCode, false)
	wantBool(t, "CommonUsageGICBCapability", dlc.CommonUsageGICBCapability, true)
	wantU8(t, "ACASAdditionalCapability", dlc.ACASAdditionalCapability, 10)

	if dlc.DTESubaddressStatus != 0xACE1 {
		t.Errorf("DTESubaddressStatus = %#04x, want 0xACE1", dlc.DTESubaddressStatus)
	}
}

// wantBool asserts a bool field equals want.
func wantBool(t *testing.T, name string, got, want bool) {
	t.Helper()

	if got != want {
		t.Errorf("%s = %t, want %t", name, got, want)
	}
}

// wantU8 asserts a uint8 field equals want.
func wantU8(t *testing.T, name string, got, want uint8) {
	t.Helper()

	if got != want {
		t.Errorf("%s = %d, want %d", name, got, want)
	}
}
