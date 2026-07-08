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

// BDS 5,2 fine position report: FOM/source 13 (GNSS height), fine latitude
// +0.3515625 deg, fine longitude -0.3515625 deg (two's complement), GNSS
// height 30000 ft. Vector built from the MB bit ranges of ICAO Doc 9871 Table
// A-2-82.
func TestPositionReportFine(t *testing.T) {
	p, err := mustVelMsg(t, "A0000000EA000180000EA6000000").PositionReportFine()
	if err != nil {
		t.Fatalf("PositionReportFine: %v", err)
	}

	if p.FOMSource != 13 {
		t.Errorf("FOMSource = %d, want 13", p.FOMSource)
	}

	if !p.AltitudeIsGNSSHeight {
		t.Error("AltitudeIsGNSSHeight = false, want true")
	}

	wantFloat(t, "LatitudeFine", p.LatitudeFine, 0.3515625, 0.000001)
	wantFloat(t, "LongitudeFine", p.LongitudeFine, -0.3515625, 0.000001)
	wantInt(t, "Altitude", p.Altitude, 30000)
}

// The fine latitude and longitude are 18-bit two's-complement values (ICAO Doc
// 9871 Table A-2-82 Note 2). These cases guard the sign decode across the
// boundary: zero, the maximum positive value, the first (most) negative value
// and a normal negative value.
func TestPositionReportFineSigned(t *testing.T) {
	for _, c := range []struct {
		name string
		hex  string
		want float64
	}{
		{"Zero", "A000000080000000000000000000", 0},
		{"MaxPositive", "A000000083FFFE00000000000000", 0.7031196},
		{"FirstNegative", "A000000084000000000000000000", -0.703125},
		{"NormalNegative", "A000000086000000000000000000", -0.3515625},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, err := mustVelMsg(t, c.hex).PositionReportFine()
			if err != nil {
				t.Fatalf("PositionReportFine: %v", err)
			}

			wantFloat(t, "LatitudeFine", p.LatitudeFine, c.want, 0.000001)
		})
	}
}

// With the status bit clear (here with residual latitude/longitude bits) and
// the altitude field zeroed, the fine position fields decode to nil.
func TestPositionReportFineNoData(t *testing.T) {
	p, err := mustVelMsg(t, "A00000007FFFFFFFFF8000000000").PositionReportFine()
	if err != nil {
		t.Fatalf("PositionReportFine: %v", err)
	}

	wantNil(t, "LatitudeFine", p.LatitudeFine == nil)
	wantNil(t, "LongitudeFine", p.LongitudeFine == nil)
	wantNil(t, "Altitude", p.Altitude == nil)
}

// A FOM/source below 12 marks the altitude as pressure-altitude, and a
// negative pressure altitude exercises the altitude sign bit.
func TestPositionReportFinePressureAltitude(t *testing.T) {
	p, err := mustVelMsg(t, "A000000098000000007F9C000000").PositionReportFine()
	if err != nil {
		t.Fatalf("PositionReportFine: %v", err)
	}

	if p.AltitudeIsGNSSHeight {
		t.Error("AltitudeIsGNSSHeight = true, want false")
	}

	wantInt(t, "Altitude", p.Altitude, -800)
}

// A pressure altitude outside the valid range is rejected as nil.
func TestPositionReportFineAltitudeOutOfRange(t *testing.T) {
	p, err := mustVelMsg(t, "A0000000E8000000003F7A000000").PositionReportFine()
	if err != nil {
		t.Fatalf("PositionReportFine: %v", err)
	}

	wantNil(t, "Altitude", p.Altitude == nil)
}

// The fine position report requires a DF20/21 reply.
func TestPositionReportFineRejectNonReply(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").PositionReportFine()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
