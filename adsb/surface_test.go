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

// Surface vectors were constructed from explicit BDS 0,6 subfield values and
// cross-checked against pyModeS v3.3.0. The position pair is the Schiphol
// surface example decoded against a (52, 4) reference.

func TestSurfaceMovement(t *testing.T) {
	t.Run("Speed", testSurfaceSpeed)
	t.Run("Track", testSurfaceTrack)
	t.Run("DF18", testSurfaceDF18)
	t.Run("RejectNonSurface", testSurfaceRejectNonSurface)
	t.Run("RejectAirbornePosition", testSurfaceRejectAirborne)
	t.Run("RejectShortFrame", testSurfaceRejectShortFrame)
}

// Movement decodes to ground speed via the non-linear BDS 0,6 table. A raw
// movement of 0 (no data) and 125-127 (reserved) leave GroundSpeed nil; 1 is
// stopped (0 kt); 124 is the >= 175 kt saturation code.
func testSurfaceSpeed(t *testing.T) {
	cases := []struct {
		name string
		hex  string
		want *float64
	}{
		{"NoData", "8C48417538000000000000000000", nil},
		{"Stopped", "8C48417538100000000000000000", f64(0)},
		{"SubKnot", "8C48417538800000000000000000", f64(0.875)},
		{"QuarterStep", "8C48417538A00000000000000000", f64(1.25)},
		{"HalfStep", "8C48417539400000000000000000", f64(5.5)},
		{"FifteenKt", "8C4841753A700000000000000000", f64(15)},
		{"TwoKtStep", "8C4841753E400000000000000000", f64(82)},
		{"FiveKtStep", "8C4841753F300000000000000000", f64(130)},
		{"Saturated", "8C4841753FC00000000000000000", f64(175)},
		{"Reserved", "8C4841753FD00000000000000000", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sm, err := mustVelMsg(t, c.hex).SurfaceMovement()
			if err != nil {
				t.Fatalf("SurfaceMovement: %v", err)
			}

			if c.want == nil {
				wantNil(t, "GroundSpeed", sm.GroundSpeed == nil)
			} else {
				wantFloat(t, "GroundSpeed", sm.GroundSpeed, *c.want, 0.001)
			}
		})
	}
}

// Ground track is reported only when its status bit is set: raw value scaled
// by 360/128 degrees. A clear status bit leaves Track nil.
func testSurfaceTrack(t *testing.T) {
	present, err := mustVelMsg(t, "8C4841753A7A1000000000000000").SurfaceMovement()
	if err != nil {
		t.Fatalf("SurfaceMovement: %v", err)
	}

	wantFloat(t, "Track", present.Track, 92.8125, 0.001)

	absent, err := mustVelMsg(t, "8C4841753A721000000000000000").SurfaceMovement()
	if err != nil {
		t.Fatalf("SurfaceMovement: %v", err)
	}

	wantNil(t, "Track", absent.Track == nil)
}

// A DF18 surface message decodes identically via ESType().
func testSurfaceDF18(t *testing.T) {
	sm, err := mustVelMsg(t, "904841753A7A1000000000000000").SurfaceMovement()
	if err != nil {
		t.Fatalf("SurfaceMovement: %v", err)
	}

	wantFloat(t, "GroundSpeed", sm.GroundSpeed, 15, 0.001)
	wantFloat(t, "Track", sm.Track, 92.8125, 0.001)
}

func testSurfaceRejectNonSurface(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").SurfaceMovement()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

func testSurfaceRejectAirborne(t *testing.T) {
	_, err := mustVelMsg(t, "8D40621D58C382D690C8AC2863A7").SurfaceMovement()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

func testSurfaceRejectShortFrame(t *testing.T) {
	_, err := mustVelMsg(t, "20001910bc45e9").SurfaceMovement()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// A surface position message populates a CPR with Surface set, and decodes
// locally against a reference using the 90-degree surface scaling.
func TestSurfacePosition(t *testing.T) {
	odd := mustVelMsg(t, "8C4841753A9A153237AEF0F275BE")
	even := mustVelMsg(t, "8C4841753AAB238733C8CD4020B1")

	oc, err := odd.CPR()
	if err != nil {
		t.Fatalf("odd CPR: %v", err)
	}

	if oc == nil || !oc.Surface {
		t.Fatal("odd CPR: expected non-nil surface report")
	}

	wantLocal(t, oc, []float64{52, 4}, 52.32056051997815, 4.735735212053572)

	ec, err := even.CPR()
	if err != nil {
		t.Fatalf("even CPR: %v", err)
	}

	if ec == nil || !ec.Surface {
		t.Fatal("even CPR: expected non-nil surface report")
	}

	wantLocal(t, ec, []float64{52, 4}, 52.32304000854492, 4.730472564697266)

	// Surface positions cannot be globally decoded without a reference.
	_, err = adsb.DecodeGlobalPosition(oc, ec)
	if err == nil {
		t.Error("DecodeGlobalPosition: expected error for surface CPR")
	}
}

// A surface even/odd pair decodes to a global position using a reference to
// resolve the zone. The returned position is that of the first argument (the
// message being located), matching DecodeGlobalPosition.
func TestSurfaceGlobalPosition(t *testing.T) {
	odd, err := mustVelMsg(t, "8C4841753A9A153237AEF0F275BE").CPR()
	if err != nil {
		t.Fatalf("odd CPR: %v", err)
	}

	even, err := mustVelMsg(t, "8C4841753AAB238733C8CD4020B1").CPR()
	if err != nil {
		t.Fatalf("even CPR: %v", err)
	}

	ref := []float64{52, 4}

	// The position returned is that of the second argument (the newer frame),
	// matching DecodeGlobalPosition.
	pos, err := adsb.DecodeGlobalPositionRef(odd, even, ref)
	wantPos(t, pos, err, 52.32304000854492, 4.730472564697266)

	pos, err = adsb.DecodeGlobalPositionRef(even, odd, ref)
	wantPos(t, pos, err, 52.32056051997815, 4.735735212053584)

	// A reference in the eastern longitude quadrant selects that quadrant.
	pos, err = adsb.DecodeGlobalPositionRef(even, odd, []float64{52, 100})
	wantPos(t, pos, err, 52.32056051997815, 94.73573521205356)
}

// A reference just west of the antimeridian must select the surface-zone
// candidate just east of it, not a candidate 90 degrees away. The expected
// longitude is computed directly from the four quadrant candidates
// (88.00, 178.00, -92.00, -2.00); the nearest to -179 by wrapped angular
// distance is 178.00.
func TestSurfaceGlobalPositionAntimeridian(t *testing.T) {
	// c2 (even) is located: even Lat decodes to 52.32304, Lon places the
	// longitude base near 88 degrees.
	even := &adsb.CPR{Nb: 17, Surface: true, F: 0, Lat: 115609, Lon: 26471}
	odd := &adsb.CPR{Nb: 17, Surface: true, F: 1, Lat: 39195, Lon: 28281}

	pos, err := adsb.DecodeGlobalPositionRef(odd, even, []float64{52, -179})
	wantPos(t, pos, err, 52.32304000854492, 178.0048942565918)
}

// A southern-hemisphere reference selects the southern latitude candidate.
// The CPRs are built directly to exercise the reference-sign branch.
func TestSurfaceGlobalPositionSouth(t *testing.T) {
	even := &adsb.CPR{Nb: 17, Surface: true, F: 0, Lat: 115609, Lon: 116941}
	odd := &adsb.CPR{Nb: 17, Surface: true, F: 1, Lat: 39195, Lon: 110320}

	// c2 (odd) is located against a southern reference.
	pos, err := adsb.DecodeGlobalPositionRef(even, odd, []float64{-52, 4})
	wantPos(t, pos, err, -37.67943948002185, 3.603276791779905)
}

func TestSurfaceGlobalPositionErrors(t *testing.T) {
	surfEven := &adsb.CPR{Nb: 17, Surface: true, F: 0, Lat: 115609, Lon: 116941}
	surfOdd := &adsb.CPR{Nb: 17, Surface: true, F: 1, Lat: 39195, Lon: 110320}
	air := &adsb.CPR{Nb: 17, Surface: false, F: 0, Lat: 1, Lon: 1}
	ref := []float64{52, 4}

	cases := []struct {
		name   string
		c1, c2 *adsb.CPR
		rp     []float64
	}{
		{"Nil", nil, surfOdd, ref},
		{"Airborne", air, &adsb.CPR{Nb: 17, F: 1}, ref},
		{"NbMismatch", surfEven, &adsb.CPR{Nb: 19, Surface: true, F: 1}, ref},
		{"SameFormat", surfEven, &adsb.CPR{Nb: 17, Surface: true, F: 0}, ref},
		{"BadRef", surfEven, surfOdd, []float64{52}},
		{"LatOutOfRange", surfEven, surfOdd, []float64{91, 4}},
		{"LonOutOfRange", surfEven, surfOdd, []float64{52, 200}},
		{
			"CrossBoundary",
			&adsb.CPR{Nb: 17, Surface: true, F: 0, Lat: 28693, Lon: 116941},
			&adsb.CPR{Nb: 17, Surface: true, F: 1, Lat: 122970, Lon: 110320},
			ref,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := adsb.DecodeGlobalPositionRef(c.c1, c.c2, c.rp)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// f64 returns a pointer to a float64 literal.
func f64(v float64) *float64 {
	return &v
}

// wantPos asserts a decoded [lat, lon] is within ~1e-5 degrees of expected.
func wantPos(t *testing.T, got []float64, err error, lat, lon float64) {
	t.Helper()

	if err != nil {
		t.Fatalf("DecodeGlobalPositionRef: %v", err)
	}

	if math.Abs(got[0]-lat) > 1e-5 {
		t.Errorf("lat = %.10f, want ~%.10f", got[0], lat)
	}

	if math.Abs(got[1]-lon) > 1e-5 {
		t.Errorf("lon = %.10f, want ~%.10f", got[1], lon)
	}
}

// wantLocal asserts a local CPR decode is within ~1e-5 degrees of the
// expected latitude and longitude.
func wantLocal(t *testing.T, c *adsb.CPR, ref []float64, lat, lon float64) {
	t.Helper()

	got, err := c.DecodeLocal(ref)
	if err != nil {
		t.Fatalf("DecodeLocal: %v", err)
	}

	if math.Abs(got[0]-lat) > 1e-5 {
		t.Errorf("lat = %.10f, want ~%.10f", got[0], lat)
	}

	if math.Abs(got[1]-lon) > 1e-5 {
		t.Errorf("lon = %.10f, want ~%.10f", got[1], lon)
	}
}
