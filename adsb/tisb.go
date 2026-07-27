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

// DF18 control field values that carry an ICAO/Mode A flag (IMF) in their ME
// field: the fine and coarse TIS-B formats and the ADS-R rebroadcast format.
const (
	tisbFineCF        = 2 // fine-format TIS-B message
	tisbCoarseCF      = 3 // coarse-format TIS-B message
	tisbFineNonICAOCF = 5 // fine-format TIS-B message with a non-ICAO address
	adsrCF            = 6 // ADS-R rebroadcast message
)

// velocityType is the extended squitter type code for an airborne velocity
// message; no shared constant exists elsewhere in the package.
const velocityType = 19

// TIS-B coarse position field scaling, from DO-260B §2.2.17.3.5.
const (
	tisbCoarseCPRBits = 12 // width of the coarse CPR latitude and longitude fields

	groundTrackStep = 360.0 / 32 // degrees per count (coarse ground track)
	groundSpeedStep = 32         // knots per ground speed range step above the base
	groundSpeedBase = 16         // knots, lower bound of ground speed code 2
)

// Field errors for message fields that are not carried by every downlink
// format. They are pre-built so that returning one does not allocate; see
// notAvailable.
var (
	errIMFNotAvailable        = notAvailable("IMF")
	errTISBCoarseNotAvailable = notAvailable("TIS-B coarse position")
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

// imfMEBit returns the ME bit position of the ICAO/Mode A flag (IMF) subfield
// for a fine-format TIS-B or ADS-R message of the given extended squitter type
// code, and whether that type code defines an IMF subfield (DO-260B §2.2.17
// and §2.2.18).
func imfMEBit(tc uint64) (int, bool) {
	switch {
	case tc >= surfacePosTypeLo && tc <= surfacePosTypeHi: // surface position
		return 21, true
	case (tc >= airPosTypeLo && tc <= airPosTypeHi) ||
		(tc >= gnssPosTypeLo && tc <= gnssPosTypeHi): // airborne position
		return 8, true
	case tc == velocityType: // airborne velocity
		return 9, true
	case tc == aircraftStatusType: // aircraft status (emergency/priority)
		return 56, true
	case tc == targetStateType: // target state and status
		return 51, true
	case tc == opStatusType: // aircraft operational status
		return 56, true
	default:
		return 0, false
	}
}

// IMF returns the ICAO/Mode A flag (IMF) of a TIS-B or ADS-R message, reporting
// how the AA address field is to be interpreted: false means the AA field holds
// a 24-bit ICAO address, true means it holds a non-ICAO address (a Mode A code
// with a track file number).
//
// The IMF subfield is present only in TIS-B and ADS-R extended squitters (DF18
// with control field 2, 3, 5 or 6). Its ME bit position is type-code specific:
// a coarse TIS-B position (control field 3) carries it in ME bit 1, while the
// fine formats redefine a type-code-specific ME bit (DO-260B §2.2.17 and
// §2.2.18). It returns an error wrapping ErrNotAvailable for any other format
// and for message types that do not define an IMF subfield.
func (m *Message) IMF() (bool, error) {
	cf, err := m.raw.CF()
	if err != nil {
		return false, newError(err, "error retrieving IMF")
	}

	switch cf {
	case tisbCoarseCF:
		return m.raw.esbits(1, 1) == 1, nil
	case tisbFineCF, tisbFineNonICAOCF, adsrCF:
		tc, err := m.raw.ESType()
		if err != nil {
			return false, newError(err, "error retrieving IMF")
		}

		bit, ok := imfMEBit(tc)
		if !ok {
			return false, errIMFNotAvailable
		}

		return m.raw.esbits(bit, bit) == 1, nil
	default:
		return false, errIMFNotAvailable
	}
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
		return nil, errTISBCoarseNotAvailable
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
