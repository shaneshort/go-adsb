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
	"kreklow.us/go/go-adsb/adsbtype"
)

// wantHazard asserts a *adsbtype.Hazard is non-nil and equal to want.
func wantHazard(t *testing.T, name string, got *adsbtype.Hazard, want adsbtype.Hazard) {
	t.Helper()

	if got == nil {
		t.Errorf("%s = nil, want %v", name, want)

		return
	}

	if *got != want {
		t.Errorf("%s = %v, want %v", name, *got, want)
	}
}

// BDS 4,4 meteorological routine air report: FOM/source GNSS, wind 200 kt from
// 180 deg, static air temperature +25 C, pressure 1013 hPa, moderate
// turbulence, humidity 50%. Vector built from the MB bit ranges of ICAO Doc
// 9871 Table A-2-68 (temperature cross-checked against readsb).
func TestMeteorologicalRoutineAirReport(t *testing.T) {
	m, err := mustVelMsg(t, "A00000002B2200192FD760000000").MeteorologicalRoutineAirReport()
	if err != nil {
		t.Fatalf("MeteorologicalRoutineAirReport: %v", err)
	}

	if m.FOMSource != adsbtype.FOM2 {
		t.Errorf("FOMSource = %v, want %v", m.FOMSource, adsbtype.FOM2)
	}

	wantFloat(t, "WindSpeed", m.WindSpeed, 200, 0.001)
	wantFloat(t, "WindDirection", m.WindDirection, 180, 0.001)

	if math.Abs(m.StaticAirTemperature-25) > 0.001 {
		t.Errorf("StaticAirTemperature = %g, want ~25", m.StaticAirTemperature)
	}

	wantFloat(t, "AverageStaticPressure", m.AverageStaticPressure, 1013, 0.001)
	wantHazard(t, "Turbulence", m.Turbulence, adsbtype.Hazard2)
	wantFloat(t, "Humidity", m.Humidity, 50, 0.001)
}

// A BDS 4,4 static air temperature with the sign bit set decodes as a negative
// value (two's complement), locking in the sign handling.
func TestMeteorologicalRoutineAirReportNegativeTemp(t *testing.T) {
	m, err := mustVelMsg(t, "A00000002B2201E72FD760000000").MeteorologicalRoutineAirReport()
	if err != nil {
		t.Fatalf("MeteorologicalRoutineAirReport: %v", err)
	}

	if math.Abs(m.StaticAirTemperature-(-25)) > 0.001 {
		t.Errorf("StaticAirTemperature = %g, want ~-25", m.StaticAirTemperature)
	}
}

// The BDS 4,4 static air temperature spans the full 11-bit two's-complement
// range of MB bits 24-34 at a 0.25 C LSB. These boundary vectors lock in the
// extremes: all magnitude bits set with the sign clear (+255.75 C), the sign
// bit alone (-256 C), and the largest signed magnitude (-0.25 C).
func TestMeteorologicalRoutineAirReportTempBoundaries(t *testing.T) {
	cases := []struct {
		name string
		hex  string
		want float64
	}{
		{"MaxPositive", "A0000000000000FFC00000000000", 255.75},
		{"MaxNegative", "A000000000000100000000000000", -256},
		{"MinusQuarter", "A0000000000001FFC00000000000", -0.25},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := mustVelMsg(t, c.hex).MeteorologicalRoutineAirReport()
			if err != nil {
				t.Fatalf("MeteorologicalRoutineAirReport: %v", err)
			}

			if math.Abs(m.StaticAirTemperature-c.want) > 0.001 {
				t.Errorf("StaticAirTemperature = %g, want %g", m.StaticAirTemperature, c.want)
			}
		})
	}
}

// BDS 4,5 meteorological hazard report: severe turbulence, moderate wind
// shear, no microburst report, light icing, severe wake vortex, static air
// temperature +30 C, pressure 1000 hPa, radio height 2000 ft. Vector built
// from the MB bit ranges of ICAO Doc 9871 Table A-2-69.
func TestMeteorologicalHazardReport(t *testing.T) {
	m, err := mustVelMsg(t, "A0000000F85F1E2FA20FA0000000").MeteorologicalHazardReport()
	if err != nil {
		t.Fatalf("MeteorologicalHazardReport: %v", err)
	}

	wantHazard(t, "Turbulence", m.Turbulence, adsbtype.Hazard3)
	wantHazard(t, "WindShear", m.WindShear, adsbtype.Hazard2)
	wantNil(t, "Microburst", m.Microburst == nil)
	wantHazard(t, "Icing", m.Icing, adsbtype.Hazard1)
	wantHazard(t, "WakeVortex", m.WakeVortex, adsbtype.Hazard3)
	wantFloat(t, "StaticAirTemperature", m.StaticAirTemperature, 30, 0.001)
	wantFloat(t, "AverageStaticPressure", m.AverageStaticPressure, 1000, 0.001)
	wantFloat(t, "RadioHeight", m.RadioHeight, 2000, 0.001)
}

// With every status bit clear, the optional meteorological fields decode to
// nil. The BDS 4,4 static air temperature has no status bit and always
// decodes (to zero for an empty field).
func TestMeteorologicalNoData(t *testing.T) {
	msg := mustVelMsg(t, "A000000000000000000000000000")

	r, err := msg.MeteorologicalRoutineAirReport()
	if err != nil {
		t.Fatalf("MeteorologicalRoutineAirReport: %v", err)
	}

	wantNil(t, "WindSpeed", r.WindSpeed == nil)
	wantNil(t, "WindDirection", r.WindDirection == nil)
	wantNil(t, "AverageStaticPressure", r.AverageStaticPressure == nil)
	wantNil(t, "Turbulence", r.Turbulence == nil)
	wantNil(t, "Humidity", r.Humidity == nil)

	if r.StaticAirTemperature != 0 {
		t.Errorf("StaticAirTemperature = %g, want 0", r.StaticAirTemperature)
	}

	h, err := msg.MeteorologicalHazardReport()
	if err != nil {
		t.Fatalf("MeteorologicalHazardReport: %v", err)
	}

	wantNil(t, "Turbulence", h.Turbulence == nil)
	wantNil(t, "WindShear", h.WindShear == nil)
	wantNil(t, "Microburst", h.Microburst == nil)
	wantNil(t, "Icing", h.Icing == nil)
	wantNil(t, "WakeVortex", h.WakeVortex == nil)
	wantNil(t, "StaticAirTemperature", h.StaticAirTemperature == nil)
	wantNil(t, "AverageStaticPressure", h.AverageStaticPressure == nil)
	wantNil(t, "RadioHeight", h.RadioHeight == nil)
}

// The meteorological reports require a DF20/21 reply.
func TestMeteorologicalRejectNonReply(t *testing.T) {
	msg := mustVelMsg(t, "8D485020994409940838175B284F")

	_, err := msg.MeteorologicalRoutineAirReport()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("MeteorologicalRoutineAirReport err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.MeteorologicalHazardReport()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("MeteorologicalHazardReport err = %v, want ErrNotAvailable", err)
	}
}
