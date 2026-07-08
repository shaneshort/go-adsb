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

// Position and state register field scaling, from ICAO Doc 9871 Tables A-2-81
// (BDS 5,1) and A-2-83 (BDS 5,3). The signed fields use two's complement per
// the table notes.
const (
	coarsePosStep = 360.0 / 1048576 // degrees per count (coarse latitude, longitude)
	coarseAltStep = 8               // feet per count (coarse pressure altitude)

	// Coarse position valid ranges (ICAO Doc 9871 Table A-2-81). Latitude is
	// coded in a +/-180 degree field but is only valid over +/-90 degrees, and
	// the pressure altitude has a narrower valid range than its field allows.
	coarseLatMax = 90     // degrees
	coarseAltMin = -1000  // feet
	coarseAltMax = 126752 // feet

	arsvHeadingStep = 90.0 / 512  // degrees per count (magnetic heading)
	arsvMachStep    = 2.048 / 256 // Mach per count
	arsvTASStep     = 0.5         // knots per count (true airspeed)
	arsvAltRateStep = 64          // feet/minute per count (altitude rate)
)

// PositionReportCoarse is a decoded Comm-B coarse position report (BDS 5,1).
// Latitude and Longitude are pointers gated by the single status bit; a nil
// pointer means the report was invalid or out of range. PressureAltitude has no
// status bit and is nil when its field is zeroed (reported invalid) or out of
// range. Latitude is valid only over -90..90 degrees.
type PositionReportCoarse struct {
	Latitude         *float64 // degrees, -90..90
	Longitude        *float64 // degrees, -180..180
	PressureAltitude *int     // feet, -1000..126752
}

// AirReferencedStateVector is a decoded Comm-B air-referenced state vector
// report (BDS 5,3). Each field is a pointer whose nil value means the field's
// status bit was clear.
type AirReferencedStateVector struct {
	MagneticHeading   *float64 // degrees, -180..180
	IndicatedAirspeed *float64 // knots
	Mach              *float64 // ratio
	TrueAirspeed      *float64 // knots
	AltitudeRate      *int     // feet/minute, positive up
}

// QuasiStaticParameterMonitoring is a decoded Comm-B quasi-static parameter
// monitoring report (BDS 5,F). Each field is a two-bit change counter: 0 means
// no valid data, and the value cycles through 1, 2 and 3, each step signalling
// a change in the monitored parameter (held in another register).
type QuasiStaticParameterMonitoring struct {
	MCPSelectedAltitude       uint8 // change monitor for BDS 4,0 MCP/FCU selected altitude
	NextWaypoint              uint8 // change monitor for BDS 4,1 to 4,3
	FMSVerticalMode           uint8 // change monitor for BDS 4,0 mode bits
	VHFChannel                uint8 // change monitor for BDS 4,8
	MeteorologicalHazards     uint8 // change monitor for BDS 4,5 hazards
	FMSSelectedAltitude       uint8 // change monitor for BDS 4,0 FMS selected altitude
	BarometricPressureSetting uint8 // change monitor for BDS 4,0 pressure setting
}

// PositionReportCoarse decodes the MB field as a BDS 5,1 coarse position report
// (ICAO Doc 9871 Table A-2-81). It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified.
func (m *Message) PositionReportCoarse() (*PositionReportCoarse, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	p := new(PositionReportCoarse)

	// A single status bit gates both latitude and longitude. Latitude is valid
	// only over the +/-90 degree range even though it uses a +/-180 field.
	if r.mbbits(1, 1) == 1 {
		if lat := float64(mbTwosComplement(r, 2, 3, 21)) * coarsePosStep; lat >= -coarseLatMax && lat <= coarseLatMax {
			p.Latitude = &lat
		}

		lon := float64(mbTwosComplement(r, 22, 23, 41)) * coarsePosStep
		p.Longitude = &lon
	}

	// The pressure altitude has no status bit; an all-zero field is invalid, as
	// is any value outside the register's valid range.
	if r.mbbits(42, 56) != 0 {
		if alt := mbTwosComplement(r, 42, 43, 56) * coarseAltStep; alt >= coarseAltMin && alt <= coarseAltMax {
			p.PressureAltitude = &alt
		}
	}

	return p, nil
}

// AirReferencedStateVector decodes the MB field as a BDS 5,3 air-referenced
// state vector report (ICAO Doc 9871 Table A-2-83). It returns an error
// wrapping ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21).
// The register identity is not verified.
func (m *Message) AirReferencedStateVector() (*AirReferencedStateVector, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	v := new(AirReferencedStateVector)

	if r.mbbits(1, 1) == 1 {
		hdg := float64(mbTwosComplement(r, 2, 3, 12)) * arsvHeadingStep
		v.MagneticHeading = &hdg
	}

	if r.mbbits(13, 13) == 1 {
		ias := float64(r.mbbits(14, 23))
		v.IndicatedAirspeed = &ias
	}

	if r.mbbits(24, 24) == 1 {
		mach := float64(r.mbbits(25, 33)) * arsvMachStep
		v.Mach = &mach
	}

	if r.mbbits(34, 34) == 1 {
		tas := float64(r.mbbits(35, 46)) * arsvTASStep
		v.TrueAirspeed = &tas
	}

	if r.mbbits(47, 47) == 1 {
		rate := mbTwosComplement(r, 48, 49, 56) * arsvAltRateStep
		v.AltitudeRate = &rate
	}

	return v, nil
}

// QuasiStaticParameterMonitoring decodes the MB field as a BDS 5,F quasi-static
// parameter monitoring report (ICAO Doc 9871 Table A-2-95). It returns an error
// wrapping ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21).
// The register identity is not verified.
func (m *Message) QuasiStaticParameterMonitoring() (*QuasiStaticParameterMonitoring, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return &QuasiStaticParameterMonitoring{
		MCPSelectedAltitude:       asU8(r.mbbits(1, 2)),
		NextWaypoint:              asU8(r.mbbits(13, 14)),
		FMSVerticalMode:           asU8(r.mbbits(17, 18)),
		VHFChannel:                asU8(r.mbbits(19, 20)),
		MeteorologicalHazards:     asU8(r.mbbits(21, 22)),
		FMSSelectedAltitude:       asU8(r.mbbits(23, 24)),
		BarometricPressureSetting: asU8(r.mbbits(25, 26)),
	}, nil
}
