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

package adsb_test

import (
	"errors"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/adsbtype"
)

// BDS 4,8 VHF channel report: three monitored channels and the 121.5 MHz guard
// channel. VHF 1 is valid (channel 3500, loudspeaker), VHF 2 is valid (channel
// 5000, headphones only), VHF 3 is invalid, and the guard channel reports
// nobody. Vector built from the MB bit ranges of ICAO Doc 9871 Table A-2-72.
func TestVHFChannelReport(t *testing.T) {
	v, err := mustVelMsg(t, "A00000001B59C9C4600001000000").VHFChannelReport()
	if err != nil {
		t.Fatalf("VHFChannelReport: %v", err)
	}

	wantVHF(t, "VHF1", v.VHF1, true, 3500, adsbtype.AudioStatus3)
	wantVHF(t, "VHF2", v.VHF2, true, 5000, adsbtype.AudioStatus2)
	wantVHF(t, "VHF3", v.VHF3, false, 0, adsbtype.AudioStatus0)

	if v.GuardAudioStatus != adsbtype.AudioStatus1 {
		t.Errorf("GuardAudioStatus = %v, want %v", v.GuardAudioStatus, adsbtype.AudioStatus1)
	}
}

// wantVHF asserts a VHF channel's validity, raw channel number and audio status.
func wantVHF(t *testing.T, name string, c adsb.VHFChannel, valid bool, ch uint16, as adsbtype.AudioStatus) {
	t.Helper()

	if c.Valid != valid {
		t.Errorf("%s.Valid = %t, want %t", name, c.Valid, valid)
	}

	if c.Channel != ch {
		t.Errorf("%s.Channel = %d, want %d", name, c.Channel, ch)
	}

	if c.AudioStatus != as {
		t.Errorf("%s.AudioStatus = %v, want %v", name, c.AudioStatus, as)
	}
}

// With the channel status bits clear (here with residual channel bits set),
// each channel is invalid and its raw channel number is zeroed.
func TestVHFChannelReportStatusClear(t *testing.T) {
	v, err := mustVelMsg(t, "A0000000FFFEFFFFBFFFEF000000").VHFChannelReport()
	if err != nil {
		t.Fatalf("VHFChannelReport: %v", err)
	}

	for _, c := range []struct {
		name string
		ch   adsb.VHFChannel
	}{{"VHF1", v.VHF1}, {"VHF2", v.VHF2}, {"VHF3", v.VHF3}} {
		if c.ch.Valid {
			t.Errorf("%s.Valid = true, want false", c.name)
		}

		if c.ch.Channel != 0 {
			t.Errorf("%s.Channel = %d, want 0", c.name, c.ch.Channel)
		}
	}
}

// The VHF channel report requires a DF20/21 reply.
func TestVHFChannelReportRejectNonReply(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").VHFChannelReport()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
