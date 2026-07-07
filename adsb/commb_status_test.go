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

// BDS E,7 transponder status and diagnostics: a representative vector covering
// every subfield type and both ends of the register, built from the MB bit
// assignments of ICAO Doc 9871 Table A-2-231.
func TestTransponderStatus(t *testing.T) {
	s, err := mustVelMsg(t, "A0000000E769A524C7A967000000").TransponderStatus()
	if err != nil {
		t.Fatalf("TransponderStatus: %v", err)
	}

	checkStatusFlags(t, s)
	checkStatusPorts(t, s)
}

// checkStatusFlags asserts the identifier and boolean subfields of the E,7
// register (MB bits 9-26).
func checkStatusFlags(t *testing.T, s *adsb.TransponderStatus) {
	t.Helper()

	if s.SDICode != adsbtype.SDI1 {
		t.Errorf("SDICode = %v, want %v", s.SDICode, adsbtype.SDI1)
	}

	// Non-diversity and failure flags (MB bits 11-16).
	wantBool(t, "NonDiversity", s.NonDiversity, true)
	wantBool(t, "DiversityFailure", s.DiversityFailure, false)
	wantBool(t, "UpperReceiverFailure", s.UpperReceiverFailure, true)
	wantBool(t, "LowerReceiverFailure", s.LowerReceiverFailure, false)
	wantBool(t, "UpperSquitterFailure", s.UpperSquitterFailure, false)
	wantBool(t, "LowerSquitterFailure", s.LowerSquitterFailure, true)

	// Input-status flags (MB bits 17-20, 23).
	wantBool(t, "AirGround1Active", s.AirGround1Active, true)
	wantBool(t, "AirGround2Active", s.AirGround2Active, false)
	wantBool(t, "GPSTimeMark1Active", s.GPSTimeMark1Active, true)
	wantBool(t, "GPSTimeMark2Active", s.GPSTimeMark2Active, false)
	wantBool(t, "ExtendedSquitterDisableActive", s.ExtendedSquitterDisableActive, false)

	// Limiting flags (MB bits 21-22).
	wantBool(t, "ModeSLimitingPowerOn", s.ModeSLimitingPowerOn, false)
	wantBool(t, "ModeSLimiting", s.ModeSLimiting, true)

	// Inactive/failed flags (MB bits 24-26).
	wantBool(t, "ACASInputInactive", s.ACASInputInactive, true)
	wantBool(t, "ADSBOutInactive", s.ADSBOutInactive, false)
	wantBool(t, "SelectedControlInactive", s.SelectedControlInactive, false)
}

// checkStatusPorts asserts the control, source and bus/port subfields of the
// E,7 register (MB bits 27-56).
func checkStatusPorts(t *testing.T, s *adsb.TransponderStatus) {
	t.Helper()

	if s.ControlInputSelection != adsbtype.CIS2 {
		t.Errorf("ControlInputSelection = %v, want %v", s.ControlInputSelection, adsbtype.CIS2)
	}

	if s.AirDataSource != adsbtype.DSS1 {
		t.Errorf("AirDataSource = %v, want %v", s.AirDataSource, adsbtype.DSS1)
	}

	wantBool(t, "AltitudeAlternatePort", s.AltitudeAlternatePort, false)

	if s.AltitudePortAStatus != adsbtype.BST1 {
		t.Errorf("AltitudePortAStatus = %v, want %v", s.AltitudePortAStatus, adsbtype.BST1)
	}

	if s.AltitudePortBStatus != adsbtype.BST2 {
		t.Errorf("AltitudePortBStatus = %v, want %v", s.AltitudePortBStatus, adsbtype.BST2)
	}

	// FMC/GNSS group (MB bits 36-40).
	wantBool(t, "FMCGNSSSourcePort2", s.FMCGNSSSourcePort2, false)
	wantBST(t, "FMCGNSS1BusStatus", s.FMCGNSS1BusStatus, adsbtype.BST1)
	wantBST(t, "FMCGNSS2BusStatus", s.FMCGNSS2BusStatus, adsbtype.BST3)

	// IRS/FMS group (MB bits 41-47).
	if s.IRSAHRSSource != adsbtype.DSS2 {
		t.Errorf("IRSAHRSSource = %v, want %v", s.IRSAHRSSource, adsbtype.DSS2)
	}

	wantBool(t, "IRSFMSSourcePort2", s.IRSFMSSourcePort2, true)
	wantBST(t, "IRSFMSDataConcentrator1Status", s.IRSFMSDataConcentrator1Status, adsbtype.BST1)
	wantBST(t, "IRSFMSDataConcentrator2Status", s.IRSFMSDataConcentrator2Status, adsbtype.BST0)

	// FMC group (MB bits 48-52).
	wantBool(t, "FMCSelectPort2", s.FMCSelectPort2, true)
	wantBST(t, "FMC1BusStatus", s.FMC1BusStatus, adsbtype.BST1)
	wantBST(t, "FMC2BusStatus", s.FMC2BusStatus, adsbtype.BST2)

	// MSP/ATSU/CMU group (MB bits 53-56); the last field ends the register.
	wantBST(t, "MSPATSUCMU1Status", s.MSPATSUCMU1Status, adsbtype.BST1)
	wantBST(t, "MSPATSUCMU2Status", s.MSPATSUCMU2Status, adsbtype.BST3)
}

// wantBST asserts an adsbtype.BST field equals want.
func wantBST(t *testing.T, name string, got, want adsbtype.BST) {
	t.Helper()

	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// The transponder status register requires a DF20/21 reply.
func TestTransponderStatusRejectNonReply(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").TransponderStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
