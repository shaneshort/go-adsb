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

import (
	"github.com/ccoveille/go-safecast/v2"
	"kreklow.us/go/go-adsb/adsbtype"
)

// TIS-B coarse position field scaling, from DO-260B §2.2.17.3.5.
const (
	tisbCoarseCF      = 3  // DF18 control field value for a coarse-format TIS-B message
	tisbCoarseCPRBits = 12 // width of the coarse CPR latitude and longitude fields

	groundTrackStep = 360.0 / 32 // degrees per count (coarse ground track)
	groundSpeedStep = 32         // knots per ground speed range step above the base
	groundSpeedBase = 16         // knots, lower bound of ground speed code 2
)

// TISBCoarsePosition is a decoded TIS-B coarse airborne position message
// (DF18, control field 3), following the ME bit assignments of DO-260B Figure
// 2-30. This message format is not related to any ADS-B format.
//
// IMF is the ICAO/Mode A flag: false means the address is a 24-bit ICAO
// address, true means it is a Mode A code with a TIS-B track number.
// ServiceVolumeID identifies the TIS-B ground station that delivered the
// surveillance data. PressureAltitude is nil when no altitude is reported.
// GroundTrack is nil unless its status bit reports it valid. GroundSpeed is the
// lower bound of a coarse range in knots (DO-260B Table 2-107) and is nil when
// no ground speed information is available. CPR is the 12-bit coarse compact
// position report, decoded with the position CPR functions.
type TISBCoarsePosition struct {
	IMF                bool
	SurveillanceStatus adsbtype.SSS
	ServiceVolumeID    uint8
	PressureAltitude   *int64   // feet
	GroundTrack        *float64 // degrees clockwise from true north
	GroundSpeed        *float64 // knots
	CPR                *CPR
}

// TISBCoarsePosition decodes the ME field as a TIS-B coarse airborne position
// message (DO-260B Figure 2-30). It returns an error wrapping ErrNotAvailable
// unless the message is a DF18 reply with control field 3 (coarse-format
// TIS-B).
func (m *Message) TISBCoarsePosition() (*TISBCoarsePosition, error) {
	cf, err := m.raw.CF()
	if err != nil {
		return nil, newError(err, "error retrieving TIS-B coarse position")
	}

	if cf != tisbCoarseCF {
		return nil, newErrorf(ErrNotAvailable,
			"TIS-B coarse position not available in control field %d", cf)
	}

	r := m.raw

	p := &TISBCoarsePosition{
		IMF:                r.esbits(1, 1) == 1,
		SurveillanceStatus: adsbtype.SSS(r.esbits(2, 3)),
		ServiceVolumeID:    asU8(r.esbits(4, 7)),
		CPR: &CPR{
			Nb:  tisbCoarseCPRBits,
			F:   asU8(r.esbits(32, 32)),
			Lat: safecast.MustConvert[uint32](r.esbits(33, 44)),
			Lon: safecast.MustConvert[uint32](r.esbits(45, 56)),
		},
	}

	alt, aerr := decodeESAlt(r.esbits(8, 19))
	if aerr == nil {
		p.PressureAltitude = &alt
	}

	if r.esbits(20, 20) == 1 {
		track := float64(r.esbits(21, 25)) * groundTrackStep
		p.GroundTrack = &track
	}

	if gs, ok := tisbGroundSpeed(r.esbits(26, 31)); ok {
		p.GroundSpeed = &gs
	}

	return p, nil
}

// tisbGroundSpeed converts the six-bit coarse ground speed code (DO-260B Table
// 2-107) to the lower bound of its range in knots. Code 0 reports no ground
// speed information (ok is false); code 1 is "less than 16 knots" (lower bound
// zero); each higher code adds a 32-knot step above a 16-knot base.
func tisbGroundSpeed(code uint64) (float64, bool) {
	switch code {
	case 0:
		return 0, false
	case 1:
		return 0, true
	default:
		return float64(groundSpeedBase) + float64(groundSpeedStep)*float64(code-2), true
	}
}
