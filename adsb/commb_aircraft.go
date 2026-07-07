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

// Antenna position register layout (BDS 2,2, ICAO Doc 9871 Table A-2-34). The
// MB field holds four 14-bit antenna records, each an antenna type (3 bits), an
// X offset (6 bits) and a Z offset (5 bits).
const (
	antennaCount      = 4
	antennaRecordBits = 14
)

// Antenna is one antenna record from a BDS 2,2 antenna positions report. X is
// the distance aft of the nose along the aircraft centre line and Z the height
// above the ground, both in metres; a nil pointer means the offset was reported
// as invalid (a raw value of zero). The maximum raw value (63 for X, 31 for Z)
// means the offset is that value or greater.
type Antenna struct {
	Type adsbtype.AntennaType
	X    *int // metres aft of the nose; nil if invalid
	Z    *int // metres above the ground; nil if invalid
}

// AntennaPositions is a decoded Comm-B antenna positions report (BDS 2,2),
// giving the position of up to four Mode S and GNSS antennas.
type AntennaPositions struct {
	Antennas [antennaCount]Antenna
}

// AntennaPositions decodes the MB field as a BDS 2,2 antenna positions report
// (ICAO Doc 9871 Table A-2-34). It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified.
func (m *Message) AntennaPositions() (*AntennaPositions, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	ap := new(AntennaPositions)

	for i := range antennaCount {
		base := 1 + i*antennaRecordBits

		a := Antenna{Type: adsbtype.AntennaType(r.mbbits(base, base+2))}

		if x := safecast.MustConvert[int](r.mbbits(base+3, base+8)); x != 0 {
			a.X = &x
		}

		if z := safecast.MustConvert[int](r.mbbits(base+9, base+13)); z != 0 {
			a.Z = &z
		}

		ap.Antennas[i] = a
	}

	return ap, nil
}

// AircraftType is a decoded Comm-B aircraft type report (BDS 2,5). The fields
// follow the ICAO Doc 8643 aircraft description: Class, EngineType and
// WakeTurbulenceCategory are single Doc 8643 letter codes (for example L for a
// landplane, J for a jet engine, M for a medium wake category), and
// ModelDesignation is the ICAO type designator such as "B738".
type AircraftType struct {
	Class                  string // Doc 8643 aircraft type letter (e.g. L, S, A, H)
	NumberOfEngines        uint8  // 0-7; a value of 7 means seven or more
	EngineType             string // Doc 8643 engine type letter (e.g. P, T, J)
	ModelDesignation       string // ICAO Doc 8643 type designator, up to four characters; empty if not specified
	WakeTurbulenceCategory string // Doc 8643 wake category letter (e.g. L, M, H, J)
}

// aircraftModelChars is the number of Doc 8643 type-designator characters in
// the model designation subfield; a fifth character is reserved.
const aircraftModelChars = 4

// aircraftModelUnspecified is the model designation sentinel that ICAO Doc 9871
// Table A-2-37 defines to mean the type designator is not specified.
const aircraftModelUnspecified = "2222"

// AircraftType decodes the MB field as a BDS 2,5 aircraft type report (ICAO Doc
// 9871 Table A-2-37). The character subfields use the 6-bit IA-5 subset and
// carry ICAO Doc 8643 codes. It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified.
func (m *Message) AircraftType() (*AircraftType, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	model := mbString(r, 16, aircraftModelChars)
	if model == aircraftModelUnspecified {
		model = ""
	}

	return &AircraftType{
		Class:                  mbString(r, 1, 1),
		NumberOfEngines:        asU8(r.mbbits(7, 9)),
		EngineType:             mbString(r, 10, 1),
		ModelDesignation:       model,
		WakeTurbulenceCategory: mbString(r, 46, 1),
	}, nil
}
