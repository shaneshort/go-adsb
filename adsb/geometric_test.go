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

// GNSS-height airborne position messages (type codes 20-22) encode the
// altitude identically to barometric (DO-260B §2.2.3.2.3.4.3: shared Q-bit /
// 25-foot encoding in feet); only the source differs. The vectors below are
// the canonical airborne-position example re-typed to codes 20-22, so the
// altitude decodes to the same 38000 ft, reported as geometric.
func TestGeometricAltitude(t *testing.T) {
	for _, c := range []struct {
		name string
		hex  string
	}{
		{"TC20", "8D40621DA0C382D690C8AC2863A7"},
		{"TC21", "8D40621DA8C382D690C8AC2863A7"},
		{"TC22", "8D40621DB0C382D690C8AC2863A7"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := mustVelMsg(t, c.hex)

			alt, err := m.Alt()
			if err != nil {
				t.Fatalf("Alt: %v", err)
			}

			if alt != 38000 {
				t.Errorf("Alt = %d, want 38000", alt)
			}

			src, err := m.AltitudeSource()
			if err != nil {
				t.Fatalf("AltitudeSource: %v", err)
			}

			if src != adsb.AltitudeGeometric {
				t.Errorf("AltitudeSource = %v, want Geometric", src)
			}

			cpr, err := m.CPR()
			if err != nil {
				t.Fatalf("CPR: %v", err)
			}

			if cpr == nil || cpr.Surface {
				t.Error("expected a non-nil airborne CPR")
			}
		})
	}
}

// A GNSS-height position with a Q-bit-0 (Gillham, 100-foot) altitude field
// exercises the shared Gillham decode path for geometric altitude, locking
// down that TC20-22 use the same encoding as barometric across both Q-bit
// values. The vector is derived from the repository's DF4 Gillham test
// (1300 ft) re-encoded into the TC20 extended-squitter altitude subfield.
func TestGeometricAltitudeGillham(t *testing.T) {
	m := mustVelMsg(t, "8D40621DA082A2D690C8AC2863A7")

	alt, err := m.Alt()
	if err != nil {
		t.Fatalf("Alt: %v", err)
	}

	if alt != 1300 {
		t.Errorf("Alt = %d, want 1300", alt)
	}

	src, err := m.AltitudeSource()
	if err != nil {
		t.Fatalf("AltitudeSource: %v", err)
	}

	if src != adsb.AltitudeGeometric {
		t.Errorf("AltitudeSource = %v, want Geometric", src)
	}
}

// AltitudeSource reports barometric for surveillance replies and TC9-18
// airborne positions, geometric for TC20-22, and ErrNotAvailable when the
// message carries no altitude.
func TestAltitudeSource(t *testing.T) {
	baroES, err := mustVelMsg(t, "8D40621D58C382D690C8AC2863A7").AltitudeSource()
	if err != nil {
		t.Fatalf("TC11 AltitudeSource: %v", err)
	}

	if baroES != adsb.AltitudeBarometric {
		t.Errorf("TC11 source = %v, want Barometric", baroES)
	}

	baroSurv, err := mustVelMsg(t, "20001910bc45e9").AltitudeSource()
	if err != nil {
		t.Fatalf("DF4 AltitudeSource: %v", err)
	}

	if baroSurv != adsb.AltitudeBarometric {
		t.Errorf("DF4 source = %v, want Barometric", baroSurv)
	}

	_, err = mustVelMsg(t, "5dac22c54b7a07").AltitudeSource()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("DF11 err = %v, want ErrNotAvailable", err)
	}

	// A DF17 extended squitter that is not a position message (TC19 velocity)
	// carries no altitude.
	_, err = mustVelMsg(t, "8D485020994409940838175B284F").AltitudeSource()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("TC19 err = %v, want ErrNotAvailable", err)
	}
}

func TestAltitudeSourceString(t *testing.T) {
	for src, want := range map[adsb.AltitudeSource]string{
		adsb.AltitudeBarometric: "Barometric",
		adsb.AltitudeGeometric:  "Geometric",
		adsb.AltitudeSource(9):  "Unknown value 9",
	} {
		if got := src.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	}
}
