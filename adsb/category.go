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
	"fmt"

	"kreklow.us/go/go-adsb/adsbtype"
)

// categorySets maps each identification type code to its emitter category
// set letter. The adsbtype.TYPE constants document the mapping (e.g. TYPE4 is
// "Identification (Category Set A)").
var categorySets = map[adsbtype.TYPE]byte{
	adsbtype.TYPE4: 'A',
	adsbtype.TYPE3: 'B',
	adsbtype.TYPE2: 'C',
	adsbtype.TYPE1: 'D',
}

// Category returns the ADS-B emitter category from an identification message
// (extended squitter type codes 1..4, BDS 0,8). The category set is selected
// by the type code (TC4 = set A, TC3 = B, TC2 = C, TC1 = D) and combined with
// the 3-bit category code into a value such as A5.
//
// It returns an error wrapping ErrNotAvailable unless the message is a
// DF17/DF18 identification message.
func (m *Message) Category() (adsbtype.AcCat, error) {
	tc, err := m.raw.ESType()
	if err != nil {
		return "", newError(err, "error retrieving category")
	}

	// The identification type code selects the category set; the 3-bit
	// category code (ME bits 6-8) selects within it, forming a value such
	// as A5.
	set, ok := categorySets[adsbtype.TYPE(tc)]
	if !ok {
		return "", newErrorf(ErrNotAvailable,
			"error retrieving category from type %d", tc)
	}

	code := m.raw.esbits(6, 8)

	return adsbtype.AcCat(fmt.Sprintf("%c%d", set, code)), nil
}
