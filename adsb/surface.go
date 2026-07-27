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

// surfaceTrackSteps is the number of discrete ground-track steps in the 7-bit
// track field; the track spans a full 360-degree revolution.
const surfaceTrackSteps = 128

// Field errors for message fields that are not carried by every downlink
// format. They are pre-built so that returning one does not allocate; see
// notAvailable.
var (
	errSurfaceMovementNotAvailable = notAvailable("surface movement")
)

// SurfaceMovement is the decoded ground speed and track from an ADS-B surface
// position message (extended squitter type codes 5..8, BDS 0,6).
//
// Optional quantities are pointers: a nil pointer means the field carried the
// register's "no data" sentinel or, for the track, an unset status bit.
type SurfaceMovement struct {
	GroundSpeed *float64 // knots
	Track       *float64 // degrees clockwise from true north, 0..360
}

// SurfaceMovement returns the decoded ground speed and track from a surface
// position message (extended squitter type codes 5..8, BDS 0,6). It returns
// an error wrapping ErrNotAvailable unless the message is a DF17/DF18 surface
// position message.
func (m *Message) SurfaceMovement() (*SurfaceMovement, error) {
	tc, err := m.raw.ESType()
	if err != nil {
		return nil, newError(err, "error retrieving surface movement")
	}

	if tc < surfacePosTypeLo || tc > surfacePosTypeHi {
		return nil, errSurfaceMovementNotAvailable
	}

	sm := new(SurfaceMovement)

	if speed, ok := surfaceSpeed(m.raw.esbits(6, 12)); ok {
		sm.GroundSpeed = &speed
	}

	// The ground track is valid only when its status bit (ME 13) is set.
	if m.raw.esbits(13, 13) == 1 {
		track := float64(m.raw.esbits(14, 20)) * 360 / surfaceTrackSteps
		sm.Track = &track
	}

	return sm, nil
}

// surfaceSpeed decodes the 7-bit BDS 0,6 movement field into a ground speed
// in knots. The encoding is piecewise-linear with increasing step size. The
// bool is false when the field carries no usable speed: 0 (no data) or
// 125-127 (reserved).
func surfaceSpeed(mov uint64) (float64, bool) {
	switch {
	case mov == 0 || mov >= 125: // no data / reserved
		return 0, false
	case mov == 1: // stopped
		return 0, true
	case mov <= 8: // 2-8: 0.125 kt steps from 0 kt
		return float64(mov-1) * 0.125, true
	case mov <= 12: // 9-12: 0.25 kt steps from 1 kt
		return 1 + float64(mov-9)*0.25, true
	case mov <= 38: // 13-38: 0.5 kt steps from 2 kt
		return 2 + float64(mov-13)*0.5, true
	case mov <= 93: // 39-93: 1 kt steps from 15 kt
		return 15 + float64(mov-39), true
	case mov <= 108: // 94-108: 2 kt steps from 70 kt
		return 70 + float64(mov-94)*2, true
	case mov <= 123: // 109-123: 5 kt steps from 100 kt
		return 100 + float64(mov-109)*5, true
	default: // 124: >= 175 kt
		return 175, true
	}
}
