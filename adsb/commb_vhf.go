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

// vhfChannelBits is the width of a VHF channel number subfield.
const vhfChannelBits = 15

// VHFChannel is one monitored channel of a BDS 4,8 VHF channel report. Valid
// reflects the channel's status bit; Channel is the raw 15-bit channel field
// and is zero when Valid is false. The channel encoding is mode-dependent: the
// general formula is 118.000 + Channel x 0.001 MHz, but VDL Mode 3, 8.33 kHz
// and analogue channels encode the field differently (indicated by its two
// most significant bits) per ICAO Doc 9871 Table A-2-72. AudioStatus reports
// the aircrew monitoring of the channel and is always present.
type VHFChannel struct {
	Valid       bool
	Channel     uint16
	AudioStatus adsbtype.AudioStatus
}

// VHFChannelReport is a decoded Comm-B VHF channel report (BDS 4,8), giving the
// three monitored VHF communications channels and the aircrew monitoring status
// of the 121.5 MHz emergency guard channel.
type VHFChannelReport struct {
	VHF1             VHFChannel
	VHF2             VHFChannel
	VHF3             VHFChannel
	GuardAudioStatus adsbtype.AudioStatus // 121.5 MHz emergency channel
}

// VHFChannelReport decodes the MB field as a BDS 4,8 VHF channel report (ICAO
// Doc 9871 Table A-2-72). It returns an error wrapping ErrNotAvailable unless
// the message is a Comm-B reply (DF 20 or 21). The register identity is not
// verified.
func (m *Message) VHFChannelReport() (*VHFChannelReport, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return &VHFChannelReport{
		VHF1:             decodeVHFChannel(r, 1, 16, 17),
		VHF2:             decodeVHFChannel(r, 19, 34, 35),
		VHF3:             decodeVHFChannel(r, 37, 52, 53),
		GuardAudioStatus: adsbtype.AudioStatus(r.mbbits(55, 56)),
	}, nil
}

// decodeVHFChannel decodes one VHF channel record: a 15-bit channel field
// starting at MB bit chStart, a status bit at statusBit, and a two-bit audio
// status starting at audioStart. The channel number is decoded only when the
// status bit is set.
func decodeVHFChannel(r *RawMessage, chStart, statusBit, audioStart int) VHFChannel {
	c := VHFChannel{
		Valid:       r.mbbits(statusBit, statusBit) == 1,
		AudioStatus: adsbtype.AudioStatus(r.mbbits(audioStart, audioStart+1)),
	}

	if c.Valid {
		c.Channel = asU16(r.mbbits(chStart, chStart+vhfChannelBits-1))
	}

	return c
}
