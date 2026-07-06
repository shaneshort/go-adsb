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
	"slices"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/adsbtype"
)

// InferBDS returns each candidate register whose format is consistent with
// the MB field. The vectors here are the same ones decoded in the Comm-B
// tests; each should list its own register among the candidates.
func TestInferBDS(t *testing.T) {
	cases := []struct {
		name string
		hex  string
		want adsbtype.BDS
	}{
		{"BDS20", "A0000000200420C4820820000000", adsbtype.BDS20},
		{"BDS40", "A0000000BE85F430A40185000000", adsbtype.BDS40},
		{"BDS50", "A00000008E5C0134A204D7000000", adsbtype.BDS50},
		{"BDS60", "A0000000E009F5322107E0000000", adsbtype.BDS60},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mustVelMsg(t, c.hex).InferBDS()
			if err != nil {
				t.Fatalf("InferBDS: %v", err)
			}

			if !slices.Contains(got, c.want) {
				t.Errorf("InferBDS = %v, want to contain %s", got, c.want)
			}
		})
	}
}

// Vectors that fail a specific validity check must not list the corresponding
// register among the inference candidates.
func TestInferBDSInvalid(t *testing.T) {
	cases := []struct {
		name   string
		hex    string
		absent adsbtype.BDS
	}{
		{"GSOutOfRange", "A00000008E5C01642204D7000000", adsbtype.BDS50},
		{"RollOutOfRange", "A00000009F5C0134A204D7000000", adsbtype.BDS50},
		{"MachOutOfRange", "A0000000E009F53EA107E0000000", adsbtype.BDS60},
		{"ReservedBitsSet", "A0000000BE800000010000000000", adsbtype.BDS40},
		{"AltStatusMismatch", "A00000003E800030A40000000000", adsbtype.BDS40},
		{"ModeStatusResidue", "A0000000BE800000000080000000", adsbtype.BDS40},
		{"TrackRateResidue", "A00000008E5C01348204D7000000", adsbtype.BDS50},
		{"VertRateResidue", "A0000000E009F5320107E0000000", adsbtype.BDS60},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mustVelMsg(t, c.hex).InferBDS()
			if err != nil {
				t.Fatalf("InferBDS: %v", err)
			}

			if slices.Contains(got, c.absent) {
				t.Errorf("InferBDS = %v, want to exclude %s", got, c.absent)
			}
		})
	}
}

// An all-zero MB field is consistent with no register (each requires a status
// or self-identifying code).
func TestInferBDSNone(t *testing.T) {
	got, err := mustVelMsg(t, "A000000000000000000000000000").InferBDS()
	if err != nil {
		t.Fatalf("InferBDS: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("InferBDS = %v, want empty", got)
	}
}

// Inference requires a Comm-B reply (DF 20 or 21).
func TestInferBDSRejectNonReply(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").InferBDS()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
