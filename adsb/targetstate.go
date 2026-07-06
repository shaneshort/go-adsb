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

package adsb

import "github.com/ccoveille/go-safecast/v2"

const (
	// targetStateType is the extended squitter type code for the target
	// state and status message (BDS 6,2).
	targetStateType = 29

	// selectedAltitudeStep is the resolution in feet per count of the
	// selected altitude field.
	selectedAltitudeStep = 32

	// baroPressureStep is the resolution in millibars per count of the
	// barometric pressure setting field.
	baroPressureStep = 0.8

	// baroPressureBase is the fixed offset in millibars subtracted from the
	// barometric pressure setting before encoding.
	baroPressureBase = 800

	// targetHeadingSteps is the number of discrete steps in the 9-bit signed
	// selected heading field spanning a full revolution.
	targetHeadingSteps = 512
)

// TargetState is a decoded ADS-B target state and status message (extended
// squitter type code 29, subtype 1, BDS 6,2). It reports the aircraft's
// selected (intended) altitude and heading, the barometric pressure setting,
// and the status of the navigation and TCAS systems.
//
// Selected altitude, barometric pressure and selected heading are pointers: a
// nil pointer means the field carried the "no data" sentinel. The autopilot,
// VNAV, altitude-hold and approach mode flags are meaningful only when
// ModeBitsValid is true.
type TargetState struct {
	SILSupplement             uint8    // SIL per-sample vs per-hour
	SelectedAltitudeFMS       bool     // false = MCP/FCU source, true = FMS source
	SelectedAltitude          *int     // feet
	BarometricPressureSetting *float64 // millibars
	SelectedHeading           *float64 // degrees clockwise from true north, 0..360

	NACp    uint8 // navigation accuracy category for position
	NICBaro bool  // barometric altitude integrity code
	SIL     uint8 // source integrity level

	ModeBitsValid    bool // MCP/FCU mode bits (autopilot, VNAV, altitude hold, approach) are being populated
	AutopilotEngaged bool
	VNAVEngaged      bool
	AltitudeHold     bool
	ApproachMode     bool
	TCASOperational  bool
	LNAVEngaged      bool // lateral navigation mode (ME 54); not gated by ModeBitsValid
}

// TargetState returns the decoded target state and status (extended squitter
// type code 29, subtype 1, BDS 6,2). It returns an error wrapping
// ErrNotAvailable unless the message is a DF17/DF18 extended squitter with
// type code 29 and subtype 1; subtype 0 is the reserved DO-260A
// trajectory-change format and is rejected.
func (m *Message) TargetState() (*TargetState, error) {
	tc, err := m.raw.ESType()
	if err != nil {
		return nil, newError(err, "error retrieving target state")
	}

	if tc != targetStateType {
		return nil, newErrorf(ErrNotAvailable,
			"error retrieving target state from type %d", tc)
	}

	r := m.raw

	if sub := r.esbits(6, 7); sub != 1 {
		return nil, newErrorf(ErrNotAvailable,
			"unsupported target state subtype %d", sub)
	}

	ts := &TargetState{
		SILSupplement:       asU8(r.esbits(8, 8)),
		SelectedAltitudeFMS: r.esbits(9, 9) == 1,
		NACp:                asU8(r.esbits(40, 43)),
		NICBaro:             r.esbits(44, 44) == 1,
		SIL:                 asU8(r.esbits(45, 46)),
		ModeBitsValid:       r.esbits(47, 47) == 1,
		AutopilotEngaged:    r.esbits(48, 48) == 1,
		VNAVEngaged:         r.esbits(49, 49) == 1,
		AltitudeHold:        r.esbits(50, 50) == 1,
		ApproachMode:        r.esbits(52, 52) == 1,
		TCASOperational:     r.esbits(53, 53) == 1,
		// ME 54 is LNAV in the published DO-260B (§2.2.3.2.7.1.3.18) and in
		// dump1090/readsb/rs1090; the WP30-18 v4.2 draft marks it reserved.
		LNAVEngaged: r.esbits(54, 54) == 1,
	}

	if alt := r.esbits(10, 20); alt != 0 {
		feet := safecast.MustConvert[int](alt-1) * selectedAltitudeStep
		ts.SelectedAltitude = &feet
	}

	if bp := r.esbits(21, 29); bp != 0 {
		mb := baroPressureBase + float64(bp-1)*baroPressureStep
		ts.BarometricPressureSetting = &mb
	}

	// Selected heading is valid only when its status bit (ME 30) is set. The
	// 9-bit value (sign bit 31 as MSB, magnitude bits 32-39) spans a full
	// revolution.
	if r.esbits(30, 30) == 1 {
		raw := (r.esbits(31, 31) << 8) | r.esbits(32, 39)
		hdg := float64(raw) * 360 / targetHeadingSteps
		ts.SelectedHeading = &hdg
	}

	return ts, nil
}
