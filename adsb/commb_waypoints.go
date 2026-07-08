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

// Waypoint register field layout, shared by BDS 5,4 to 5,6 (ICAO Doc 9871
// Tables A-2-84 to A-2-86). A single status bit gates the whole report.
const (
	waypointETAStep = 60.0 / 512 // minutes per count (estimated time of arrival, time to go)
	waypointFLStep  = 10         // flight level per count
	waypointIDChars = 5          // characters in the waypoint identity

	// waypointTimeSentinel is the all-ones estimated-time / time-to-go value
	// that indicates the waypoint is one hour or more away.
	waypointTimeSentinel = 0x1FF // nine bits set
	waypointHour         = 60    // minutes reported for the one-hour-or-more sentinel
)

// Waypoint is a decoded Comm-B waypoint report (BDS 5,4, 5,5 or 5,6). A single
// status bit gates the whole report: when Valid is false the other fields are
// unset. Identity is the waypoint identifier, left-padded with '0' characters
// when the identifier has fewer than five characters.
//
// The estimated time of arrival and time to go are in minutes. When the field
// holds its all-ones sentinel the waypoint is one hour or more away: the
// corresponding OneHourOrMore flag is set and the time is reported as 60.
type Waypoint struct {
	Valid                               bool    // single status bit
	Identity                            string  // waypoint identifier
	EstimatedTimeOfArrival              float64 // minutes
	EstimatedTimeOfArrivalOneHourOrMore bool
	EstimatedFlightLevel                int     // flight level
	TimeToGo                            float64 // minutes, direct route
	TimeToGoOneHourOrMore               bool
}

// decodeWaypoint decodes the shared BDS 5,4 to 5,6 waypoint format from the MB
// field. When the status bit is clear it returns an invalid (zero) report.
func decodeWaypoint(r *RawMessage) *Waypoint {
	if r.mbbits(1, 1) != 1 {
		return &Waypoint{}
	}

	eta, etaMax := waypointTime(r.mbbits(32, 40))
	ttg, ttgMax := waypointTime(r.mbbits(47, 55))

	return &Waypoint{
		Valid:                               true,
		Identity:                            mbString(r, 2, waypointIDChars),
		EstimatedTimeOfArrival:              eta,
		EstimatedTimeOfArrivalOneHourOrMore: etaMax,
		EstimatedFlightLevel:                safecast.MustConvert[int](r.mbbits(41, 46)) * waypointFLStep,
		TimeToGo:                            ttg,
		TimeToGoOneHourOrMore:               ttgMax,
	}
}

// waypointTime converts a raw nine-bit waypoint time field to minutes. Its
// all-ones sentinel indicates one hour or more, reported as 60 minutes with the
// returned flag set.
func waypointTime(raw uint64) (float64, bool) {
	if raw == waypointTimeSentinel {
		return waypointHour, true
	}

	return float64(raw) * waypointETAStep, false
}

// Waypoint1 decodes the MB field as a BDS 5,4 next waypoint report (ICAO Doc
// 9871 Table A-2-84). It returns an error wrapping ErrNotAvailable unless the
// message is a Comm-B reply (DF 20 or 21). The register identity is not
// verified.
func (m *Message) Waypoint1() (*Waypoint, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeWaypoint(r), nil
}

// Waypoint2 decodes the MB field as a BDS 5,5 next-plus-one waypoint report
// (ICAO Doc 9871 Table A-2-85). It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified.
func (m *Message) Waypoint2() (*Waypoint, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeWaypoint(r), nil
}

// Waypoint3 decodes the MB field as a BDS 5,6 next-plus-two waypoint report
// (ICAO Doc 9871 Table A-2-86). It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified.
func (m *Message) Waypoint3() (*Waypoint, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeWaypoint(r), nil
}
