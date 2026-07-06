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

// Category vectors were constructed from explicit type-code / category-code
// values and cross-checked against pyModeS v3.3.0. The category set is
// selected by the type code (TC4 = set A, TC3 = B, TC2 = C, TC1 = D).
func TestCategory(t *testing.T) {
	cases := []struct {
		name string
		hex  string
		want adsbtype.AcCat
		desc string // expected String() output
	}{
		{"TC4_SetA_Heavy", "8D4840D625205056671820000000", adsbtype.A5, "Heavy (> 300000 lbs)"},
		{"TC3_SetB_Glider", "8D4840D6191CC244152820000000", adsbtype.B1, "Glider / sailplane"},
		{"TC2_SetC_Obstacle", "8D4840D6133C24D4820820000000", adsbtype.C3, "Point obstacle (includes tethered balloons)"},
		{"TC4_SetA_NoInfo", "8D4840D62038F0C1520820000000", adsbtype.A0, "No ADS-B emitter category information"},
		{"TC1_SetD_Reserved", "8D4840D60A4C5504820820000000", adsbtype.D2, "Reserved"},
		{"TC1_SetD_NoInfo", "8D4840D6084C5504C20820000000", adsbtype.D0, "No ADS-B emitter category information"},
		{"DF18_TC4_SetA_Heavy", "904840D6255094C2C60820000000", adsbtype.A5, "Heavy (> 300000 lbs)"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mustVelMsg(t, c.hex).Category()
			if err != nil {
				t.Fatalf("Category: %v", err)
			}

			if got != c.want {
				t.Errorf("Category = %q, want %q", got, c.want)
			}

			if got.String() != c.desc {
				t.Errorf("Category.String() = %q, want %q", got.String(), c.desc)
			}
		})
	}
}

// A velocity message (type code 19) is not an identification message.
func TestCategoryRejectNonIdent(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").Category()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}

// A short-frame surveillance message (DF4) has no extended squitter.
func TestCategoryRejectShortFrame(t *testing.T) {
	_, err := mustVelMsg(t, "20001910bc45e9").Category()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
