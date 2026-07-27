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

// opStatusType is the extended squitter type code for the aircraft
// operational status message (BDS 6,5).
const opStatusType = 31

// gpsOffsetStep is the resolution in metres per count of the surface GPS
// antenna offset subfields.
const gpsOffsetStep = 2

// Field errors for message fields that are not carried by every downlink
// format. They are pre-built so that returning one does not allocate; see
// notAvailable.
var (
	errOperationalStatusNotAvailable = notAvailable("operational status")
)

// OperationalStatus is a decoded ADS-B aircraft operational status message
// (extended squitter type code 31, BDS 6,5). The layout of the capability
// class and operational mode regions depends on both the Subtype (0 =
// airborne, 1 = surface) and the ADS-B Version; the subtype-specific fields
// are carried in the Airborne or Surface member, exactly one of which is
// non-nil for a defined subtype.
type OperationalStatus struct {
	Subtype uint8 // 0 = airborne, 1 = surface
	Version uint8 // ADS-B version (0 = DO-260, 1 = DO-260A, 2 = DO-260B)

	NICSupplementA bool         // navigation integrity category supplement A
	NACp           uint8        // navigation accuracy category for position
	SIL            uint8        // source integrity level
	SILSupplement  uint8        // SIL per-sample vs per-hour (version 2)
	HRD            adsbtype.HRD // horizontal reference direction

	CapabilityClass uint16 // raw capability class region (ME 9-24)
	OperationalMode uint16 // raw operational mode region (ME 25-40)

	Airborne *AirborneStatus // non-nil for subtype 0
	Surface  *SurfaceStatus  // non-nil for subtype 1
}

// AirborneStatus holds the airborne (subtype 0) capability class and
// operational mode subfields. Field applicability varies by version: CDTI is
// version 0/1; Has1090ESIn, HasUATIn, GVA, SingleAntenna and SDA are version
// 2; ARVCapable, TSCapable, TCCapability and NICBaro are version 1/2.
type AirborneStatus struct {
	ACASOperational bool  // TCAS/ACAS operational (polarity normalised)
	CDTI            bool  // cockpit display of traffic information (v0/v1)
	Has1090ESIn     bool  // 1090ES receive capability (v2)
	HasUATIn        bool  // UAT receive capability (v2)
	ARVCapable      bool  // air-referenced velocity report capability
	TSCapable       bool  // target state report capability
	TCCapability    uint8 // trajectory change report capability
	GVA             uint8 // geometric vertical accuracy (v2)
	NICBaro         bool  // barometric altitude integrity code (v1/v2)

	TCASRAActive  bool  // TCAS/ACAS resolution advisory active
	IdentActive   bool  // IDENT switch active
	ReceivingATC  bool  // receiving ATC services
	SingleAntenna bool  // single antenna (v2)
	SDA           uint8 // system design assurance (v2)
}

// SurfaceStatus holds the surface (subtype 1) capability class and
// operational mode subfields. CDTI is version 1; Has1090ESIn, HasUATIn, NACv,
// NICSupplementC, SingleAntenna, SDA and the GPS antenna offsets are version 2.
type SurfaceStatus struct {
	PositionOffsetApplied bool  // position offset applied (POA)
	CDTI                  bool  // cockpit display of traffic information (v1)
	Has1090ESIn           bool  // 1090ES receive capability (v2)
	B2Low                 bool  // class B2 transmitter below 70 W
	HasUATIn              bool  // UAT receive capability (v2)
	NACv                  uint8 // navigation accuracy category for velocity (v2)
	NICSupplementC        bool  // NIC supplement C (v2)
	LengthWidthCode       uint8 // aircraft/vehicle length and width code
	TrackAngleIsHeading   bool  // surface direction field is heading, not track

	TCASRAActive  bool  // TCAS/ACAS resolution advisory active
	IdentActive   bool  // IDENT switch active
	ReceivingATC  bool  // receiving ATC services
	SingleAntenna bool  // single antenna (v2)
	SDA           uint8 // system design assurance (v2)

	GPSAntennaOffsetRight        bool  // lateral offset direction (v2)
	GPSAntennaOffsetLateral      uint8 // lateral offset, metres (v2)
	GPSAntennaOffsetLongitudinal uint8 // longitudinal offset aft of nose, metres (v2)
}

// OperationalStatus returns the decoded aircraft operational status (extended
// squitter type code 31, BDS 6,5). It returns an error wrapping
// ErrNotAvailable unless the message is a DF17/DF18 extended squitter with
// type code 31.
func (m *Message) OperationalStatus() (*OperationalStatus, error) {
	tc, err := m.raw.ESType()
	if err != nil {
		return nil, newError(err, "error retrieving operational status")
	}

	if tc != opStatusType {
		return nil, errOperationalStatusNotAvailable
	}

	r := m.raw

	os := &OperationalStatus{
		Subtype:         asU8(r.esbits(6, 8)),
		Version:         asU8(r.esbits(41, 43)),
		NICSupplementA:  r.esbits(44, 44) == 1,
		NACp:            asU8(r.esbits(45, 48)),
		SIL:             asU8(r.esbits(51, 52)),
		HRD:             adsbtype.HRD(r.esbits(54, 54)),
		CapabilityClass: asU16(r.esbits(9, 24)),
		OperationalMode: asU16(r.esbits(25, 40)),
	}

	// Only versions 0-2 (DO-260/A/B) are defined; the subfield layouts of
	// later versions are unknown, so decoding them would be misleading.
	if os.Version > 2 {
		return nil, newErrorf(ErrNotAvailable,
			"unsupported ADS-B version %d", os.Version)
	}

	if os.Version == 2 {
		os.SILSupplement = asU8(r.esbits(55, 55))
	}

	switch os.Subtype {
	case 0:
		os.Airborne = decodeAirborneStatus(r, os.Version)
	case 1:
		os.Surface = decodeSurfaceStatus(r, os.Version)
	default:
		return nil, newErrorf(ErrNotAvailable,
			"reserved operational status subtype %d", os.Subtype)
	}

	return os, nil
}

// decodeAirborneStatus decodes the airborne (subtype 0) capability class and
// operational mode subfields for the given ADS-B version.
func decodeAirborneStatus(r *RawMessage, version uint8) *AirborneStatus {
	a := new(AirborneStatus)

	decodeAirborneCC(r, version, a)

	// Operational mode subfields use the format-0 layout (ME 25-26 == 0).
	if r.esbits(25, 26) == 0 {
		a.TCASRAActive = r.esbits(27, 27) == 1
		a.IdentActive = r.esbits(28, 28) == 1
		a.ReceivingATC = r.esbits(29, 29) == 1

		if version == 2 {
			a.SingleAntenna = r.esbits(30, 30) == 1
			a.SDA = asU8(r.esbits(31, 32))
		}
	}

	// The barometric altitude integrity code is defined from version 1.
	if version >= 1 {
		a.NICBaro = r.esbits(53, 53) == 1
	}

	return a
}

// decodeAirborneCC decodes the version-dependent airborne capability class
// subfields. In versions 0 and 1, ME bit 11 is "Not-TCAS" (inverted) and ME
// bit 12 is CDTI; in version 2, ME bit 11 is "TCAS Operational" (direct
// polarity) and ME bit 12 is "1090ES IN".
func decodeAirborneCC(r *RawMessage, version uint8, a *AirborneStatus) {
	switch version {
	case 0, 1:
		a.ACASOperational = r.esbits(11, 11) == 0
		a.CDTI = r.esbits(12, 12) == 1

		if version == 1 {
			a.ARVCapable = r.esbits(15, 15) == 1
			a.TSCapable = r.esbits(16, 16) == 1
			a.TCCapability = asU8(r.esbits(17, 18))
		}
	case 2:
		a.ACASOperational = r.esbits(11, 11) == 1
		a.Has1090ESIn = r.esbits(12, 12) == 1
		a.ARVCapable = r.esbits(15, 15) == 1
		a.TSCapable = r.esbits(16, 16) == 1
		a.TCCapability = asU8(r.esbits(17, 18))
		a.HasUATIn = r.esbits(19, 19) == 1
		a.GVA = asU8(r.esbits(49, 50))
	}
}

// decodeSurfaceStatus decodes the surface (subtype 1) capability class and
// operational mode subfields for the given ADS-B version.
func decodeSurfaceStatus(r *RawMessage, version uint8) *SurfaceStatus {
	s := &SurfaceStatus{
		LengthWidthCode:     asU8(r.esbits(21, 24)),
		TrackAngleIsHeading: r.esbits(53, 53) == 1,
	}

	decodeSurfaceCC(r, version, s)

	// Operational mode subfields use the format-0 layout (ME 25-26 == 0).
	if r.esbits(25, 26) == 0 {
		s.TCASRAActive = r.esbits(27, 27) == 1
		s.IdentActive = r.esbits(28, 28) == 1
		s.ReceivingATC = r.esbits(29, 29) == 1

		if version == 2 {
			s.SingleAntenna = r.esbits(30, 30) == 1
			s.SDA = asU8(r.esbits(31, 32))
			decodeGPSAntennaOffset(r, s)
		}
	}

	return s
}

// decodeSurfaceCC decodes the version-dependent surface capability class
// subfields. ME bit 12 is CDTI in version 1 and "1090ES IN" in version 2.
func decodeSurfaceCC(r *RawMessage, version uint8, s *SurfaceStatus) {
	switch version {
	case 1:
		s.PositionOffsetApplied = r.esbits(11, 11) == 1
		s.CDTI = r.esbits(12, 12) == 1
		s.B2Low = r.esbits(15, 15) == 1
	case 2:
		s.PositionOffsetApplied = r.esbits(11, 11) == 1
		s.Has1090ESIn = r.esbits(12, 12) == 1
		s.B2Low = r.esbits(15, 15) == 1
		s.HasUATIn = r.esbits(16, 16) == 1
		s.NACv = asU8(r.esbits(17, 19))
		s.NICSupplementC = r.esbits(20, 20) == 1
	}
}

// decodeGPSAntennaOffset decodes the surface GPS antenna offset subfield: a
// lateral offset (direction plus 0/2/4/6 m) and a longitudinal offset aft of
// the nose (0-62 m). A raw magnitude of 0 means "no data".
func decodeGPSAntennaOffset(r *RawMessage, s *SurfaceStatus) {
	s.GPSAntennaOffsetRight = r.esbits(33, 33) == 1
	s.GPSAntennaOffsetLateral = asU8(r.esbits(34, 35)) * gpsOffsetStep
	s.GPSAntennaOffsetLongitudinal = asU8(r.esbits(36, 40)) * gpsOffsetStep
}

// asU8 narrows a bit-field value to uint8.
func asU8(v uint64) uint8 {
	return safecast.MustConvert[uint8](v)
}

// asU16 narrows a bit-field value to uint16.
func asU16(v uint64) uint16 {
	return safecast.MustConvert[uint16](v)
}
