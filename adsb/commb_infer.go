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

import "kreklow.us/go/go-adsb/adsbtype"

// Comm-B register inference limits. DF20/21 replies carry no BDS code, so the
// register is inferred by checking each candidate's status bits, reserved
// bits and value ranges for plausibility. The ranges follow the validation
// used by dump1090/readsb.
const (
	minSelectedAlt = 1000  // feet
	maxSelectedAlt = 50000 // feet
	minBaroSetting = 900   // millibars
	maxBaroSetting = 1100  // millibars
	minSpeed       = 50    // knots
	maxSpeed       = 700   // knots
	maxRollAngle   = 40    // degrees
	maxTrackRate   = 10    // degrees/second
	minMach        = 0.1
	maxMach        = 0.9
	maxVertRate    = 6000 // feet/minute

	bds10Code = 0x10 // BDS 1,0 self-identifying prefix (ME 1-8)
	bds20Code = 0x20 // BDS 2,0 self-identifying prefix (ME 1-8)
)

// InferBDS returns the Comm-B BDS registers whose format is consistent with
// the MB field of a DF20/21 reply. Because these replies carry no explicit
// register identifier, the result is a best-effort set of candidates: it may
// be empty, or contain more than one register when the field is ambiguous.
// It returns an error wrapping ErrNotAvailable unless the message is a Comm-B
// reply.
func (m *Message) InferBDS() ([]adsbtype.BDS, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	var candidates []adsbtype.BDS

	for _, c := range []struct {
		bds   adsbtype.BDS
		valid func(*RawMessage) bool
	}{
		{adsbtype.BDS10, validBDS10},
		{adsbtype.BDS20, validBDS20},
		{adsbtype.BDS40, validBDS40},
		{adsbtype.BDS50, validBDS50},
		{adsbtype.BDS60, validBDS60},
	} {
		if c.valid(r) {
			candidates = append(candidates, c.bds)
		}
	}

	return candidates, nil
}

// validBDS10 reports whether the MB field is a data link capability report,
// which self-identifies with the code 0x10 in its first eight bits.
func validBDS10(r *RawMessage) bool {
	return r.mbbits(1, 8) == bds10Code
}

// validBDS20 reports whether the MB field is an aircraft identification
// report, which self-identifies with the code 0x20 in its first eight bits.
func validBDS20(r *RawMessage) bool {
	return r.mbbits(1, 8) == bds20Code
}

// validBDS40 reports whether the MB field is plausibly a selected vertical
// intention report (BDS 4,0).
func validBDS40(r *RawMessage) bool {
	// Reserved bits must be zero.
	if r.mbbits(40, 47) != 0 || r.mbbits(52, 53) != 0 {
		return false
	}

	mcp, fms, baro := r.mbbits(1, 1) == 1, r.mbbits(14, 14) == 1, r.mbbits(27, 27) == 1
	if !mcp && !fms && !baro && r.mbbits(48, 48) != 1 && r.mbbits(54, 54) != 1 {
		return false
	}

	if !altPlausible(r, mcp, 2, 13) || !altPlausible(r, fms, 15, 26) {
		return false
	}

	baroRaw := r.mbbits(28, 39)
	if baro {
		mb := baroPressureBase + float64(baroRaw)*commBBaroStep

		return mb >= minBaroSetting && mb <= maxBaroSetting
	}

	return baroRaw == 0
}

// altPlausible checks a selected-altitude subfield: in range when its status
// bit is set, or zero when clear.
func altPlausible(r *RawMessage, valid bool, n, z int) bool {
	raw := r.mbbits(n, z)
	if !valid {
		return raw == 0
	}

	alt := float64(raw) * commBAltStep

	return alt >= minSelectedAlt && alt <= maxSelectedAlt
}

// validBDS50 reports whether the MB field is plausibly a track and turn
// report (BDS 5,0).
func validBDS50(r *RawMessage) bool {
	if r.mbbits(1, 1) != 1 || r.mbbits(12, 12) != 1 || r.mbbits(24, 24) != 1 || r.mbbits(46, 46) != 1 {
		return false
	}

	roll := float64(r.mbbits(3, 11)) * rollAngleStep
	if r.mbbits(2, 2) == 1 {
		roll -= rollSignOffset
	}

	if roll < -maxRollAngle || roll >= maxRollAngle {
		return false
	}

	gs := float64(r.mbbits(25, 34)) * commBSpeedStep
	tas := float64(r.mbbits(47, 56)) * commBSpeedStep

	if gs < minSpeed || gs > maxSpeed || tas < minSpeed || tas > maxSpeed {
		return false
	}

	if r.mbbits(35, 35) == 1 {
		rate := float64(r.mbbits(37, 45)) * trackRateStep
		if r.mbbits(36, 36) == 1 {
			rate -= trackRateSignOffset
		}

		if rate < -maxTrackRate || rate > maxTrackRate {
			return false
		}
	}

	return true
}

// validBDS60 reports whether the MB field is plausibly a heading and speed
// report (BDS 6,0).
func validBDS60(r *RawMessage) bool {
	baroValid, inertialValid := r.mbbits(35, 35) == 1, r.mbbits(46, 46) == 1
	if r.mbbits(1, 1) != 1 || r.mbbits(13, 13) != 1 || r.mbbits(24, 24) != 1 || (!baroValid && !inertialValid) {
		return false
	}

	ias := float64(r.mbbits(14, 23))
	if ias < minSpeed || ias > maxSpeed {
		return false
	}

	mach := float64(r.mbbits(25, 34)) * commBMachStep
	if mach < minMach || mach > maxMach {
		return false
	}

	if baroValid && !vertRatePlausible(r, 36, 37, 45) {
		return false
	}

	return !inertialValid || vertRatePlausible(r, 47, 48, 56)
}

// vertRatePlausible checks a signed vertical-rate subfield against the plausible
// range, given its sign bit and magnitude bit positions.
func vertRatePlausible(r *RawMessage, sign, n, z int) bool {
	rate := float64(r.mbbits(n, z)) * commBVertRateStep
	if r.mbbits(sign, sign) == 1 {
		rate -= vertRateSignOffset
	}

	return rate >= -maxVertRate && rate <= maxVertRate
}
