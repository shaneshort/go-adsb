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

// Meteorological report field scaling, from ICAO Doc 9871 Table A-2-68
// (BDS 4,4) and Table A-2-69 (BDS 4,5). Wind speed and static pressure use a
// unit LSB and need no scaling factor.
//
// Both static-air-temperature fields are two's complement (Table A-2-68/69
// Note 2). BDS 4,4 Table A-2-68 labels the temperature magnitude as 10 bits
// (MB 25-34) with a separate sign bit (MB 24) but marks its MSB as 64 C, which
// is inconsistent with a 0.25 C LSB over 10 bits; readsb resolves this by
// treating MB 24-34 as an 11-bit two's-complement value, which this package
// follows. BDS 4,5 Table A-2-69 is self-consistent as a 10-bit two's-
// complement value (MB 17-26).
const (
	windDirStep     = 180.0 / 256 // degrees per count (true wind direction)
	satStep         = 0.25        // degrees Celsius per count (static air temperature)
	humidityStep    = 100.0 / 64  // percent per count (relative humidity)
	radioHeightStep = 16          // feet per count (radio height)

	sat44SignOffset = 1024 // subtracted from the BDS 4,4 temperature magnitude when its sign bit is set
	sat45SignOffset = 512  // subtracted from the BDS 4,5 temperature magnitude when its sign bit is set
)

// MeteorologicalRoutineAirReport is a decoded Comm-B meteorological routine air
// report (BDS 4,4). Each optional field is a pointer whose nil value means the
// field's status bit was clear. The static air temperature has no status bit
// and is always present. FOMSource identifies the navigation source of the
// wind data.
type MeteorologicalRoutineAirReport struct {
	FOMSource             adsbtype.FOM     // figure of merit / source (always present)
	WindSpeed             *float64         // knots
	WindDirection         *float64         // degrees true, 0..360
	StaticAirTemperature  float64          // degrees Celsius (always present)
	AverageStaticPressure *float64         // hectopascals
	Turbulence            *adsbtype.Hazard // severity coding
	Humidity              *float64         // percent
}

// MeteorologicalHazardReport is a decoded Comm-B meteorological hazard report
// (BDS 4,5). Each field is a pointer whose nil value means the field's status
// bit was clear.
type MeteorologicalHazardReport struct {
	Turbulence            *adsbtype.Hazard
	WindShear             *adsbtype.Hazard
	Microburst            *adsbtype.Hazard
	Icing                 *adsbtype.Hazard
	WakeVortex            *adsbtype.Hazard
	StaticAirTemperature  *float64 // degrees Celsius
	AverageStaticPressure *float64 // hectopascals
	RadioHeight           *float64 // feet
}

// MeteorologicalRoutineAirReport decodes the MB field as a BDS 4,4
// meteorological routine air report. It returns an error wrapping
// ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21). The
// register identity is not verified; see the status bits for field validity.
func (m *Message) MeteorologicalRoutineAirReport() (*MeteorologicalRoutineAirReport, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	rep := &MeteorologicalRoutineAirReport{FOMSource: adsbtype.FOM(r.mbbits(1, 4))}

	// One status bit gates both wind speed and direction.
	if r.mbbits(5, 5) == 1 {
		ws := float64(r.mbbits(6, 14))
		rep.WindSpeed = &ws

		wd := float64(r.mbbits(15, 23)) * windDirStep
		rep.WindDirection = &wd
	}

	temp := safecast.MustConvert[int](r.mbbits(25, 34))
	if r.mbbits(24, 24) == 1 {
		temp -= sat44SignOffset
	}

	rep.StaticAirTemperature = float64(temp) * satStep

	if r.mbbits(35, 35) == 1 {
		p := float64(r.mbbits(36, 46))
		rep.AverageStaticPressure = &p
	}

	if r.mbbits(47, 47) == 1 {
		turb := adsbtype.Hazard(r.mbbits(48, 49))
		rep.Turbulence = &turb
	}

	if r.mbbits(50, 50) == 1 {
		h := float64(r.mbbits(51, 56)) * humidityStep
		rep.Humidity = &h
	}

	return rep, nil
}

// MeteorologicalHazardReport decodes the MB field as a BDS 4,5 meteorological
// hazard report. It returns an error wrapping ErrNotAvailable unless the
// message is a Comm-B reply (DF 20 or 21). The register identity is not
// verified; see the status bits for field validity.
func (m *Message) MeteorologicalHazardReport() (*MeteorologicalHazardReport, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	rep := &MeteorologicalHazardReport{
		Turbulence: hazardField(r, 1, 2, 3),
		WindShear:  hazardField(r, 4, 5, 6),
		Microburst: hazardField(r, 7, 8, 9),
		Icing:      hazardField(r, 10, 11, 12),
		WakeVortex: hazardField(r, 13, 14, 15),
	}

	if r.mbbits(16, 16) == 1 {
		temp := safecast.MustConvert[int](r.mbbits(18, 26))
		if r.mbbits(17, 17) == 1 {
			temp -= sat45SignOffset
		}

		t := float64(temp) * satStep
		rep.StaticAirTemperature = &t
	}

	if r.mbbits(27, 27) == 1 {
		p := float64(r.mbbits(28, 38))
		rep.AverageStaticPressure = &p
	}

	if r.mbbits(39, 39) == 1 {
		rh := float64(r.mbbits(40, 51)) * radioHeightStep
		rep.RadioHeight = &rh
	}

	return rep, nil
}

// hazardField decodes a status-gated two-bit hazard severity subfield. The
// status bit is at MB position st and the value spans MB bits msb..lsb. It
// returns nil when the status bit is clear.
func hazardField(r *RawMessage, st, msb, lsb int) *adsbtype.Hazard {
	if r.mbbits(st, st) != 1 {
		return nil
	}

	h := adsbtype.Hazard(r.mbbits(msb, lsb))

	return &h
}
