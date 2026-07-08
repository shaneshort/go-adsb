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

// BDS 5,1 coarse position report: latitude +45 deg, longitude -90 deg,
// pressure altitude 30000 ft. Vector built from the MB bit ranges of ICAO Doc
// 9871 Table A-2-81 (two's complement per the table notes).
func TestPositionReportCoarse(t *testing.T) {
	p, err := mustVelMsg(t, "A000000090000600000EA6000000").PositionReportCoarse()
	if err != nil {
		t.Fatalf("PositionReportCoarse: %v", err)
	}

	wantFloat(t, "Latitude", p.Latitude, 45, 0.001)
	wantFloat(t, "Longitude", p.Longitude, -90, 0.001)
	wantInt(t, "PressureAltitude", p.PressureAltitude, 30000)
}

// With the status bit clear (here with residual latitude/longitude bits set)
// and the pressure altitude field zeroed, every coarse position field decodes
// to nil.
func TestPositionReportCoarseNoData(t *testing.T) {
	p, err := mustVelMsg(t, "A00000007FFFFFFFFF8000000000").PositionReportCoarse()
	if err != nil {
		t.Fatalf("PositionReportCoarse: %v", err)
	}

	wantNil(t, "Latitude", p.Latitude == nil)
	wantNil(t, "Longitude", p.Longitude == nil)
	wantNil(t, "PressureAltitude", p.PressureAltitude == nil)
}

// A latitude outside the valid -90..90 range and a pressure altitude outside
// the valid -1000..126752 ft range are rejected as nil, while an in-range
// longitude is still returned (ICAO Doc 9871 Table A-2-81).
func TestPositionReportCoarseOutOfRange(t *testing.T) {
	p, err := mustVelMsg(t, "A0000000B0000000000000000000").PositionReportCoarse()
	if err != nil {
		t.Fatalf("PositionReportCoarse: %v", err)
	}

	wantNil(t, "Latitude", p.Latitude == nil)
	wantFloat(t, "Longitude", p.Longitude, 0, 0.001)

	p, err = mustVelMsg(t, "A000000000000000003F7A000000").PositionReportCoarse()
	if err != nil {
		t.Fatalf("PositionReportCoarse: %v", err)
	}

	wantNil(t, "PressureAltitude", p.PressureAltitude == nil)
}

// BDS 5,1 with negative signed fields: latitude -45 deg and pressure altitude
// -800 ft, exercising the latitude and altitude sign bits.
func TestPositionReportCoarseNegative(t *testing.T) {
	p, err := mustVelMsg(t, "A0000000F0000000007F9C000000").PositionReportCoarse()
	if err != nil {
		t.Fatalf("PositionReportCoarse: %v", err)
	}

	wantFloat(t, "Latitude", p.Latitude, -45, 0.001)
	wantInt(t, "PressureAltitude", p.PressureAltitude, -800)
}

// BDS 5,3 air-referenced state vector: magnetic heading 90 deg, IAS 250 kt,
// Mach 0.8, TAS 300 kt, altitude rate +1024 ft/min. Vector built from the MB
// bit ranges of ICAO Doc 9871 Table A-2-83.
func TestAirReferencedStateVector(t *testing.T) {
	v, err := mustVelMsg(t, "A0000000A009F532496210000000").AirReferencedStateVector()
	if err != nil {
		t.Fatalf("AirReferencedStateVector: %v", err)
	}

	wantFloat(t, "MagneticHeading", v.MagneticHeading, 90, 0.001)
	wantFloat(t, "IndicatedAirspeed", v.IndicatedAirspeed, 250, 0.001)
	wantFloat(t, "Mach", v.Mach, 0.8, 0.001)
	wantFloat(t, "TrueAirspeed", v.TrueAirspeed, 300, 0.001)
	wantInt(t, "AltitudeRate", v.AltitudeRate, 1024)
}

// With every status bit clear, the air-referenced state vector fields decode
// to nil even when the value bits carry residual data.
func TestAirReferencedStateVectorNoData(t *testing.T) {
	v, err := mustVelMsg(t, "A00000007FF7FEFFBFFDFF000000").AirReferencedStateVector()
	if err != nil {
		t.Fatalf("AirReferencedStateVector: %v", err)
	}

	wantNil(t, "MagneticHeading", v.MagneticHeading == nil)
	wantNil(t, "IndicatedAirspeed", v.IndicatedAirspeed == nil)
	wantNil(t, "Mach", v.Mach == nil)
	wantNil(t, "TrueAirspeed", v.TrueAirspeed == nil)
	wantNil(t, "AltitudeRate", v.AltitudeRate == nil)
}

// BDS 5,3 with negative signed fields: magnetic heading -90 deg and altitude
// rate -1024 ft/min, exercising the heading and altitude-rate sign bits.
func TestAirReferencedStateVectorNegative(t *testing.T) {
	v, err := mustVelMsg(t, "A0000000E009F5324963F0000000").AirReferencedStateVector()
	if err != nil {
		t.Fatalf("AirReferencedStateVector: %v", err)
	}

	wantFloat(t, "MagneticHeading", v.MagneticHeading, -90, 0.001)
	wantInt(t, "AltitudeRate", v.AltitudeRate, -1024)
}

// BDS 5,F quasi-static parameter monitoring: each 2-bit field is a change
// counter (0 = no data, cycling 1-2-3 on each change). Vector built from the
// MB bit ranges of ICAO Doc 9871 Table A-2-95.
func TestQuasiStaticParameterMonitoring(t *testing.T) {
	q, err := mustVelMsg(t, "A00000008004D9C0000000000000").QuasiStaticParameterMonitoring()
	if err != nil {
		t.Fatalf("QuasiStaticParameterMonitoring: %v", err)
	}

	wantU8(t, "MCPSelectedAltitude", q.MCPSelectedAltitude, 2)
	wantU8(t, "NextWaypoint", q.NextWaypoint, 1)
	wantU8(t, "FMSVerticalMode", q.FMSVerticalMode, 3)
	wantU8(t, "VHFChannel", q.VHFChannel, 1)
	wantU8(t, "MeteorologicalHazards", q.MeteorologicalHazards, 2)
	wantU8(t, "FMSSelectedAltitude", q.FMSSelectedAltitude, 1)
	wantU8(t, "BarometricPressureSetting", q.BarometricPressureSetting, 3)
}

// An all-zero register reports no data (a zero counter) for every parameter.
func TestQuasiStaticParameterMonitoringNoData(t *testing.T) {
	q, err := mustVelMsg(t, "A000000000000000000000000000").QuasiStaticParameterMonitoring()
	if err != nil {
		t.Fatalf("QuasiStaticParameterMonitoring: %v", err)
	}

	if q.MCPSelectedAltitude != 0 || q.NextWaypoint != 0 || q.FMSVerticalMode != 0 ||
		q.VHFChannel != 0 || q.MeteorologicalHazards != 0 || q.FMSSelectedAltitude != 0 ||
		q.BarometricPressureSetting != 0 {
		t.Errorf("expected all-zero counters, got %+v", q)
	}
}

// The position and state registers require a DF20/21 reply.
func TestPositionStateRejectNonReply(t *testing.T) {
	msg := mustVelMsg(t, "8D485020994409940838175B284F")

	_, err := msg.PositionReportCoarse()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("PositionReportCoarse err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.AirReferencedStateVector()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("AirReferencedStateVector err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.QuasiStaticParameterMonitoring()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("QuasiStaticParameterMonitoring err = %v, want ErrNotAvailable", err)
	}
}
