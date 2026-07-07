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

// BDS E,3 with the character format (type 01) carries an 8-character
// transponder type name (ICAO Doc 9871 Table A-2-227).
func TestTransponderPartNumberCharacter(t *testing.T) {
	ti, err := mustVelMsg(t, "A0000000A08418828C3900000000").TransponderPartNumber()
	if err != nil {
		t.Fatalf("TransponderPartNumber: %v", err)
	}

	if !ti.Valid {
		t.Error("Valid = false, want true")
	}

	if ti.Format != adsbtype.IDF1 {
		t.Errorf("Format = %v, want %v", ti.Format, adsbtype.IDF1)
	}

	if ti.TypeName != "ABCDEFGH" {
		t.Errorf("TypeName = %q, want %q", ti.TypeName, "ABCDEFGH")
	}

	if ti.PartNumber != "" {
		t.Errorf("PartNumber = %q, want empty", ti.PartNumber)
	}
}

// BDS E,3 with the part-number format (type 00) carries 12 BCD part-number
// digits.
func TestTransponderPartNumberBCD(t *testing.T) {
	ti, err := mustVelMsg(t, "A000000082468ACF120240000000").TransponderPartNumber()
	if err != nil {
		t.Fatalf("TransponderPartNumber: %v", err)
	}

	if ti.Format != adsbtype.IDF0 {
		t.Errorf("Format = %v, want %v", ti.Format, adsbtype.IDF0)
	}

	if ti.PartNumber != "123456789012" {
		t.Errorf("PartNumber = %q, want %q", ti.PartNumber, "123456789012")
	}

	if ti.TypeName != "" {
		t.Errorf("TypeName = %q, want empty", ti.TypeName)
	}
}

// When the status bit is clear, the payload fields are empty even if the
// register carries a non-zero payload (the format bits remain visible).
func TestTransponderPartNumberStatusClear(t *testing.T) {
	ti, err := mustVelMsg(t, "A000000002468ACF120240000000").TransponderPartNumber()
	if err != nil {
		t.Fatalf("TransponderPartNumber: %v", err)
	}

	if ti.Valid {
		t.Error("Valid = true, want false")
	}

	if ti.PartNumber != "" {
		t.Errorf("PartNumber = %q, want empty", ti.PartNumber)
	}

	if ti.TypeName != "" {
		t.Errorf("TypeName = %q, want empty", ti.TypeName)
	}
}

// The transponder software revision, ACAS part number and ACAS software
// revision registers (BDS E,4 to E,6) share the E,3 format and decoder.
func TestTransponderIdentSharedFormat(t *testing.T) {
	msg := mustVelMsg(t, "A0000000A08418828C3900000000")

	for _, tc := range []struct {
		name string
		call func() (*adsb.TransponderIdentification, error)
	}{
		{"TransponderSoftwareRevision", msg.TransponderSoftwareRevision},
		{"ACASUnitPartNumber", msg.ACASUnitPartNumber},
		{"ACASUnitSoftwareRevision", msg.ACASUnitSoftwareRevision},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ti, err := tc.call()
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if ti.TypeName != "ABCDEFGH" {
				t.Errorf("TypeName = %q, want %q", ti.TypeName, "ABCDEFGH")
			}
		})
	}
}

// The transponder-identity registers require a DF20/21 reply.
func TestTransponderIdentRejectNonReply(t *testing.T) {
	msg := mustVelMsg(t, "8D485020994409940838175B284F")

	for _, tc := range []struct {
		name string
		call func() (*adsb.TransponderIdentification, error)
	}{
		{"TransponderPartNumber", msg.TransponderPartNumber},
		{"TransponderSoftwareRevision", msg.TransponderSoftwareRevision},
		{"ACASUnitPartNumber", msg.ACASUnitPartNumber},
		{"ACASUnitSoftwareRevision", msg.ACASUnitSoftwareRevision},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.call()
			if !errors.Is(err, adsb.ErrNotAvailable) {
				t.Errorf("%s err = %v, want ErrNotAvailable", tc.name, err)
			}
		})
	}
}
