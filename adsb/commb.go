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

// Comm-B (DF20/21) BDS register field scaling. Bit ranges are from ICAO
// Annex 10 Vol IV Table 3-10; the LSB scaling and sign handling follow ICAO
// Doc 9871 as implemented by dump1090/readsb.
const (
	commBAltStep      = 16          // feet per count (selected altitude)
	commBBaroStep     = 0.1         // millibars per count (pressure setting)
	commBSpeedStep    = 2           // knots per count (ground / true airspeed)
	commBVertRateStep = 32          // feet/minute per count
	commBMachStep     = 2.048 / 512 // Mach per count
	rollAngleStep     = 45.0 / 256  // degrees per count
	trackHeadingStep  = 90.0 / 512  // degrees per count (track, heading)
	trackRateStep     = 8.0 / 256   // degrees/second per count

	rollSignOffset      = 90    // degrees, subtracted when the roll sign bit is set
	headingSignOffset   = 180   // degrees, added when the track/heading sign bit is set
	trackRateSignOffset = 16    // degrees/second, subtracted when the sign bit is set
	vertRateSignOffset  = 16384 // feet/minute, subtracted when the sign bit is set
)

// SelectedVerticalIntention is a decoded Comm-B track selected vertical
// intention report (BDS 4,0). Selected altitudes and the pressure setting are
// pointers: a nil pointer means the field's status bit was clear. The VNAV,
// altitude-hold and approach flags are meaningful only when ModeBitsValid is
// true.
type SelectedVerticalIntention struct {
	MCPSelectedAltitude       *int     // feet (MCP/FCU)
	FMSSelectedAltitude       *int     // feet (FMS)
	BarometricPressureSetting *float64 // millibars

	ModeBitsValid bool
	VNAVMode      bool
	AltitudeHold  bool
	ApproachMode  bool

	TargetAltitudeSourceValid bool
	TargetAltitudeSource      uint8 // 0 unknown, 1 aircraft altitude, 2 MCP/FCU, 3 FMS
}

// TrackAndTurn is a decoded Comm-B track and turn report (BDS 5,0). Each field
// is a pointer: a nil pointer means the field's status bit was clear.
type TrackAndTurn struct {
	RollAngle      *float64 // degrees, positive = right wing down
	TrueTrack      *float64 // degrees clockwise from true north, 0..360
	GroundSpeed    *float64 // knots
	TrackAngleRate *float64 // degrees/second, positive = right turn
	TrueAirspeed   *float64 // knots
}

// HeadingAndSpeed is a decoded Comm-B heading and speed report (BDS 6,0). Each
// field is a pointer: a nil pointer means the field's status bit was clear.
type HeadingAndSpeed struct {
	MagneticHeading          *float64 // degrees clockwise from magnetic north, 0..360
	IndicatedAirspeed        *float64 // knots
	Mach                     *float64 // ratio
	BarometricAltitudeRate   *int     // feet/minute, positive up
	InertialVerticalVelocity *int     // feet/minute, positive up
}

// mbbits returns bits n through z of the 56-bit Comm-B message (MB) field,
// which occupies frame bits 33-88. The first bit is numbered 1.
func (r *RawMessage) mbbits(n int, z int) uint64 {
	return r.Bits(n+32, z+32)
}

// commBRaw returns the underlying RawMessage if the message is a Comm-B reply
// (DF 20 or 21), which carries the 56-bit MB field.
func (m *Message) commBRaw() (*RawMessage, error) {
	df, err := m.raw.DF()
	if err != nil {
		return nil, newError(err, "error retrieving Comm-B")
	}

	if df != 20 && df != 21 {
		return nil, newErrorf(ErrNotAvailable, "Comm-B not available in format %d", df)
	}

	return m.raw, nil
}

// SelectedVerticalIntention decodes the MB field as a BDS 4,0 selected
// vertical intention report. It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified; see the status bits for field validity.
func (m *Message) SelectedVerticalIntention() (*SelectedVerticalIntention, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	svi := new(SelectedVerticalIntention)

	if r.mbbits(1, 1) == 1 {
		alt := safecast.MustConvert[int](r.mbbits(2, 13)) * commBAltStep
		svi.MCPSelectedAltitude = &alt
	}

	if r.mbbits(14, 14) == 1 {
		alt := safecast.MustConvert[int](r.mbbits(15, 26)) * commBAltStep
		svi.FMSSelectedAltitude = &alt
	}

	if r.mbbits(27, 27) == 1 {
		mb := baroPressureBase + float64(r.mbbits(28, 39))*commBBaroStep
		svi.BarometricPressureSetting = &mb
	}

	if r.mbbits(48, 48) == 1 {
		svi.ModeBitsValid = true
		svi.VNAVMode = r.mbbits(49, 49) == 1
		svi.AltitudeHold = r.mbbits(50, 50) == 1
		svi.ApproachMode = r.mbbits(51, 51) == 1
	}

	if r.mbbits(54, 54) == 1 {
		svi.TargetAltitudeSourceValid = true
		svi.TargetAltitudeSource = asU8(r.mbbits(55, 56))
	}

	return svi, nil
}

// TrackAndTurn decodes the MB field as a BDS 5,0 track and turn report. It
// returns an error wrapping ErrNotAvailable unless the message is a Comm-B
// reply (DF 20 or 21). The register identity is not verified; see the status
// bits for field validity.
func (m *Message) TrackAndTurn() (*TrackAndTurn, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	tt := new(TrackAndTurn)

	if r.mbbits(1, 1) == 1 {
		roll := float64(r.mbbits(3, 11)) * rollAngleStep
		if r.mbbits(2, 2) == 1 {
			roll -= rollSignOffset
		}

		tt.RollAngle = &roll
	}

	if r.mbbits(12, 12) == 1 {
		track := float64(r.mbbits(14, 23)) * trackHeadingStep
		if r.mbbits(13, 13) == 1 {
			track += headingSignOffset
		}

		tt.TrueTrack = &track
	}

	if r.mbbits(24, 24) == 1 {
		gs := float64(r.mbbits(25, 34)) * commBSpeedStep
		tt.GroundSpeed = &gs
	}

	if r.mbbits(35, 35) == 1 {
		rate := float64(r.mbbits(37, 45)) * trackRateStep
		if r.mbbits(36, 36) == 1 {
			rate -= trackRateSignOffset
		}

		tt.TrackAngleRate = &rate
	}

	if r.mbbits(46, 46) == 1 {
		tas := float64(r.mbbits(47, 56)) * commBSpeedStep
		tt.TrueAirspeed = &tas
	}

	return tt, nil
}

// HeadingAndSpeed decodes the MB field as a BDS 6,0 heading and speed report.
// It returns an error wrapping ErrNotAvailable unless the message is a Comm-B
// reply (DF 20 or 21). The register identity is not verified; see the status
// bits for field validity.
func (m *Message) HeadingAndSpeed() (*HeadingAndSpeed, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	hs := new(HeadingAndSpeed)

	if r.mbbits(1, 1) == 1 {
		hdg := float64(r.mbbits(3, 12)) * trackHeadingStep
		if r.mbbits(2, 2) == 1 {
			hdg += headingSignOffset
		}

		hs.MagneticHeading = &hdg
	}

	if r.mbbits(13, 13) == 1 {
		ias := float64(r.mbbits(14, 23))
		hs.IndicatedAirspeed = &ias
	}

	if r.mbbits(24, 24) == 1 {
		mach := float64(r.mbbits(25, 34)) * commBMachStep
		hs.Mach = &mach
	}

	if r.mbbits(35, 35) == 1 {
		rate := safecast.MustConvert[int](r.mbbits(37, 45)) * commBVertRateStep
		if r.mbbits(36, 36) == 1 {
			rate -= vertRateSignOffset
		}

		hs.BarometricAltitudeRate = &rate
	}

	if r.mbbits(46, 46) == 1 {
		ivv := safecast.MustConvert[int](r.mbbits(48, 56)) * commBVertRateStep
		if r.mbbits(47, 47) == 1 {
			ivv -= vertRateSignOffset
		}

		hs.InertialVerticalVelocity = &ivv
	}

	return hs, nil
}
