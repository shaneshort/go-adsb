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
)

// A DF16 long air-air surveillance reply carries an ACAS resolution advisory
// (register 30) in its MV field, decoded at the same frame bits as the TC28
// subtype 2 broadcast. This vector holds the same register-30 content as the
// TC28 ACAS RA test.
func TestACASRADF16(t *testing.T) {
	ra, err := mustVelMsg(t, "8000000030800165018874000000").ACASRA()
	if err != nil {
		t.Fatalf("ACASRA: %v", err)
	}

	assertEq(t, "ARA", ra.ARA, 0x2000)
	assertEq(t, "RAC", ra.RAC, 0x5)
	assertTrue(t, "RATerminated", ra.RATerminated)
	assertFalse(t, "MultipleThreat", ra.MultipleThreat)
	assertEq(t, "ThreatTypeIndicator", ra.ThreatTypeIndicator, 1)
	assertTrue(t, "SingleThreat", ra.SingleThreat)

	if ra.ThreatICAO != 0x40621D {
		t.Errorf("ThreatICAO = %06X, want 40621D", ra.ThreatICAO)
	}
}

// The ACAS RA is only available from a DF16 reply.
func TestACASRARejectNonDF16(t *testing.T) {
	_, err := mustVelMsg(t, "8D485020994409940838175B284F").ACASRA()
	if !errors.Is(err, adsb.ErrNotAvailable) {
		t.Errorf("err = %v, want ErrNotAvailable", err)
	}
}
