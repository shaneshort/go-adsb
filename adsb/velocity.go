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
	"math"

	"github.com/ccoveille/go-safecast/v2"
	"kreklow.us/go/go-adsb/adsbtype"
)

// BDS 0,9 velocity subtypes. Subtypes 1 and 3 report at 1-knot resolution;
// the supersonic subtypes 2 and 4 report at 4-knot resolution.
const (
	velGroundSubsonic   uint8 = 1 // ground speed and track
	velGroundSupersonic uint8 = 2 // ground speed and track, supersonic
	velAirSubsonic      uint8 = 3 // airspeed and heading
	velAirSupersonic    uint8 = 4 // airspeed and heading, supersonic
)

// BDS 0,9 field scale factors and sentinels.
const (
	velSubsonicScale   = 1    // knots per count (subsonic subtypes)
	velSupersonicScale = 4    // knots per count (supersonic subtypes)
	verticalRateStep   = 64   // feet/minute per count
	gnssBaroStep       = 25   // feet per count
	headingSteps       = 1024 // heading counts per full 360-degree revolution
	gnssBaroNoData     = 127  // all-ones 7-bit magnitude sentinel ("no data")
)

// Field errors for message fields that are not carried by every downlink
// format. They are pre-built so that returning one does not allocate; see
// notAvailable.
var (
	errVelocityNotAvailable = notAvailable("velocity")
)

// Velocity is a decoded ADS-B airborne velocity message (extended squitter
// type code 19, BDS 0,9). The Subtype selects which horizontal quantity is
// reported: subtypes 1 and 2 report ground speed and track; subtypes 3 and 4
// report airspeed and heading. Subtypes 2 and 4 are the supersonic variants.
//
// Optional quantities are pointers: a nil pointer means the field carried the
// register's "no data" sentinel and no value is reported.
type Velocity struct {
	Subtype uint8 // 1..4

	// Ground-speed subtypes (1, 2).
	GroundSpeed *float64 // knots
	Track       *float64 // degrees clockwise from true north, 0..360

	// Airspeed subtypes (3, 4).
	Airspeed     *float64     // knots
	AirspeedType adsbtype.AST // IAS or TAS; meaningful when Airspeed != nil
	Heading      *float64     // degrees clockwise from magnetic north, 0..360

	// Vertical rate — reported by all subtypes.
	VerticalRate       *int         // feet/minute, positive up
	VerticalRateSource adsbtype.VRS // baro or GNSS; meaningful when VerticalRate != nil

	// GNSS minus barometric altitude difference.
	GNSSBaroDiff *int // feet, signed (positive = GNSS above baro)

	// Status / accuracy metadata (always present).
	IntentChange  bool
	IFRCapability bool
	NACv          uint8 // 0..7, velocity accuracy category
}

// Velocity returns the decoded airborne velocity (extended squitter type
// code 19, BDS 0,9). It returns an error wrapping ErrNotAvailable unless the
// message is a DF17/DF18 extended squitter with type code 19 and a defined
// subtype (1..4).
func (m *Message) Velocity() (*Velocity, error) {
	tc, err := m.raw.ESType()
	if err != nil {
		return nil, newError(err, "error retrieving velocity")
	}

	if adsbtype.TYPE(tc) != adsbtype.TYPE19 {
		return nil, errVelocityNotAvailable
	}

	r := m.raw

	v := &Velocity{
		Subtype:       safecast.MustConvert[uint8](r.esbits(6, 8)),
		IntentChange:  r.esbits(9, 9) == 1,
		IFRCapability: r.esbits(10, 10) == 1,
		NACv:          safecast.MustConvert[uint8](r.esbits(11, 13)),
	}

	switch v.Subtype {
	case velGroundSubsonic, velGroundSupersonic:
		decodeGroundSpeed(r, v)
	case velAirSubsonic, velAirSupersonic:
		decodeAirspeed(r, v)
	default:
		return nil, newErrorf(ErrNotAvailable,
			"unsupported velocity subtype %d", v.Subtype)
	}

	decodeVerticalRate(r, v)
	decodeGNSSBaroDiff(r, v)

	return v, nil
}

// velScale returns the velocity resolution in knots per count: 1 for the
// subsonic subtypes (1, 3) and 4 for the supersonic subtypes (2, 4).
func velScale(subtype uint8) float64 {
	if subtype == velGroundSupersonic || subtype == velAirSupersonic {
		return velSupersonicScale
	}

	return velSubsonicScale
}

// decodeGroundSpeed decodes the subtype 1/2 east-west and north-south
// velocity components into ground speed and true-north track. A zero raw
// component is the "no data" sentinel, leaving GroundSpeed and Track nil.
func decodeGroundSpeed(r *RawMessage, v *Velocity) {
	ew := r.esbits(15, 24)
	ns := r.esbits(26, 35)

	if ew == 0 || ns == 0 {
		return
	}

	scale := velScale(v.Subtype)

	vEW := (float64(ew) - 1) * scale
	if r.esbits(14, 14) == 1 { // 1 = west
		vEW = -vEW
	}

	vNS := (float64(ns) - 1) * scale
	if r.esbits(25, 25) == 1 { // 1 = south
		vNS = -vNS
	}

	gs := math.Hypot(vEW, vNS)

	track := math.Atan2(vEW, vNS) * 180 / math.Pi
	if track < 0 {
		track += 360
	}

	v.GroundSpeed = &gs
	v.Track = &track
}

// decodeAirspeed decodes the subtype 3/4 magnetic heading and airspeed.
// Heading is reported only when its status bit is set; a zero raw airspeed is
// the "no data" sentinel.
func decodeAirspeed(r *RawMessage, v *Velocity) {
	if r.esbits(14, 14) == 1 { // magnetic heading status
		hdg := float64(r.esbits(15, 24)) / headingSteps * 360
		v.Heading = &hdg
	}

	as := r.esbits(26, 35)
	if as == 0 {
		return
	}

	airspeed := (float64(as) - 1) * velScale(v.Subtype)
	v.Airspeed = &airspeed
	v.AirspeedType = adsbtype.AST(r.esbits(25, 25))
}

// decodeVerticalRate decodes the vertical rate common to every subtype. A
// zero raw magnitude is the "no data" sentinel.
func decodeVerticalRate(r *RawMessage, v *Velocity) {
	mag := r.esbits(38, 46)
	if mag == 0 {
		return
	}

	rate := safecast.MustConvert[int](mag-1) * verticalRateStep
	if r.esbits(37, 37) == 1 { // 1 = down
		rate = -rate
	}

	v.VerticalRate = &rate
	v.VerticalRateSource = adsbtype.VRS(r.esbits(36, 36))
}

// decodeGNSSBaroDiff decodes the GNSS-minus-barometric altitude difference
// (sign bit 49, 7-bit magnitude bits 50-56). A raw magnitude of 0 (all zeros)
// or 127 (all ones) is the "no data" sentinel.
func decodeGNSSBaroDiff(r *RawMessage, v *Velocity) {
	mag := r.esbits(50, 56)
	if mag == 0 || mag == gnssBaroNoData {
		return
	}

	diff := safecast.MustConvert[int](mag-1) * gnssBaroStep
	if r.esbits(49, 49) == 1 { // 1 = GNSS below baro
		diff = -diff
	}

	v.GNSSBaroDiff = &diff
}
