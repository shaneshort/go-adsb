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

import "kreklow.us/go/go-adsb/adsbtype"

// aircraftStatusType is the extended squitter type code for the aircraft
// status message (emergency/priority status and TCAS RA broadcast).
const aircraftStatusType = 28

// modeADisabled is the Mode A code (octal 3000) that disables broadcast of
// the Mode A code in the emergency/priority status message.
const modeADisabled = 3000

// ACAS RA threat position scaling (ICAO Annex 10 Vol IV §4.3.8.4.2.2.1.6).
const (
	acasRangeStep      = 0.1 // NM per count of the threat range field
	acasBearingStep    = 6   // degrees per sector of the threat bearing field
	acasBearingSectors = 60  // highest assigned threat bearing sector
)

// AircraftStatus is a decoded extended squitter aircraft status message
// (type code 28). Subtype 1 carries emergency/priority status and the Mode A
// code; subtype 2 carries a TCAS resolution advisory broadcast. Exactly one
// of Emergency or ACASRA is non-nil for a defined subtype.
type AircraftStatus struct {
	Subtype   uint8
	Emergency *EmergencyStatus // non-nil for subtype 1
	ACASRA    *ACASRA          // non-nil for subtype 2
}

// EmergencyStatus is the emergency/priority status and Mode A code from a
// subtype 1 aircraft status message.
type EmergencyStatus struct {
	State  adsbtype.EPS // emergency/priority status
	Squawk []byte       // Mode A code as four octal digits; nil if broadcast disabled
}

// ACASRA is the TCAS resolution advisory broadcast from a subtype 2 aircraft
// status message. The fields conform to transponder register 30 and are
// decoded per ICAO Annex 10 Vol IV §4.3.8.4.2.2.1.
//
// The ARA sub-flags (Corrective through Positive) use the single-threat
// coding, which applies when SingleThreat is true. When SingleThreat is false
// and MultipleThreat is true, ARA bits 42-47 carry the multiple-threat coding
// instead; the raw ARA field is retained for that case.
//
// The threat identity fields are populated according to ThreatTypeIndicator:
// ThreatICAO for a value of 1, and ThreatAltitude/Range/Bearing for 2. They
// are left zero/nil for the no-identity (0) and reserved (3) cases; the rest
// of the resolution advisory remains valid regardless.
type ACASRA struct {
	ARA                 uint16 // active resolution advisories (14 bits, raw)
	RAC                 uint8  // resolution advisory complements (4 bits, raw)
	RATerminated        bool   // RA has been terminated
	MultipleThreat      bool   // more than one threat
	ThreatTypeIndicator uint8  // 0 = none, 1 = Mode-S address, 2 = alt/range/bearing, 3 = reserved
	ThreatIdentity      uint64 // threat identity data (26 bits, raw)
	ThreatICAO          uint64 // threat Mode-S address, set when ThreatTypeIndicator == 1

	SingleThreat     bool // single threat or same-direction RA (ARA bit 41)
	Corrective       bool // RA is corrective rather than preventive
	DownwardSense    bool // RA has a downward sense
	IncreasedRate    bool // RA is an increased-rate advisory
	SenseReversal    bool // RA is a sense reversal
	AltitudeCrossing bool // RA is altitude crossing
	Positive         bool // RA is positive rather than a vertical speed limit

	DoNotPassBelow bool // resolution advisory complement (RAC)
	DoNotPassAbove bool
	DoNotTurnLeft  bool
	DoNotTurnRight bool

	ThreatAltitude *int64   // threat barometric altitude, feet (ThreatTypeIndicator == 2)
	ThreatRange    *float64 // threat range, NM (ThreatTypeIndicator == 2)
	ThreatBearing  *float64 // threat bearing relative to heading, degrees (ThreatTypeIndicator == 2)
}

// AircraftStatus returns the decoded aircraft status message (extended
// squitter type code 28). It returns an error wrapping ErrNotAvailable unless
// the message is a DF17/DF18 extended squitter with type code 28 and a
// defined subtype (1 or 2).
func (m *Message) AircraftStatus() (*AircraftStatus, error) {
	tc, err := m.raw.ESType()
	if err != nil {
		return nil, newError(err, "error retrieving aircraft status")
	}

	if tc != aircraftStatusType {
		return nil, newErrorf(ErrNotAvailable,
			"error retrieving aircraft status from type %d", tc)
	}

	r := m.raw
	as := &AircraftStatus{Subtype: asU8(r.esbits(6, 8))}

	switch as.Subtype {
	case 1:
		as.Emergency = decodeEmergency(r)
	case 2:
		as.ACASRA = decodeACASRA(r)
	default:
		return nil, newErrorf(ErrNotAvailable,
			"reserved aircraft status subtype %d", as.Subtype)
	}

	return as, nil
}

// decodeEmergency decodes the subtype 1 emergency/priority status and Mode A
// code.
func decodeEmergency(r *RawMessage) *EmergencyStatus {
	e := &EmergencyStatus{
		State: adsbtype.EPS(r.esbits(9, 11)),
	}

	sqk := decodeModeA(r)
	if octalSquawk(sqk) != modeADisabled {
		e.Squawk = sqk
	}

	return e
}

// decodeModeA decodes the Mode A (4096) code from the pulse sequence in ME
// bits 12-24 (C1, A1, C2, A2, C4, A4, X, B1, D1, B2, D2, B4, D4) into four
// octal digits in A, B, C, D order.
func decodeModeA(r *RawMessage) []byte {
	a := asU8(r.esbits(13, 13)<<2 | r.esbits(15, 15)<<1 | r.esbits(17, 17))
	b := asU8(r.esbits(19, 19)<<2 | r.esbits(21, 21)<<1 | r.esbits(23, 23))
	c := asU8(r.esbits(12, 12)<<2 | r.esbits(14, 14)<<1 | r.esbits(16, 16))
	d := asU8(r.esbits(20, 20)<<2 | r.esbits(22, 22)<<1 | r.esbits(24, 24))

	return []byte{a, b, c, d}
}

// octalSquawk combines four octal squawk digits into a decimal value for
// comparison against sentinel codes such as 3000.
func octalSquawk(sqk []byte) int {
	return int(sqk[0])*1000 + int(sqk[1])*100 + int(sqk[2])*10 + int(sqk[3])
}

// acasRAFormat is the downlink format of the long air-air surveillance reply
// whose MV field carries the ACAS resolution advisory (register 30).
const acasRAFormat = 16

// ACASRA returns the decoded ACAS resolution advisory from the MV field of a
// DF16 long air-air surveillance reply, interpreting it as transponder
// register 30 (ICAO Annex 10 Vol IV §4.3.8.4.2.2.1). The MV field occupies the
// same frame bits (33-88) as the extended squitter ME field, so the register
// decode is shared with the TC28 subtype 2 broadcast.
//
// It returns an error wrapping ErrNotAvailable unless the message is a DF16
// reply. The register identity is not verified; a DF16 reply to a GICB request
// for another register decodes as a meaningless RA.
func (m *Message) ACASRA() (*ACASRA, error) {
	df, err := m.raw.DF()
	if err != nil {
		return nil, newError(err, "error retrieving ACAS RA")
	}

	if df != acasRAFormat {
		return nil, newErrorf(ErrNotAvailable, "ACAS RA not available in format %d", df)
	}

	return decodeACASRA(m.raw), nil
}

// decodeACASRA decodes the TCAS resolution advisory (register 30) per ICAO
// Annex 10 Vol IV §4.3.8.4.2.2.1. It reads the shared frame bits 41-88, so it
// serves the TC28 subtype 2 broadcast, the DF16 MV field, and Comm-B register
// 30 alike.
func decodeACASRA(r *RawMessage) *ACASRA {
	ra := &ACASRA{
		ARA:                 asU16(r.esbits(9, 22)),
		RAC:                 asU8(r.esbits(23, 26)),
		RATerminated:        r.esbits(27, 27) == 1,
		MultipleThreat:      r.esbits(28, 28) == 1,
		ThreatTypeIndicator: asU8(r.esbits(29, 30)),
		ThreatIdentity:      r.esbits(31, 56),

		SingleThreat:     r.esbits(9, 9) == 1,
		Corrective:       r.esbits(10, 10) == 1,
		DownwardSense:    r.esbits(11, 11) == 1,
		IncreasedRate:    r.esbits(12, 12) == 1,
		SenseReversal:    r.esbits(13, 13) == 1,
		AltitudeCrossing: r.esbits(14, 14) == 1,
		Positive:         r.esbits(15, 15) == 1,

		DoNotPassBelow: r.esbits(23, 23) == 1,
		DoNotPassAbove: r.esbits(24, 24) == 1,
		DoNotTurnLeft:  r.esbits(25, 25) == 1,
		DoNotTurnRight: r.esbits(26, 26) == 1,
	}

	switch ra.ThreatTypeIndicator {
	case 1:
		// The threat identity data begins with the threat's Mode-S address.
		ra.ThreatICAO = r.esbits(31, 54)
	case 2:
		decodeThreatPosition(r, ra)
	}

	return ra
}

// decodeThreatPosition decodes the threat altitude, range and bearing carried
// in the threat identity data when the threat type indicator is 2.
func decodeThreatPosition(r *RawMessage, ra *ACASRA) {
	// TIDA (ME 31-43) is a Mode C altitude code, decoded like an altitude
	// code field.
	alt, err := decodeAC(r.esbits(31, 43))
	if err == nil {
		ra.ThreatAltitude = &alt
	}

	// TIDR (ME 44-50): range in NM; 0 means no estimate.
	if n := r.esbits(44, 50); n != 0 {
		nm := float64(n-1) * acasRangeStep
		ra.ThreatRange = &nm
	}

	// TIDB (ME 51-56): bearing in degrees; 0 (and the unassigned 61-63) means
	// no estimate. Codes 1-60 map to the midpoint of a 6-degree sector.
	if n := r.esbits(51, 56); n >= 1 && n <= acasBearingSectors {
		deg := acasBearingStep*float64(n) - acasBearingStep/2
		ra.ThreatBearing = &deg
	}
}
