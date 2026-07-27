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
	"errors"
	"testing"

	"kreklow.us/go/go-adsb/adsb"
)

// allocRuns is the number of iterations testing.AllocsPerRun averages over.
const allocRuns = 1000

// Message vectors used by the allocation matrix.
//
// The naVectorsRaw set is the same set the raw field harness in raw_test.go
// uses, so that between them every downlink format the package accepts is
// represented and every raw field reaches its not-available branch on at
// least one of them.
var naVectorsRaw = []string{
	"02a183bb451e00",               // DF0
	"20000db867652d",               // DF4
	"2ab800673a57d0",               // DF5
	"5daa234a912889",               // DF11
	"80e1953058ab0160a09be809c86e", // DF16
	"8da2f111581fb4842d1f59eea2b7", // DF17
	"92a1ce0e90b973c26a380f56254c", // DF18
	"9aa1ce0e90b973c26a380f56254c", // DF19
	"a000149710030a80e500005b757a", // DF20
	"a8e9786015a68e5baedb2aba4f91", // DF21
	"c2255448ac2a74d003547a6db1a1", // DF24
}

// Vectors chosen so that the decoder under test reaches its own pre-built
// error rather than wrapping a failure of its guard accessor.
const (
	// naAirPos is a DF17 extended squitter with type code 11, an airborne
	// position. ESType and DF both succeed on it, so a decoder that rejects
	// the type code or the downlink format returns its pre-built error.
	naAirPos = "8D40621D58C382D690C8AC2863A7"

	// naVelocity is a DF17 extended squitter with type code 19, an airborne
	// velocity. Type code 19 carries neither a position nor an altitude, so
	// CPR, AltitudeSource and ESAltitude reach their pre-built errors on it.
	naVelocity = "8D485020994409940838175B284F"

	// naAllCall is a DF11 all-call reply. Alt reaches its pre-built error
	// only through its default arm, which needs a downlink format outside
	// both the surveillance set (0, 4, 16, 20) and the extended squitter set
	// (17, 18).
	naAllCall = "5daa234a912889"

	// naDF18CF0 is a DF18 reply with control field 0, an ADS-B message from a
	// non-transponder device. CF succeeds on it, so IMF and
	// TISBCoarsePosition reach their pre-built errors; on a DF17 message CF
	// itself fails and they wrap that failure instead.
	naDF18CF0 = "90a1ce0e90b973c26a380f56254c"

	// naSurveillance is a DF4 altitude reply, which carries no extended
	// squitter at all. It is used to exercise the wrapping paths.
	naSurveillance = "20001910bc45e9"
)

// naRawMessage unmarshals a hex vector into a RawMessage.
func naRawMessage(t *testing.T, m string) *adsb.RawMessage {
	t.Helper()

	data, err := hex.DecodeString(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rm := new(adsb.RawMessage)

	err = rm.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return rm
}

// naMessage unmarshals a hex vector into a Message.
func naMessage(t *testing.T, m string) *adsb.Message {
	t.Helper()

	data, err := hex.DecodeString(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := new(adsb.Message)

	err = msg.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return msg
}

// TestNotAvailableIsAllocationFree asserts that returning a field that a
// message does not carry costs no allocation.
//
// A field that a message does not carry is normal control flow, not an
// exception: probing for one must not allocate. The pre-built errors are
// declared with the static type error rather than the concrete adsbError, and
// this test fails if that is ever reverted, since returning a concrete value
// as an error boxes it and costs one allocation.
//
// Zero allocation is a property of the current compiler's escape analysis
// rather than of the language specification. This test is the measurement
// that keeps the property true.
func TestNotAvailableIsAllocationFree(t *testing.T) {
	t.Run("RawFields", testNotAvailableRawFields)
	t.Run("MessageDecoders", testNotAvailableDecoders)
	t.Run("WrappedPaths", testNotAvailableWrapped)
}

// rawFields returns every RawMessage accessor with an integer result that has
// a not-available branch. DF is absent: it fails only when no data is loaded,
// so it has no such branch. MD is absent because it returns a byte slice.
func rawFields(rm *adsb.RawMessage) map[string]func() (uint64, error) {
	return map[string]func() (uint64, error){
		"AA": rm.AA, "AC": rm.AC, "AF": rm.AF, "AP": rm.AP,
		"CA": rm.CA, "CC": rm.CC, "CF": rm.CF, "DP": rm.DP,
		"DR": rm.DR, "FS": rm.FS, "ID": rm.ID, "KE": rm.KE,
		"MB": rm.MB, "ME": rm.ME, "MV": rm.MV, "ND": rm.ND,
		"PI": rm.PI, "RI": rm.RI, "SL": rm.SL, "UM": rm.UM,
		"VS": rm.VS, "ESType": rm.ESType, "ESAltitude": rm.ESAltitude,
	}
}

// testNotAvailableRawFields covers every RawMessage field accessor, across the
// full set of downlink formats the package accepts.
func testNotAvailableRawFields(t *testing.T) {
	measured := make(map[string]bool)

	for _, vector := range naVectorsRaw {
		rm := naRawMessage(t, vector)

		for name, fn := range rawFields(rm) {
			if assertNotAvailableFree(t, vector+" "+name, discard(fn)) {
				measured[name] = true
			}
		}

		// MD returns a byte slice rather than an integer, so it is not part
		// of the map above.
		if assertNotAvailableFree(t, vector+" MD", discard(rm.MD)) {
			measured["MD"] = true
		}
	}

	// Every accessor must reach its not-available branch on at least one of
	// the vectors. Without this the cross-product above would silently stop
	// measuring an accessor, and still report success, if the vector set or
	// an accessor's format coverage ever changed.
	fields := rawFields(naRawMessage(t, naVectorsRaw[0]))

	names := make([]string, 0, len(fields)+1)
	names = append(names, "MD")

	for name := range fields {
		names = append(names, name)
	}

	for _, name := range names {
		if !measured[name] {
			t.Errorf("%s reached no not-available branch; the vector set no longer covers it", name)
		}
	}

	// ESAltitude reaches its own pre-built error only when ESType succeeds
	// and returns a type code that carries no altitude. Type code 19 is such
	// a code; on the vectors above ESType itself fails, so the error returned
	// is the one belonging to ESType.
	rm := naRawMessage(t, naVelocity)

	requireNotAvailableFree(t, "ESAltitude on type 19", discard(rm.ESAltitude))
}

// testNotAvailableDecoders covers the Message decoders. Each is paired with a
// vector on which its guard accessor succeeds, so that the decoder returns its
// own pre-built error rather than wrapping a failure of the guard.
func testNotAvailableDecoders(t *testing.T) {
	airPos := naMessage(t, naAirPos)
	velocity := naMessage(t, naVelocity)
	allCall := naMessage(t, naAllCall)
	df18 := naMessage(t, naDF18CF0)

	cases := []struct {
		name string
		fn   func() error
	}{
		// Guard is ESType, which succeeds on a DF17 message; type code 11 is
		// wrong for each of these decoders.
		{"Velocity", discard(airPos.Velocity)},
		{"Category", discard(airPos.Category)},
		{"SurfaceMovement", discard(airPos.SurfaceMovement)},
		{"AircraftStatus", discard(airPos.AircraftStatus)},
		{"OperationalStatus", discard(airPos.OperationalStatus)},
		{"TargetState", discard(airPos.TargetState)},

		// Guard is DF, which succeeds on any unmarshalled message.
		{"Call", discard(airPos.Call)},
		{"Sqk", discard(airPos.Sqk)},
		{"ACASRA", discard(airPos.ACASRA)},

		// Type code 19 carries no position and no altitude reference frame.
		{"CPR", discard(velocity.CPR)},
		{"AltitudeSource", discard(velocity.AltitudeSource)},

		// Alt reaches its pre-built error only through its default arm.
		{"Alt", discard(allCall.Alt)},

		// Guard is CF, which succeeds only on a DF18 message.
		{"IMF", discard(df18.IMF)},
		{"TISBCoarsePosition", discard(df18.TISBCoarsePosition)},

		// The Comm-B family rejects through commBRaw, whose guard is DF.
		{"DataLinkCapability", discard(airPos.DataLinkCapability)},
		{"SelectedVerticalIntention", discard(airPos.SelectedVerticalIntention)},
		{"TrackAndTurn", discard(airPos.TrackAndTurn)},
		{"HeadingAndSpeed", discard(airPos.HeadingAndSpeed)},
		{"CommonUsageGICB", discard(airPos.CommonUsageGICB)},
		{"RegistrationMarkings", discard(airPos.RegistrationMarkings)},
		{"EmergencyPriorityStatus", discard(airPos.EmergencyPriorityStatus)},
		{"InferBDS", discard(airPos.InferBDS)},
	}

	for _, c := range cases {
		requireNotAvailableFree(t, c.name, c.fn)
	}
}

// testNotAvailableWrapped documents the paths this optimisation deliberately
// does not make free. A decoder whose guard accessor fails wraps that failure,
// and wrapping a dynamic error boxes it. One allocation is the floor; the
// assertion fails loudly if a future change makes such a path worse.
func testNotAvailableWrapped(t *testing.T) {
	surveillance := naMessage(t, naSurveillance)
	airPos := naMessage(t, naAirPos)
	velocity := naMessage(t, naVelocity)

	cases := []struct {
		name string
		fn   func() error
	}{
		// ESType fails on a DF4 reply and Velocity wraps it.
		{"Velocity", discard(surveillance.Velocity)},

		// CF fails on a DF17 message and both of these wrap it.
		{"IMF", discard(airPos.IMF)},
		{"TISBCoarsePosition", discard(airPos.TISBCoarsePosition)},

		// ESAltitude fails on type code 19 and Alt wraps it.
		{"Alt", discard(velocity.Alt)},
	}

	for _, c := range cases {
		var err error

		allocs := testing.AllocsPerRun(allocRuns, func() {
			err = c.fn()
		})

		if err == nil {
			t.Errorf("%s  expected an error, received nil", c.name)

			continue
		}

		if !errors.Is(err, adsb.ErrNotAvailable) {
			t.Errorf("%s  expected an error wrapping ErrNotAvailable, received: %v", c.name, err)
		}

		if allocs > 1 {
			t.Errorf("%s  allocated %v times, want at most 1", c.name, allocs)
		}
	}
}

// discard adapts a decoder to a function returning only its error, so that
// decoders with unrelated result types share one table.
func discard[T any](fn func() (T, error)) func() error {
	return func() error {
		_, err := fn()

		return err
	}
}

// assertNotAvailableFree runs fn and, when it reports a not-available error,
// asserts that producing that error allocates nothing. It reports whether it
// measured one. A call that succeeds is skipped rather than failed: the field
// is carried by that message, so there is no not-available branch to measure.
// Use it only where a skip is expected, and pair it with a check that every
// accessor was measured somewhere; use requireNotAvailableFree otherwise.
func assertNotAvailableFree(t *testing.T, name string, fn func() error) bool {
	t.Helper()

	var err error

	allocs := testing.AllocsPerRun(allocRuns, func() {
		err = fn()
	})

	if err == nil || !errors.Is(err, adsb.ErrNotAvailable) {
		return false
	}

	if allocs != 0 {
		t.Errorf("%s  allocated %v times, want 0", name, allocs)
	}

	return true
}

// requireNotAvailableFree asserts that fn reports a not-available error and
// that producing it allocates nothing. It fails when the call succeeds or
// reports another kind of error, so a case that is specified to reach a
// pre-built error cannot quietly stop measuring one.
func requireNotAvailableFree(t *testing.T, name string, fn func() error) {
	t.Helper()

	var err error

	allocs := testing.AllocsPerRun(allocRuns, func() {
		err = fn()
	})

	switch {
	case err == nil:
		t.Errorf("%s  expected a not-available error, received nil", name)
	case !errors.Is(err, adsb.ErrNotAvailable):
		t.Errorf("%s  expected an error wrapping ErrNotAvailable, received: %v", name, err)
	case allocs != 0:
		t.Errorf("%s  allocated %v times, want 0", name, allocs)
	}
}
