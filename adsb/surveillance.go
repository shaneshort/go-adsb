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

// This file exposes the Mode S surveillance and protocol subfields as decoded,
// typed values. The bit positions and format validity are enforced by the
// underlying RawMessage accessors and follow the reply formats of ICAO Annex
// 10 Vol IV. Each accessor returns an error wrapping ErrNotAvailable when the
// field is not present in the message's downlink format.

// The six-bit utility message field is a four-bit interrogator identifier
// subfield (IIS) followed by a two-bit identifier designator subfield (IDS).
const (
	utilityMessageIDSBits = 2
	utilityMessageIDSMask = 0x3
)

// UtilityMessage is a decoded utility message field (UM) from a surveillance
// altitude, surveillance identity or Comm-B reply (DF4/5/20/21). It carries the
// interrogator identifier subfield (IIS) and the identifier designator subfield
// (IDS), which reports the protocol the IIS relates to.
type UtilityMessage struct {
	InterrogatorID uint8               // IIS: interrogator identifier subfield (4 bits)
	Designator     adsbtype.Designator // IDS: identifier designator subfield
}

// Capability returns the transponder capability field (CA) of an all-call
// reply or extended squitter (DF11/17). It returns an error wrapping
// ErrNotAvailable for any other downlink format.
func (m *Message) Capability() (adsbtype.CA, error) {
	v, err := m.raw.CA()
	if err != nil {
		return 0, newError(err, "error retrieving capability")
	}

	return adsbtype.CA(v), nil
}

// CrossLinkCapability returns the cross-link capability field (CC) of a short
// air-air surveillance reply (DF0), reporting whether the transponder supports
// the cross-link protocol. It returns an error wrapping ErrNotAvailable for any
// other downlink format.
func (m *Message) CrossLinkCapability() (adsbtype.CC, error) {
	v, err := m.raw.CC()
	if err != nil {
		return 0, newError(err, "error retrieving cross-link capability")
	}

	return adsbtype.CC(v), nil
}

// ControlField returns the control field (CF) of a non-transponder extended
// squitter (DF18), identifying the source and format of the message. It
// returns an error wrapping ErrNotAvailable for any other downlink format.
func (m *Message) ControlField() (adsbtype.CF, error) {
	v, err := m.raw.CF()
	if err != nil {
		return 0, newError(err, "error retrieving control field")
	}

	return adsbtype.CF(v), nil
}

// FlightStatus returns the flight status field (FS) of a surveillance altitude,
// surveillance identity or Comm-B reply (DF4/5/20/21), reporting the alert,
// SPI and airborne/on-ground state. It returns an error wrapping
// ErrNotAvailable for any other downlink format.
func (m *Message) FlightStatus() (adsbtype.FS, error) {
	v, err := m.raw.FS()
	if err != nil {
		return 0, newError(err, "error retrieving flight status")
	}

	return adsbtype.FS(v), nil
}

// DownlinkRequest returns the downlink request field (DR) of a surveillance
// altitude, surveillance identity or Comm-B reply (DF4/5/20/21), reporting any
// pending Comm-B, ACAS or ELM transmission. It returns an error wrapping
// ErrNotAvailable for any other downlink format.
func (m *Message) DownlinkRequest() (adsbtype.DR, error) {
	v, err := m.raw.DR()
	if err != nil {
		return 0, newError(err, "error retrieving downlink request")
	}

	return adsbtype.DR(v), nil
}

// UtilityMessage returns the utility message field (UM) of a surveillance
// altitude, surveillance identity or Comm-B reply (DF4/5/20/21), split into its
// interrogator identifier (IIS) and identifier designator (IDS) subfields. It
// returns an error wrapping ErrNotAvailable for any other downlink format.
func (m *Message) UtilityMessage() (*UtilityMessage, error) {
	v, err := m.raw.UM()
	if err != nil {
		return nil, newError(err, "error retrieving utility message")
	}

	return &UtilityMessage{
		InterrogatorID: asU8(v >> utilityMessageIDSBits),
		Designator:     adsbtype.Designator(v & utilityMessageIDSMask),
	}, nil
}

// VerticalStatus returns the vertical status field (VS) of an air-air
// surveillance reply (DF0/16), reporting whether the aircraft is airborne or on
// the ground. It returns an error wrapping ErrNotAvailable for any other
// downlink format.
func (m *Message) VerticalStatus() (adsbtype.VS, error) {
	v, err := m.raw.VS()
	if err != nil {
		return 0, newError(err, "error retrieving vertical status")
	}

	return adsbtype.VS(v), nil
}

// SensitivityLevel returns the ACAS sensitivity level field (SL) of an air-air
// surveillance reply (DF0/16). It returns an error wrapping ErrNotAvailable for
// any other downlink format.
func (m *Message) SensitivityLevel() (adsbtype.SL, error) {
	v, err := m.raw.SL()
	if err != nil {
		return 0, newError(err, "error retrieving sensitivity level")
	}

	return adsbtype.SL(v), nil
}

// ReplyInformation returns the reply information field (RI) of an air-air
// surveillance reply (DF0/16), reporting the aircraft's ACAS capability or
// maximum airspeed. It returns an error wrapping ErrNotAvailable for any other
// downlink format.
func (m *Message) ReplyInformation() (adsbtype.RI, error) {
	v, err := m.raw.RI()
	if err != nil {
		return 0, newError(err, "error retrieving reply information")
	}

	return adsbtype.RI(v), nil
}
