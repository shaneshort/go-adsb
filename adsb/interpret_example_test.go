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
	"fmt"

	"kreklow.us/go/go-adsb/adsb"
	"kreklow.us/go/go-adsb/beast"
)

// ExampleInterpretModeS interprets a single hex Mode S payload and prints the
// aircraft identification it carries.
func ExampleInterpretModeS() {
	data, _ := hex.DecodeString("8D4840D6202CC371C32CE0576098")

	in, err := adsb.InterpretModeS(data, adsb.InterpretOptions{})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, o := range in.Observations {
		if cs, ok := o.(adsb.CallsignObservation); ok {
			fmt.Println("callsign:", cs.Callsign)
		}
	}
	// Output: callsign: KLM1023
}

// ExampleInterpretation_observations shows switching over the typed
// observations produced for an airborne position message.
func ExampleInterpretation_observations() {
	data, _ := hex.DecodeString("8D40621D58C382D690C8AC2863A7")

	in, _ := adsb.InterpretModeS(data, adsb.InterpretOptions{})

	fmt.Println("family:", in.Family)

	for _, o := range in.Observations {
		switch obs := o.(type) {
		case adsb.AltitudeObservation:
			fmt.Printf("altitude: %d ft (%s)\n", obs.Altitude, obs.Source)
		case adsb.CPRObservation:
			fmt.Printf("position: CPR format %d (local not resolved: %t)\n",
				obs.CPR.F, obs.LocalPosition == nil)
		}
	}
	// Output:
	// family: ADS-B
	// position: CPR format 0 (local not resolved: true)
	// altitude: 38000 ft (Barometric)
}

// ExampleInterpretModeS_withReference resolves a local position when a
// reference position is supplied.
func ExampleInterpretModeS_withReference() {
	data, _ := hex.DecodeString("8C4841753A9A153237AEF0F275BE") // surface position

	in, _ := adsb.InterpretModeS(data, adsb.InterpretOptions{
		Reference: []float64{52, 4},
	})

	for _, o := range in.Observations {
		if cpr, ok := o.(adsb.CPRObservation); ok && cpr.LocalPosition != nil {
			fmt.Printf("local position: %.4f, %.4f\n",
				cpr.LocalPosition[0], cpr.LocalPosition[1])
		}
	}
	// Output: local position: 52.3206, 4.7357
}

// ExampleInterpretation_commBCandidates shows that heuristic Comm-B inference
// produces candidates rather than facts.
func ExampleInterpretation_commBCandidates() {
	data, _ := hex.DecodeString("A0000000BE85F430A40185000000")

	in, _ := adsb.InterpretModeS(data, adsb.InterpretOptions{InferCommB: true})

	for _, c := range in.Candidates {
		fmt.Printf("candidate: %s (confidence %s)\n", c.BDS, c.Confidence)
	}
	// Output: candidate: Selected vertical intention (confidence Candidate)
}

// ExampleInterpretBeastFrame interprets a Beast frame, preserving its metadata.
func ExampleInterpretBeastFrame() {
	msg, _ := hex.DecodeString("8D4840D6202CC371C32CE0576098")

	raw := make([]byte, 0, 9+len(msg))
	raw = append(raw, 0x1a, 0x33, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x64)
	raw = append(raw, msg...)

	frame := new(beast.Frame)

	err := frame.UnmarshalBinary(raw)
	if err != nil {
		fmt.Println(err)

		return
	}

	in, err := adsb.InterpretBeastFrame(frame, adsb.InterpretOptions{})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Printf("family: %s, signal present: %t\n", in.Family, in.Meta.Signal != nil)
	// Output: family: ADS-B, signal present: true
}
