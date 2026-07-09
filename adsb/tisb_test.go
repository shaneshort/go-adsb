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

// BDS-independent TIS-B coarse airborne position (DF18, CF 3): ICAO address,
// no surveillance condition, service volume 5, pressure altitude 38000 ft,
// ground track 90 deg, ground speed range starting at 272 kt, and a 12-bit
// even-format coarse CPR position. Vector built from the ME bit ranges of
// DO-260B Figure 2-30.
func TestTISBCoarsePosition(t *testing.T) {
	p, err := mustVelMsg(t, "9340621D0B8714143E87D0000000").TISBCoarsePosition()
	if err != nil {
		t.Fatalf("TISBCoarsePosition: %v", err)
	}

	wantBool(t, "IMF", p.IMF, false)

	if p.SurveillanceStatus != adsbtype.SSS0 {
		t.Errorf("SurveillanceStatus = %v, want %v", p.SurveillanceStatus, adsbtype.SSS0)
	}

	wantU8(t, "ServiceVolumeID", p.ServiceVolumeID, 5)

	if p.PressureAltitude == nil || *p.PressureAltitude != 38000 {
		t.Errorf("PressureAltitude = %v, want 38000", p.PressureAltitude)
	}

	wantFloat(t, "GroundTrack", p.GroundTrack, 90, 0.001)
	wantFloat(t, "GroundSpeed", p.GroundSpeed, 272, 0.001)

	if p.CPR == nil {
		t.Fatal("CPR = nil")
	}

	if p.CPR.Nb != 12 || p.CPR.F != 0 || p.CPR.Lat != 1000 || p.CPR.Lon != 2000 {
		t.Errorf("CPR = %+v, want {Nb:12 F:0 Lat:1000 Lon:2000}", p.CPR)
	}
}

// A TIS-B coarse position with a Mode A / track-number address, an invalid
// altitude, an invalid ground track and no ground speed information decodes
// with IMF set and the optional fields nil.
func TestTISBCoarsePositionInvalid(t *testing.T) {
	p, err := mustVelMsg(t, "9340621DE0000001000000000000").TISBCoarsePosition()
	if err != nil {
		t.Fatalf("TISBCoarsePosition: %v", err)
	}

	wantBool(t, "IMF", p.IMF, true)

	if p.SurveillanceStatus != adsbtype.SSS3 {
		t.Errorf("SurveillanceStatus = %v, want %v", p.SurveillanceStatus, adsbtype.SSS3)
	}

	wantNil(t, "PressureAltitude", p.PressureAltitude == nil)
	wantNil(t, "GroundTrack", p.GroundTrack == nil)
	wantNil(t, "GroundSpeed", p.GroundSpeed == nil)
}

// A clear ground-track status bit yields a nil ground track even when the track
// angle bits carry residual data.
func TestTISBCoarsePositionTrackResidue(t *testing.T) {
	p, err := mustVelMsg(t, "9340621D00000F80000000000000").TISBCoarsePosition()
	if err != nil {
		t.Fatalf("TISBCoarsePosition: %v", err)
	}

	wantNil(t, "GroundTrack", p.GroundTrack == nil)
}

// The coarse ground speed is a stepped range (DO-260B Table 2-107): code 1 is
// the lower bound 0 ("less than 16 knots") and code 63 is 1968 knots.
func TestTISBCoarseGroundSpeed(t *testing.T) {
	for _, c := range []struct {
		name string
		hex  string
		want float64
	}{
		{"Code1", "9340621D00000002000000000000", 0},
		{"Code63", "9340621D0000007E000000000000", 1968},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, err := mustVelMsg(t, c.hex).TISBCoarsePosition()
			if err != nil {
				t.Fatalf("TISBCoarsePosition: %v", err)
			}

			wantFloat(t, "GroundSpeed", p.GroundSpeed, c.want, 0.001)
		})
	}
}

// A 12-bit coarse CPR decodes identically to a 17-bit CPR whose encoded values
// represent the same fraction (shifted left by five bits), verifying that the
// position decoder scales by 2^Nb in both the local and the global (even/odd
// pair) paths.
func TestCPRCoarseScaling(t *testing.T) {
	ref := []float64{52.0, 4.0}

	coarse := &adsb.CPR{Nb: 12, F: 0, Lat: 2000, Lon: 3000}
	fine := &adsb.CPR{Nb: 17, F: 0, Lat: 2000 << 5, Lon: 3000 << 5}

	cc, err := coarse.DecodeLocal(ref)
	if err != nil {
		t.Fatalf("coarse DecodeLocal: %v", err)
	}

	ff, err := fine.DecodeLocal(ref)
	if err != nil {
		t.Fatalf("fine DecodeLocal: %v", err)
	}

	if cc[0] != ff[0] || cc[1] != ff[1] {
		t.Errorf("local: coarse %v != fine %v", cc, ff)
	}

	// The global (even/odd) path must scale by 2^Nb too.
	gotCoarse, err := adsb.DecodeGlobalPosition(
		&adsb.CPR{Nb: 12, F: 0, Lat: 2906, Lon: 1605},
		&adsb.CPR{Nb: 12, F: 1, Lat: 2317, Lon: 1568},
	)
	if err != nil {
		t.Fatalf("coarse global: %v", err)
	}

	gotFine, err := adsb.DecodeGlobalPosition(
		&adsb.CPR{Nb: 17, F: 0, Lat: 2906 << 5, Lon: 1605 << 5},
		&adsb.CPR{Nb: 17, F: 1, Lat: 2317 << 5, Lon: 1568 << 5},
	)
	if err != nil {
		t.Fatalf("fine global: %v", err)
	}

	if gotCoarse[0] != gotFine[0] || gotCoarse[1] != gotFine[1] {
		t.Errorf("global: coarse %v != fine %v", gotCoarse, gotFine)
	}
}

// An unset (zero) Nb must be treated as the default 17-bit encoding when
// pairing CPRs, so a legacy Nb=0 value pairs with a normally-decoded Nb=17
// value in both the airborne and surface global paths.
func TestCPRDefaultNbPairing(t *testing.T) {
	// Airborne: Nb=0 paired with Nb=17 decodes as if both were 17-bit.
	odd := &adsb.CPR{Nb: 17, F: 1, Lat: 74158, Lon: 50194}

	got, err := adsb.DecodeGlobalPosition(&adsb.CPR{Nb: 0, F: 0, Lat: 93000, Lon: 51372}, odd)
	if err != nil {
		t.Fatalf("airborne Nb=0/17: %v", err)
	}

	want, err := adsb.DecodeGlobalPosition(&adsb.CPR{Nb: 17, F: 0, Lat: 93000, Lon: 51372}, odd)
	if err != nil {
		t.Fatalf("airborne Nb=17/17: %v", err)
	}

	if got[0] != want[0] || got[1] != want[1] {
		t.Errorf("airborne Nb=0 %v != Nb=17 %v", got, want)
	}

	// Surface: the same equivalence through the reference decode path.
	ref := []float64{-52, 4}
	sOdd := &adsb.CPR{Nb: 17, Surface: true, F: 1, Lat: 39195, Lon: 110320}

	sGot, err := adsb.DecodeGlobalPositionRef(
		&adsb.CPR{Nb: 0, Surface: true, F: 0, Lat: 115609, Lon: 116941}, sOdd, ref)
	if err != nil {
		t.Fatalf("surface Nb=0/17: %v", err)
	}

	sWant, err := adsb.DecodeGlobalPositionRef(
		&adsb.CPR{Nb: 17, Surface: true, F: 0, Lat: 115609, Lon: 116941}, sOdd, ref)
	if err != nil {
		t.Fatalf("surface Nb=17/17: %v", err)
	}

	if sGot[0] != sWant[0] || sGot[1] != sWant[1] {
		t.Errorf("surface Nb=0 %v != Nb=17 %v", sGot, sWant)
	}
}

// IMF reports the ICAO/Mode A flag of a TIS-B or ADS-R message. Its ME bit
// position is type-code specific (DO-260B §2.2.17 and §2.2.18): airborne
// position bit 8, airborne velocity bit 9, surface position bit 21, coarse
// position bit 1.
func TestIMF(t *testing.T) {
	for _, c := range []struct {
		name string
		hex  string
		want bool
	}{
		{"FineAirbornePosition", "9240621D59000000000000000000", true}, // CF2 TC11 bit8
		{"ADSRAirborneVelocity", "9640621D98800000000000000000", true}, // CF6 TC19 bit9
		{"FineSurfacePosition", "9240621D28000800000000000000", true},  // CF2 TC5 bit21
		{"CoarseSet", "9340621D80000000000000000000", true},            // CF3 bit1
		{"CoarseClear", "9340621D0B8714143E87D0000000", false},         // CF3 bit1 clear
	} {
		t.Run(c.name, func(t *testing.T) {
			imf, err := mustVelMsg(t, c.hex).IMF()
			if err != nil {
				t.Fatalf("IMF: %v", err)
			}

			wantBool(t, "IMF", imf, c.want)
		})
	}
}

// IMF is not available for ADS-B messages, message types without a defined IMF
// subfield, or non-DF18 formats.
func TestIMFReject(t *testing.T) {
	for _, c := range []struct {
		name string
		hex  string
	}{
		{"ADSB", "9040621D59000000000000000000"},            // CF0 = ADS-B, no IMF
		{"IdentNoIMFField", "9240621D08000000000000000000"}, // CF2 TC1 has no IMF
		{"DF17", "8D485020994409940838175B284F"},            // not a DF18 message
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := mustVelMsg(t, c.hex).IMF()
			if !errors.Is(err, adsb.ErrNotAvailable) {
				t.Errorf("err = %v, want ErrNotAvailable", err)
			}
		})
	}
}

// The TIS-B coarse position is only available from a DF18 reply with control
// field 3 (coarse format TIS-B); other formats return ErrNotAvailable.
func TestTISBCoarsePositionRejectOther(t *testing.T) {
	// DF18 with CF 0 (a normal ADS-B message from a non-transponder device).
	_, err := mustVelMsg(t, "9040621D0B8714143E87D0000000").TISBCoarsePosition()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("CF 0 err = %v, want ErrNotAvailable", err)
	}

	// A DF17 extended squitter has no control field.
	_, err = mustVelMsg(t, "8D485020994409940838175B284F").TISBCoarsePosition()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("DF17 err = %v, want ErrNotAvailable", err)
	}
}
