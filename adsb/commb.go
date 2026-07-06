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

// DataLinkCapability is a decoded Comm-B data link capability report
// (BDS 1,0). The fields follow the MB bit assignments of ICAO Annex 10 Vol IV
// Table 3-6.
//
// ACAS capability spans two non-contiguous subfields (MB bits 16 and 37-40):
// ACASCapability holds bit 16 and ACASAdditionalCapability holds bits 37-40 as
// a raw value; Annex 10 does not further subdivide the latter (its coding is
// defined in ICAO Doc 9871). The uplink and downlink ELM capability fields are
// likewise retained as raw coded values.
type DataLinkCapability struct {
	ContinuationFlag              bool   // MB bit 9: further capability reports follow
	OverlayCommandCapability      bool   // MB bit 15
	ACASCapability                bool   // MB bit 16
	ModeSSubnetworkVersion        uint8  // MB bits 17-23
	TransponderEnhancedProtocol   bool   // MB bit 24: 1 = Level 5, 0 = Level 2-4
	SpecificServicesCapability    bool   // MB bit 25: any GICB/MSP service register loaded
	UplinkELMCapability           uint8  // MB bits 26-28, raw coded throughput
	DownlinkELMCapability         uint8  // MB bits 29-32, raw coded throughput
	AircraftIdentificationCapable bool   // MB bit 33
	SquitterCapability            bool   // MB bit 34: squitter capability subfield (SCS)
	SurveillanceIdentifierCode    bool   // MB bit 35: SI code capability (SIC)
	CommonUsageGICBCapability     bool   // MB bit 36: common-usage GICB report (BDS 1,7) changed
	ACASAdditionalCapability      uint8  // MB bits 37-40, raw
	DTESubaddressStatus           uint16 // MB bits 41-56: status of DTE sub-addresses 0-15
}

// DataLinkCapability decodes the MB field as a BDS 1,0 data link capability
// report. It returns an error wrapping ErrNotAvailable unless the message is a
// Comm-B reply (DF 20 or 21). The register identity is not verified beyond the
// downlink format; use InferBDS to check that the MB field self-identifies as
// BDS 1,0 (its first eight bits equal 0x10).
func (m *Message) DataLinkCapability() (*DataLinkCapability, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return &DataLinkCapability{
		ContinuationFlag:              r.mbbits(9, 9) == 1,
		OverlayCommandCapability:      r.mbbits(15, 15) == 1,
		ACASCapability:                r.mbbits(16, 16) == 1,
		ModeSSubnetworkVersion:        asU8(r.mbbits(17, 23)),
		TransponderEnhancedProtocol:   r.mbbits(24, 24) == 1,
		SpecificServicesCapability:    r.mbbits(25, 25) == 1,
		UplinkELMCapability:           asU8(r.mbbits(26, 28)),
		DownlinkELMCapability:         asU8(r.mbbits(29, 32)),
		AircraftIdentificationCapable: r.mbbits(33, 33) == 1,
		SquitterCapability:            r.mbbits(34, 34) == 1,
		SurveillanceIdentifierCode:    r.mbbits(35, 35) == 1,
		CommonUsageGICBCapability:     r.mbbits(36, 36) == 1,
		ACASAdditionalCapability:      asU8(r.mbbits(37, 40)),
		DTESubaddressStatus:           asU16(r.mbbits(41, 56)),
	}, nil
}

// gicbRegisters maps each assigned BDS 1,7 status bit to the GICB register it
// reports as available, per ICAO Doc 9871 Table A-2-23. MB bits 25-26
// (reserved for aircraft capability) and 30-56 (reserved) carry no register
// mapping and are omitted.
var gicbRegisters = []struct {
	bit int
	bds adsbtype.BDS
}{
	{1, adsbtype.BDS05},
	{2, adsbtype.BDS06},
	{3, adsbtype.BDS07},
	{4, adsbtype.BDS08},
	{5, adsbtype.BDS09},
	{6, adsbtype.BDS0A},
	{7, adsbtype.BDS20},
	{8, adsbtype.BDS21},
	{9, adsbtype.BDS40},
	{10, adsbtype.BDS41},
	{11, adsbtype.BDS42},
	{12, adsbtype.BDS43},
	{13, adsbtype.BDS44},
	{14, adsbtype.BDS45},
	{15, adsbtype.BDS48},
	{16, adsbtype.BDS50},
	{17, adsbtype.BDS51},
	{18, adsbtype.BDS52},
	{19, adsbtype.BDS53},
	{20, adsbtype.BDS54},
	{21, adsbtype.BDS55},
	{22, adsbtype.BDS56},
	{23, adsbtype.BDS5F},
	{24, adsbtype.BDS60},
	{27, adsbtype.BDSE1},
	{28, adsbtype.BDSE2},
	{29, adsbtype.BDSF1},
}

// CommonUsageGICB decodes the MB field as a BDS 1,7 common usage GICB
// capability report and returns the GICB registers currently reported as
// available (status bit set), in ascending register order. It returns an
// error wrapping ErrNotAvailable unless the message is a Comm-B reply
// (DF 20 or 21). The register identity is not verified; use InferBDS to check
// that the MB field is plausibly a BDS 1,7 report.
func (m *Message) CommonUsageGICB() ([]adsbtype.BDS, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	var avail []adsbtype.BDS

	for _, g := range gicbRegisters {
		if r.mbbits(g.bit, g.bit) == 1 {
			avail = append(avail, g.bds)
		}
	}

	return avail, nil
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
