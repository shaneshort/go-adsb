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

// Transponder and ACAS identity register field layout, shared by BDS E,3 to
// E,6 (ICAO Doc 9871 Tables A-2-227 to A-2-230). The status bit is MB bit 1,
// the format type is MB bits 2-3, and the 48-bit payload (MB bits 4-51) is
// either 12 BCD part-number digits or 8 IA-5 characters; MB bits 52-56 are
// reserved.
const (
	identPayloadStart = 4  // first MB bit of the identity payload
	identBCDDigits    = 12 // BCD part-number digits (format IDF0)
	identChars        = 8  // IA-5 characters (format IDF1)
)

// TransponderIdentification is a decoded transponder or ACAS identity register
// (BDS E,3 to E,6). Valid reflects the status bit; when it is false both
// payload strings are empty. When Valid is true, exactly one of PartNumber
// (BCD digits) or TypeName (characters) is populated according to Format, and
// both are empty for a reserved format.
type TransponderIdentification struct {
	Valid      bool         // status bit (MB bit 1)
	Format     adsbtype.IDF // format type (MB bits 2-3)
	PartNumber string       // BCD digits, populated when Format is IDF0
	TypeName   string       // characters, populated when Format is IDF1
}

// decodeTransponderIdent decodes the shared BDS E,3 to E,6 identity format from
// the MB field. The payload is decoded only when the status bit is set; the
// format bits remain visible regardless.
func decodeTransponderIdent(r *RawMessage) *TransponderIdentification {
	ti := &TransponderIdentification{
		Valid:  r.mbbits(1, 1) == 1,
		Format: adsbtype.IDF(r.mbbits(2, 3)),
	}

	if !ti.Valid {
		return ti
	}

	switch ti.Format {
	case adsbtype.IDF0:
		ti.PartNumber = mbBCD(r, identPayloadStart, identBCDDigits)
	case adsbtype.IDF1:
		ti.TypeName = mbString(r, identPayloadStart, identChars)
	case adsbtype.IDF2, adsbtype.IDF3:
		// Reserved format types carry no defined payload.
	}

	return ti
}

// mbBCD decodes n consecutive 4-bit BCD digits from the MB field starting at
// bit start into a decimal string. A nibble greater than 9 is rendered as '?'.
func mbBCD(r *RawMessage, start, n int) string {
	b := make([]byte, n)

	for i := range n {
		pos := start + i*4

		d := r.mbbits(pos, pos+3)
		if d <= 9 {
			b[i] = '0' + asU8(d)
		} else {
			b[i] = '?'
		}
	}

	return string(b)
}

// TransponderPartNumber decodes the MB field as a BDS E,3 transponder type /
// part number register (ICAO Doc 9871 Table A-2-227). It returns an error
// wrapping ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21).
// The register identity is not verified.
func (m *Message) TransponderPartNumber() (*TransponderIdentification, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeTransponderIdent(r), nil
}

// TransponderSoftwareRevision decodes the MB field as a BDS E,4 transponder
// software revision number register (ICAO Doc 9871 Table A-2-228). It returns
// an error wrapping ErrNotAvailable unless the message is a Comm-B reply
// (DF 20 or 21). The register identity is not verified.
func (m *Message) TransponderSoftwareRevision() (*TransponderIdentification, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeTransponderIdent(r), nil
}

// ACASUnitPartNumber decodes the MB field as a BDS E,5 ACAS unit part number
// register (ICAO Doc 9871 Table A-2-229). It returns an error wrapping
// ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21). The
// register identity is not verified.
func (m *Message) ACASUnitPartNumber() (*TransponderIdentification, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeTransponderIdent(r), nil
}

// ACASUnitSoftwareRevision decodes the MB field as a BDS E,6 ACAS unit software
// revision register (ICAO Doc 9871 Table A-2-230). It returns an error wrapping
// ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21). The
// register identity is not verified.
func (m *Message) ACASUnitSoftwareRevision() (*TransponderIdentification, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return decodeTransponderIdent(r), nil
}
