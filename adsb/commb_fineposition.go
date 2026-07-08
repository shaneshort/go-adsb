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

// Fine position register field scaling, from ICAO Doc 9871 Table A-2-82
// (BDS 5,2). The altitude reuses the coarse position altitude scaling and
// range (BDS 5,1).
const (
	fineLatLonStep = 90.0 / 16777216 // degrees per count (fine latitude, longitude)

	fineFOMGNSSMin = 12 // lowest FOM/source value whose altitude is a GNSS height
	fineFOMGNSSMax = 14 // highest FOM/source value whose altitude is a GNSS height
)

// PositionReportFine is a decoded Comm-B fine position report (BDS 5,2). It
// provides a high-precision refinement of the coarse position in BDS 5,1 and
// must be interpreted together with it.
//
// FOMSource is the figure-of-merit / source code (0-14, 15 reserved; see ICAO
// Doc 9871 Table A-2-82); AltitudeIsGNSSHeight reports whether Altitude is a
// GNSS height (FOMSource 12-14) rather than a pressure altitude. LatitudeFine
// and LongitudeFine are the signed (two's complement) fine refinements to
// combine with the BDS 5,1 coarse latitude and longitude; they are nil when the
// status bit is clear. Altitude is nil when its field is zeroed (reported
// invalid) or out of range.
type PositionReportFine struct {
	FOMSource            uint8
	AltitudeIsGNSSHeight bool
	LatitudeFine         *float64 // degrees
	LongitudeFine        *float64 // degrees
	Altitude             *int     // feet, -1000..126752
}

// PositionReportFine decodes the MB field as a BDS 5,2 fine position report
// (ICAO Doc 9871 Table A-2-82). It returns an error wrapping ErrNotAvailable
// unless the message is a Comm-B reply (DF 20 or 21). The register identity is
// not verified.
func (m *Message) PositionReportFine() (*PositionReportFine, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	p := &PositionReportFine{FOMSource: asU8(r.mbbits(2, 5))}
	p.AltitudeIsGNSSHeight = p.FOMSource >= fineFOMGNSSMin && p.FOMSource <= fineFOMGNSSMax

	// A single status bit gates the fine latitude and longitude, which are
	// two's complement (their most significant bit is the sign).
	if r.mbbits(1, 1) == 1 {
		lat := float64(mbTwosComplement(r, 6, 7, 23)) * fineLatLonStep
		p.LatitudeFine = &lat

		lon := float64(mbTwosComplement(r, 24, 25, 41)) * fineLatLonStep
		p.LongitudeFine = &lon
	}

	// The altitude has no status bit; an all-zero field is invalid, as is any
	// value outside the register's valid range.
	if r.mbbits(42, 56) != 0 {
		if alt := mbTwosComplement(r, 42, 43, 56) * coarseAltStep; alt >= coarseAltMin && alt <= coarseAltMax {
			p.Altitude = &alt
		}
	}

	return p, nil
}
