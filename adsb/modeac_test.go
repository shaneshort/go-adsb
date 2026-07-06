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
	"bytes"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
)

// The Mode C altitude vector is derived from the repository's DF4 Gillham
// test (1300 ft): its 13-bit AC field pulses repacked into the Beast Mode A/C
// layout must decode back to 1300 ft. Its Mode A interpretation is squawk
// 0710.
func TestDecodeModeACAltitude(t *testing.T) {
	ma, err := adsb.DecodeModeAC([]byte{0x07, 0x10})
	if err != nil {
		t.Fatalf("DecodeModeAC: %v", err)
	}

	if !bytes.Equal(ma.Squawk, []byte{0, 7, 1, 0}) {
		t.Errorf("Squawk = %v, want [0 7 1 0]", ma.Squawk)
	}

	if ma.SPI {
		t.Error("SPI = true, want false")
	}

	if ma.Altitude == nil || *ma.Altitude != 1300 {
		t.Errorf("Altitude = %v, want 1300", ma.Altitude)
	}
}

// A VFR squawk of 1200 decodes as Mode A identity 1200.
func TestDecodeModeACSquawk(t *testing.T) {
	ma, err := adsb.DecodeModeAC([]byte{0x12, 0x00})
	if err != nil {
		t.Fatalf("DecodeModeAC: %v", err)
	}

	if !bytes.Equal(ma.Squawk, []byte{1, 2, 0, 0}) {
		t.Errorf("Squawk = %v, want [1 2 0 0]", ma.Squawk)
	}
}

// The D1 pulse is Mode A only; a code that sets it has no valid Mode C
// altitude even though its other pulses match an altitude code.
func TestDecodeModeACD1Set(t *testing.T) {
	ma, err := adsb.DecodeModeAC([]byte{0x07, 0x11})
	if err != nil {
		t.Fatalf("DecodeModeAC: %v", err)
	}

	if !bytes.Equal(ma.Squawk, []byte{0, 7, 1, 1}) {
		t.Errorf("Squawk = %v, want [0 7 1 1]", ma.Squawk)
	}

	if ma.Altitude != nil {
		t.Errorf("Altitude = %v, want nil (D1 set)", *ma.Altitude)
	}
}

// SPI marks a Mode A identity reply, so no Mode C altitude is reported even
// when the remaining pulses match an altitude code.
func TestDecodeModeACSPINoAltitude(t *testing.T) {
	ma, err := adsb.DecodeModeAC([]byte{0x07, 0x90})
	if err != nil {
		t.Fatalf("DecodeModeAC: %v", err)
	}

	if !ma.SPI {
		t.Error("SPI = false, want true")
	}

	if !bytes.Equal(ma.Squawk, []byte{0, 7, 1, 0}) {
		t.Errorf("Squawk = %v, want [0 7 1 0]", ma.Squawk)
	}

	if ma.Altitude != nil {
		t.Errorf("Altitude = %v, want nil (SPI set)", *ma.Altitude)
	}
}

// The SPI pulse (0x0080) is reported.
func TestDecodeModeACSPI(t *testing.T) {
	ma, err := adsb.DecodeModeAC([]byte{0x00, 0x80})
	if err != nil {
		t.Fatalf("DecodeModeAC: %v", err)
	}

	if !ma.SPI {
		t.Error("SPI = false, want true")
	}

	if !bytes.Equal(ma.Squawk, []byte{0, 0, 0, 0}) {
		t.Errorf("Squawk = %v, want [0 0 0 0]", ma.Squawk)
	}
}

// A payload of the wrong length is an error.
func TestDecodeModeACBadLength(t *testing.T) {
	_, err := adsb.DecodeModeAC([]byte{0x00})
	if err == nil {
		t.Error("expected error for 1-byte payload")
	}
}
