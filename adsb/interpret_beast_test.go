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
	"encoding/hex"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/beast"
)

// beastFrame builds a Beast frame of the given type byte with a fixed timestamp
// and signal level around the supplied payload.
func beastFrame(t *testing.T, typ byte, payload []byte) *beast.Frame {
	t.Helper()

	raw := make([]byte, 0, 9+len(payload))
	raw = append(raw, 0x1a, typ, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x64)
	raw = append(raw, payload...)

	f := new(beast.Frame)

	err := f.UnmarshalBinary(raw)
	if err != nil {
		t.Fatalf("Frame.UnmarshalBinary: %v", err)
	}

	return f
}

// A Mode S Beast frame is interpreted as its message, with the Beast metadata
// (type, timestamp and signal) preserved.
func TestInterpretBeastModeS(t *testing.T) {
	msg, err := hex.DecodeString("8D40621D58C382D690C8AC2863A7")
	if err != nil {
		t.Fatalf("hex.DecodeString: %v", err)
	}

	in, err := adsb.InterpretBeastFrame(beastFrame(t, 0x33, msg), adsb.InterpretOptions{})
	if err != nil {
		t.Fatalf("InterpretBeastFrame: %v", err)
	}

	if in.Family != adsb.FamilyADSB {
		t.Errorf("Family = %v, want ADSB", in.Family)
	}

	if in.Meta.BeastType != 0x33 {
		t.Errorf("BeastType = %#x, want 0x33", in.Meta.BeastType)
	}

	if in.Meta.Timestamp == nil {
		t.Error("Timestamp not preserved")
	}

	if in.Meta.Signal == nil || *in.Meta.Signal != 0x64 {
		t.Errorf("Signal = %v, want 0x64", in.Meta.Signal)
	}

	if observation(in, adsb.KindCPR) == nil {
		t.Error("no CPR observation from Mode S frame")
	}
}

// A Mode A/C Beast frame is classified as the Mode A/C family with the decoded
// Mode A/C data in the metadata.
func TestInterpretBeastModeAC(t *testing.T) {
	in, err := adsb.InterpretBeastFrame(beastFrame(t, 0x31, []byte{0x02, 0x00}), adsb.InterpretOptions{})
	if err != nil {
		t.Fatalf("InterpretBeastFrame: %v", err)
	}

	if in.Family != adsb.FamilyModeAC {
		t.Errorf("Family = %v, want ModeAC", in.Family)
	}

	if in.Meta.ModeAC == nil {
		t.Error("Meta.ModeAC not decoded")
	}

	if in.Meta.BeastType != 0x31 {
		t.Errorf("BeastType = %#x, want 0x31", in.Meta.BeastType)
	}
}
