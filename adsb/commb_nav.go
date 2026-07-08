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

// Next waypoint register field scaling, from ICAO Doc 9871 Tables A-2-66
// (BDS 4,2) and A-2-67 (BDS 4,3). The signed fields use two's complement per
// the table notes.
const (
	waypointLatLonStep = 90.0 / 131072 // degrees per count (latitude, longitude)
	waypointAltStep    = 8             // feet per count (crossing altitude)
	bearingStep        = 360.0 / 2048  // degrees per count (bearing to waypoint)
	timeToGoStep       = 0.1           // minutes per count
	distanceToGoStep   = 0.1           // nautical miles per count

	nextWaypointIDChars = 9 // characters in the next waypoint identifier
)

// mbTwosComplement reads the magnitude field at MB bits n through z together
// with the sign bit at MB position sign, returning the signed value as a two's
// complement integer (the sign bit carries the negative weight of the field).
func mbTwosComplement(r *RawMessage, sign, n, z int) int {
	v := safecast.MustConvert[int](r.mbbits(n, z))
	if r.mbbits(sign, sign) == 1 {
		v -= 1 << (z - n + 1)
	}

	return v
}

// NextWaypointPosition is a decoded Comm-B next waypoint position report
// (BDS 4,2). Each field is a pointer whose nil value means the field's status
// bit was clear.
type NextWaypointPosition struct {
	Latitude         *float64 // degrees, -180..180
	Longitude        *float64 // degrees, -180..180
	CrossingAltitude *int     // feet
}

// NextWaypointInformation is a decoded Comm-B next waypoint information report
// (BDS 4,3). Each field is a pointer whose nil value means the field's status
// bit was clear.
type NextWaypointInformation struct {
	BearingToWaypoint *float64 // degrees from true north, -180..180
	TimeToGo          *float64 // minutes
	DistanceToGo      *float64 // nautical miles
}

// NextWaypointIdentifier decodes the MB field as a BDS 4,1 next waypoint
// identifier report and returns the waypoint name (up to nine IA-5 characters,
// trailing spaces trimmed), or an empty string when the status bit is clear. It
// returns an error wrapping ErrNotAvailable unless the message is a Comm-B
// reply (DF 20 or 21). The register identity is not verified.
func (m *Message) NextWaypointIdentifier() (string, error) {
	r, err := m.commBRaw()
	if err != nil {
		return "", err
	}

	if r.mbbits(1, 1) != 1 {
		return "", nil
	}

	return mbString(r, 2, nextWaypointIDChars), nil
}

// NextWaypointPosition decodes the MB field as a BDS 4,2 next waypoint position
// report (ICAO Doc 9871 Table A-2-66). It returns an error wrapping
// ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21). The
// register identity is not verified.
func (m *Message) NextWaypointPosition() (*NextWaypointPosition, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	p := new(NextWaypointPosition)

	if r.mbbits(1, 1) == 1 {
		lat := float64(mbTwosComplement(r, 2, 3, 20)) * waypointLatLonStep
		p.Latitude = &lat
	}

	if r.mbbits(21, 21) == 1 {
		lon := float64(mbTwosComplement(r, 22, 23, 40)) * waypointLatLonStep
		p.Longitude = &lon
	}

	if r.mbbits(41, 41) == 1 {
		alt := mbTwosComplement(r, 42, 43, 56) * waypointAltStep
		p.CrossingAltitude = &alt
	}

	return p, nil
}

// NextWaypointInformation decodes the MB field as a BDS 4,3 next waypoint
// information report (ICAO Doc 9871 Table A-2-67). It returns an error wrapping
// ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21). The
// register identity is not verified.
func (m *Message) NextWaypointInformation() (*NextWaypointInformation, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	i := new(NextWaypointInformation)

	if r.mbbits(1, 1) == 1 {
		brg := float64(mbTwosComplement(r, 2, 3, 12)) * bearingStep
		i.BearingToWaypoint = &brg
	}

	if r.mbbits(13, 13) == 1 {
		ttg := float64(r.mbbits(14, 25)) * timeToGoStep
		i.TimeToGo = &ttg
	}

	if r.mbbits(26, 26) == 1 {
		dtg := float64(r.mbbits(27, 42)) * distanceToGoStep
		i.DistanceToGo = &dtg
	}

	return i, nil
}
