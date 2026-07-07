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

// Surveillance-field test vectors, constructed from explicit subfield values
// using the reply formats of ICAO Annex 10 Vol IV.
const (
	survDF0  = "06818000000000"               // VS=1, CC=1, SL=4, RI=3
	survDF4  = "220AA000000000"               // FS=2, DR=1, UM: IIS=5, IDS=1
	survDF11 = "5D000000000000"               // CA=5
	survDF16 = "8042000000000000000000000000" // VS=0, SL=2, RI=4
	survDF18 = "9200000000000000000000000000" // CF=2
	survDF17 = "8D485020994409940838175B284F" // extended squitter, CA=5
)

// FlightStatus decodes the flight status field of a surveillance or Comm-B
// reply (DF4/5/20/21); other formats return ErrNotAvailable.
func TestFlightStatus(t *testing.T) {
	fs, err := mustVelMsg(t, survDF4).FlightStatus()
	if err != nil {
		t.Fatalf("FlightStatus: %v", err)
	}

	if fs != adsbtype.FS2 {
		t.Errorf("FlightStatus = %v, want %v", fs, adsbtype.FS2)
	}

	_, err = mustVelMsg(t, survDF17).FlightStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("FlightStatus err = %v, want ErrNotAvailable", err)
	}
}

// DownlinkRequest decodes the downlink request field (DF4/5/20/21).
func TestDownlinkRequest(t *testing.T) {
	dr, err := mustVelMsg(t, survDF4).DownlinkRequest()
	if err != nil {
		t.Fatalf("DownlinkRequest: %v", err)
	}

	if dr != adsbtype.DR1 {
		t.Errorf("DownlinkRequest = %v, want %v", dr, adsbtype.DR1)
	}

	_, err = mustVelMsg(t, survDF17).DownlinkRequest()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("DownlinkRequest err = %v, want ErrNotAvailable", err)
	}
}

// UtilityMessage decodes the interrogator identifier and designator subfields
// (DF4/5/20/21).
func TestUtilityMessage(t *testing.T) {
	um, err := mustVelMsg(t, survDF4).UtilityMessage()
	if err != nil {
		t.Fatalf("UtilityMessage: %v", err)
	}

	if um.InterrogatorID != 5 {
		t.Errorf("InterrogatorID = %d, want 5", um.InterrogatorID)
	}

	if um.Designator != adsbtype.Designator1 {
		t.Errorf("Designator = %v, want %v", um.Designator, adsbtype.Designator1)
	}

	_, err = mustVelMsg(t, survDF17).UtilityMessage()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("UtilityMessage err = %v, want ErrNotAvailable", err)
	}
}

// The flight status, downlink request and utility message fields occupy the
// same bit positions in every reply that carries them (DF4/5/20/21), so each
// format decodes to identical values. TestFlightStatus and its siblings cover
// DF4; this exercises the remaining formats.
func TestSurveillanceCommBFormats(t *testing.T) {
	for _, hex := range []string{
		"2A0AA000000000",               // DF5, FS=2 DR=1 IIS=5 IDS=1
		"A20AA00000000000000000000000", // DF20
		"AA0AA00000000000000000000000", // DF21
	} {
		t.Run(hex[:2], func(t *testing.T) {
			m := mustVelMsg(t, hex)

			fs, err := m.FlightStatus()
			if err != nil {
				t.Fatalf("FlightStatus: %v", err)
			}

			if fs != adsbtype.FS2 {
				t.Errorf("FlightStatus = %v, want %v", fs, adsbtype.FS2)
			}

			dr, err := m.DownlinkRequest()
			if err != nil {
				t.Fatalf("DownlinkRequest: %v", err)
			}

			if dr != adsbtype.DR1 {
				t.Errorf("DownlinkRequest = %v, want %v", dr, adsbtype.DR1)
			}

			um, err := m.UtilityMessage()
			if err != nil {
				t.Fatalf("UtilityMessage: %v", err)
			}

			if um.InterrogatorID != 5 || um.Designator != adsbtype.Designator1 {
				t.Errorf("UtilityMessage = %+v, want {InterrogatorID:5 Designator:%v}",
					um, adsbtype.Designator1)
			}
		})
	}
}

// VerticalStatus decodes the vertical status field of an air-air surveillance
// reply (DF0/16).
func TestVerticalStatus(t *testing.T) {
	vs, err := mustVelMsg(t, survDF0).VerticalStatus()
	if err != nil {
		t.Fatalf("VerticalStatus: %v", err)
	}

	if vs != adsbtype.VS1 {
		t.Errorf("VerticalStatus = %v, want %v", vs, adsbtype.VS1)
	}

	vs, err = mustVelMsg(t, survDF16).VerticalStatus()
	if err != nil {
		t.Fatalf("VerticalStatus (DF16): %v", err)
	}

	if vs != adsbtype.VS0 {
		t.Errorf("VerticalStatus (DF16) = %v, want %v", vs, adsbtype.VS0)
	}

	_, err = mustVelMsg(t, survDF4).VerticalStatus()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("VerticalStatus err = %v, want ErrNotAvailable", err)
	}
}

// SensitivityLevel decodes the ACAS sensitivity level field (DF0/16).
func TestSensitivityLevel(t *testing.T) {
	sl, err := mustVelMsg(t, survDF0).SensitivityLevel()
	if err != nil {
		t.Fatalf("SensitivityLevel: %v", err)
	}

	if sl != adsbtype.SL4 {
		t.Errorf("SensitivityLevel = %v, want %v", sl, adsbtype.SL4)
	}

	sl, err = mustVelMsg(t, survDF16).SensitivityLevel()
	if err != nil {
		t.Fatalf("SensitivityLevel (DF16): %v", err)
	}

	if sl != adsbtype.SL2 {
		t.Errorf("SensitivityLevel (DF16) = %v, want %v", sl, adsbtype.SL2)
	}

	_, err = mustVelMsg(t, survDF4).SensitivityLevel()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("SensitivityLevel err = %v, want ErrNotAvailable", err)
	}
}

// ReplyInformation decodes the reply information field (DF0/16).
func TestReplyInformation(t *testing.T) {
	ri, err := mustVelMsg(t, survDF0).ReplyInformation()
	if err != nil {
		t.Fatalf("ReplyInformation: %v", err)
	}

	if ri != adsbtype.RI3 {
		t.Errorf("ReplyInformation = %v, want %v", ri, adsbtype.RI3)
	}

	ri, err = mustVelMsg(t, survDF16).ReplyInformation()
	if err != nil {
		t.Fatalf("ReplyInformation (DF16): %v", err)
	}

	if ri != adsbtype.RI4 {
		t.Errorf("ReplyInformation (DF16) = %v, want %v", ri, adsbtype.RI4)
	}

	_, err = mustVelMsg(t, survDF4).ReplyInformation()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("ReplyInformation err = %v, want ErrNotAvailable", err)
	}
}

// CrossLinkCapability decodes the cross-link capability field (DF0 only).
func TestCrossLinkCapability(t *testing.T) {
	cc, err := mustVelMsg(t, survDF0).CrossLinkCapability()
	if err != nil {
		t.Fatalf("CrossLinkCapability: %v", err)
	}

	if cc != adsbtype.CC1 {
		t.Errorf("CrossLinkCapability = %v, want %v", cc, adsbtype.CC1)
	}

	_, err = mustVelMsg(t, survDF16).CrossLinkCapability()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("CrossLinkCapability err = %v, want ErrNotAvailable", err)
	}
}

// Capability decodes the transponder capability field (DF11/17).
func TestCapability(t *testing.T) {
	ca, err := mustVelMsg(t, survDF11).Capability()
	if err != nil {
		t.Fatalf("Capability: %v", err)
	}

	if ca != adsbtype.CA5 {
		t.Errorf("Capability = %v, want %v", ca, adsbtype.CA5)
	}

	ca, err = mustVelMsg(t, survDF17).Capability()
	if err != nil {
		t.Fatalf("Capability (DF17): %v", err)
	}

	if ca != adsbtype.CA5 {
		t.Errorf("Capability (DF17) = %v, want %v", ca, adsbtype.CA5)
	}

	_, err = mustVelMsg(t, survDF0).Capability()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("Capability err = %v, want ErrNotAvailable", err)
	}
}

// ControlField decodes the control field of a non-transponder extended
// squitter (DF18).
func TestControlField(t *testing.T) {
	cf, err := mustVelMsg(t, survDF18).ControlField()
	if err != nil {
		t.Fatalf("ControlField: %v", err)
	}

	if cf != adsbtype.CF2 {
		t.Errorf("ControlField = %v, want %v", cf, adsbtype.CF2)
	}

	_, err = mustVelMsg(t, survDF17).ControlField()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("ControlField err = %v, want ErrNotAvailable", err)
	}
}
