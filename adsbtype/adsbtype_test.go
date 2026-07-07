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

package adsbtype_test

import (
	"fmt"
	"testing"

	"kreklow.us/go/go-adsb/adsbtype"
)

// TestConst tests string formatting of constant values.
func TestConst(t *testing.T) {
	for val, out := range map[any]string{
		adsbtype.CA0: "adsbtype.CA: Level 1",
		adsbtype.CC0: "adsbtype.CC: Not supported",
		adsbtype.CF0: "adsbtype.CF: ADS-B message, non-transponder device with ICAO address",
		adsbtype.DF0: "adsbtype.DF: Short air-air surveillance (ACAS)",
		adsbtype.DR0: "adsbtype.DR: No request",
		adsbtype.FS0: "adsbtype.FS: No alert, no SPI, airborne",
		adsbtype.RI0: "adsbtype.RI: No ACAS",
		adsbtype.SL0: "adsbtype.SL: ACAS inoperative",
		adsbtype.VS0: "adsbtype.VS: Airborne",

		adsbtype.ATS0:  "adsbtype.ATS: Barometric altitude",
		adsbtype.BDS02: "adsbtype.BDS: Linked Comm-B, segment 2",
		adsbtype.BDSE1: "adsbtype.BDS: Reserved for Mode S BITE (built-in test equipment)",
		adsbtype.BDSE2: "adsbtype.BDS: Reserved for Mode S BITE (built-in test equipment)",
		adsbtype.SSS0:  "adsbtype.SSS: No condition information",
		adsbtype.TRS0:  "adsbtype.TRS: No capability",
		adsbtype.D0:    "adsbtype.AcCat: No ADS-B emitter category information",
		adsbtype.D2:    "adsbtype.AcCat: Reserved",
		adsbtype.EPS0:  "adsbtype.EPS: No emergency",
		adsbtype.EPS5:  "adsbtype.EPS: Unlawful interference",
		adsbtype.HRD0:  "adsbtype.HRD: True north",
		adsbtype.HRD1:  "adsbtype.HRD: Magnetic north",
		adsbtype.VRS0:  "adsbtype.VRS: GNSS",
		adsbtype.VRS1:  "adsbtype.VRS: Barometric",
		adsbtype.AST0:  "adsbtype.AST: Indicated airspeed (IAS)",
		adsbtype.AST1:  "adsbtype.AST: True airspeed (TAS)",

		adsbtype.FOM2:    "adsbtype.FOM: GNSS",
		adsbtype.FOM(7):  "adsbtype.FOM: Reserved",
		adsbtype.Hazard0: "adsbtype.Hazard: Nil",
		adsbtype.Hazard3: "adsbtype.Hazard: Severe",
		adsbtype.IDF0:    "adsbtype.IDF: Part number",
		adsbtype.IDF1:    "adsbtype.IDF: Character",
		adsbtype.IDF2:    "adsbtype.IDF: Reserved",
		adsbtype.BDS18:   "adsbtype.BDS: Mode S specific services GICB capability report (1 of 5)",

		adsbtype.Designator1: "adsbtype.Designator: Comm-B interrogator identifier",
		adsbtype.SDI0:        "adsbtype.SDI: Not used",
		adsbtype.SDI1:        "adsbtype.SDI: Side 1",
		adsbtype.CIS2:        "adsbtype.CIS: Port B or 2",
		adsbtype.DSS0:        "adsbtype.DSS: No data or not used",
		adsbtype.DSS1:        "adsbtype.DSS: Source 1 in use",
		adsbtype.BST3:        "adsbtype.BST: Fail",

		adsbtype.TYPE0: "adsbtype.TYPE: No position information",
	} {
		result := fmt.Sprintf("%T: %s", val, val)
		if result != out {
			t.Errorf("expected %s | received %s\n", out, result)
		}
	}
}

func TestConstUnknown(t *testing.T) {
	for val, out := range map[any]string{
		adsbtype.CA(99): "adsbtype.CA: Unknown value 99",
		adsbtype.CC(99): "adsbtype.CC: Unknown value 99",
		adsbtype.CF(99): "adsbtype.CF: Unknown value 99",
		adsbtype.DF(99): "adsbtype.DF: Unknown value 99",
		adsbtype.DR(99): "adsbtype.DR: Unknown value 99",
		adsbtype.FS(99): "adsbtype.FS: Unknown value 99",
		adsbtype.RI(99): "adsbtype.RI: Unknown value 99",
		adsbtype.SL(99): "adsbtype.SL: Unknown value 99",
		adsbtype.VS(99): "adsbtype.VS: Unknown value 99",

		adsbtype.ATS(99):   "adsbtype.ATS: Unknown value 99",
		adsbtype.BDS(0x99): "adsbtype.BDS: Unknown value 99",
		adsbtype.SSS(99):   "adsbtype.SSS: Unknown value 99",
		adsbtype.TRS(99):   "adsbtype.TRS: Unknown value 99",
		adsbtype.EPS(99):   "adsbtype.EPS: Unknown value 99",
		adsbtype.HRD(99):   "adsbtype.HRD: Unknown value 99",
		adsbtype.VRS(99):   "adsbtype.VRS: Unknown value 99",
		adsbtype.AST(99):   "adsbtype.AST: Unknown value 99",

		adsbtype.FOM(99):        "adsbtype.FOM: Unknown value 99",
		adsbtype.Hazard(99):     "adsbtype.Hazard: Unknown value 99",
		adsbtype.IDF(99):        "adsbtype.IDF: Unknown value 99",
		adsbtype.Designator(99): "adsbtype.Designator: Unknown value 99",
		adsbtype.SDI(99):        "adsbtype.SDI: Unknown value 99",
		adsbtype.CIS(99):        "adsbtype.CIS: Unknown value 99",
		adsbtype.DSS(99):        "adsbtype.DSS: Unknown value 99",
		adsbtype.BST(99):        "adsbtype.BST: Unknown value 99",

		adsbtype.TYPE(99): "adsbtype.TYPE: Unknown value 99",
	} {
		result := fmt.Sprintf("%T: %s", val, val)
		if result != out {
			t.Errorf("expected %s | received %s\n", out, result)
		}
	}
}
