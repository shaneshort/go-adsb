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

package adsb_test

import (
	"errors"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
)

// BDS 4,1 next waypoint identifier: the nine-character waypoint name, trailing
// spaces trimmed. Vector built from the MB bit ranges of ICAO Doc 9871 Table
// A-2-65.
func TestNextWaypointIdentifier(t *testing.T) {
	id, err := mustVelMsg(t, "A000000096602C41041040000000").NextWaypointIdentifier()
	if err != nil {
		t.Fatalf("NextWaypointIdentifier: %v", err)
	}

	if id != "KLAX" {
		t.Errorf("NextWaypointIdentifier = %q, want %q", id, "KLAX")
	}

	// A clear status bit yields an empty identifier even when the character
	// bits carry residual data.
	id, err = mustVelMsg(t, "A00000007FFFFFFFFFFFFF000000").NextWaypointIdentifier()
	if err != nil {
		t.Fatalf("NextWaypointIdentifier: %v", err)
	}

	if id != "" {
		t.Errorf("NextWaypointIdentifier = %q, want empty", id)
	}
}

// BDS 4,2 with negative signed fields: waypoint latitude -45 deg and crossing
// altitude -4000 ft, exercising the latitude and altitude sign bits.
func TestNextWaypointPositionNegative(t *testing.T) {
	p, err := mustVelMsg(t, "A0000000F000080000FE0C000000").NextWaypointPosition()
	if err != nil {
		t.Fatalf("NextWaypointPosition: %v", err)
	}

	wantFloat(t, "Latitude", p.Latitude, -45, 0.001)
	wantInt(t, "CrossingAltitude", p.CrossingAltitude, -4000)
}

// BDS 4,3 with a negative bearing (-90 deg), exercising the bearing sign bit.
func TestNextWaypointInformationNegative(t *testing.T) {
	i, err := mustVelMsg(t, "A0000000E0089640FA0000000000").NextWaypointInformation()
	if err != nil {
		t.Fatalf("NextWaypointInformation: %v", err)
	}

	wantFloat(t, "BearingToWaypoint", i.BearingToWaypoint, -90, 0.001)
}

// BDS 4,2 next waypoint position: latitude +45 deg, longitude -90 deg, crossing
// altitude 10000 ft. Vector built from the MB bit ranges of ICAO Doc 9871 Table
// A-2-66 (two's complement per Note).
func TestNextWaypointPosition(t *testing.T) {
	p, err := mustVelMsg(t, "A000000090000E000084E2000000").NextWaypointPosition()
	if err != nil {
		t.Fatalf("NextWaypointPosition: %v", err)
	}

	wantFloat(t, "Latitude", p.Latitude, 45, 0.001)
	wantFloat(t, "Longitude", p.Longitude, -90, 0.001)
	wantInt(t, "CrossingAltitude", p.CrossingAltitude, 10000)
}

// With every status bit clear, the position fields decode to nil even when the
// value bits carry residual data.
func TestNextWaypointPositionNoData(t *testing.T) {
	p, err := mustVelMsg(t, "A00000007FFFF7FFFF7FFF000000").NextWaypointPosition()
	if err != nil {
		t.Fatalf("NextWaypointPosition: %v", err)
	}

	wantNil(t, "Latitude", p.Latitude == nil)
	wantNil(t, "Longitude", p.Longitude == nil)
	wantNil(t, "CrossingAltitude", p.CrossingAltitude == nil)
}

// BDS 4,3 next waypoint information: bearing +90 deg, time to go 30 minutes,
// distance to go 100 NM. Vector built from the MB bit ranges of ICAO Doc 9871
// Table A-2-67.
func TestNextWaypointInformation(t *testing.T) {
	i, err := mustVelMsg(t, "A0000000A0089640FA0000000000").NextWaypointInformation()
	if err != nil {
		t.Fatalf("NextWaypointInformation: %v", err)
	}

	wantFloat(t, "BearingToWaypoint", i.BearingToWaypoint, 90, 0.001)
	wantFloat(t, "TimeToGo", i.TimeToGo, 30, 0.001)
	wantFloat(t, "DistanceToGo", i.DistanceToGo, 100, 0.001)
}

// With every status bit clear, the information fields decode to nil even when
// the value bits carry residual data.
func TestNextWaypointInformationNoData(t *testing.T) {
	i, err := mustVelMsg(t, "A00000007FF7FFBFFFFFFF000000").NextWaypointInformation()
	if err != nil {
		t.Fatalf("NextWaypointInformation: %v", err)
	}

	wantNil(t, "BearingToWaypoint", i.BearingToWaypoint == nil)
	wantNil(t, "TimeToGo", i.TimeToGo == nil)
	wantNil(t, "DistanceToGo", i.DistanceToGo == nil)
}

// The next-waypoint registers require a DF20/21 reply.
func TestNextWaypointRejectNonReply(t *testing.T) {
	msg := mustVelMsg(t, "8D485020994409940838175B284F")

	_, err := msg.NextWaypointIdentifier()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("NextWaypointIdentifier err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.NextWaypointPosition()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("NextWaypointPosition err = %v, want ErrNotAvailable", err)
	}

	_, err = msg.NextWaypointInformation()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("NextWaypointInformation err = %v, want ErrNotAvailable", err)
	}
}
