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

// ModeAC is a decoded Mode A/C reply. A Mode A/C reply is 13 pulses that a
// receiver cannot unambiguously classify as a Mode A (identity) or Mode C
// (altitude) reply, so both interpretations are provided: Squawk is the Mode A
// identity code and Altitude is the Mode C pressure altitude. Altitude is nil
// when the pulses do not form a valid Mode C code (for example, a genuine
// Mode A identity that has no Gillham altitude equivalent).
type ModeAC struct {
	Squawk   []byte // Mode A identity as four octal digits
	SPI      bool   // special position identification pulse
	Altitude *int64 // Mode C pressure altitude in feet
}

// DecodeModeAC decodes the two-byte Mode A/C payload of a Beast type-1 frame
// (as returned by beast.Frame.ModeAC). The payload holds the 13 reply pulses
// in the layout 00:A4:A2:A1 00:B4:B2:B1 00:C4:C2:C1 00:D4:D2:D1.
func DecodeModeAC(data []byte) (*ModeAC, error) {
	if len(data) != 2 {
		return nil, newErrorf(nil, "expected 2 bytes, received %d", len(data))
	}

	v := uint64(data[0])<<8 | uint64(data[1])

	ma := &ModeAC{
		Squawk: []byte{
			asU8((v >> 12) & 0x7),
			asU8((v >> 8) & 0x7),
			asU8((v >> 4) & 0x7),
			asU8(v & 0x7),
		},
		SPI: v&0x0080 != 0,
	}

	alt, err := decodeAC(modeACtoAC(v))
	if err == nil {
		ma.Altitude = &alt
	}

	return ma, nil
}

// modeACtoAC repacks the pulses of a Beast Mode A/C code into the 13-bit
// altitude code (AC) field layout expected by decodeAC. Mode C does not use
// the D1 pulse, so the M and Q bits are left zero, selecting the Gillham
// (100-foot) decode path.
func modeACtoAC(v uint64) uint64 {
	a1, a2, a4 := (v>>12)&1, (v>>13)&1, (v>>14)&1
	b1, b2, b4 := (v>>8)&1, (v>>9)&1, (v>>10)&1
	c1, c2, c4 := (v>>4)&1, (v>>5)&1, (v>>6)&1
	d2, d4 := (v>>1)&1, (v>>2)&1

	return c1<<12 | a1<<11 | c2<<10 | a2<<9 | c4<<8 | a4<<7 |
		b1<<5 | b2<<3 | d2<<2 | b4<<1 | d4
}
