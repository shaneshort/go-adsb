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
)

// exampleMessage decodes a hex string into a Message, panicking on error. It
// keeps the examples focused on the decoding calls being demonstrated.
func exampleMessage(s string) *adsb.Message {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	msg := new(adsb.Message)

	err = msg.UnmarshalBinary(b)
	if err != nil {
		panic(err)
	}

	return msg
}

// ExampleMessage_SurfaceMovement decodes the ground speed and track from a
// surface position message. Both fields are pointers: a nil pointer means the
// field carried the "no data" sentinel.
func ExampleMessage_SurfaceMovement() {
	msg := exampleMessage("8C4841753A9A153237AEF0F275BE")

	sm, err := msg.SurfaceMovement()
	if err != nil {
		fmt.Println(err)

		return
	}

	if sm.GroundSpeed != nil {
		fmt.Printf("ground speed: %.0f kt\n", *sm.GroundSpeed)
	}

	if sm.Track != nil {
		fmt.Printf("track: %.4f degrees\n", *sm.Track)
	}

	// Output:
	// ground speed: 17 kt
	// track: 92.8125 degrees
}

// ExampleCPR_DecodeLocal decodes a surface position against a nearby
// reference point. A single surface message is resolved against a known
// location such as the receiver; the reference is passed as
// [latitude, longitude] in degrees.
func ExampleCPR_DecodeLocal() {
	msg := exampleMessage("8C4841753A9A153237AEF0F275BE")

	cpr, err := msg.CPR()
	if err != nil {
		fmt.Println(err)

		return
	}

	pos, err := cpr.DecodeLocal([]float64{52.0, 4.0})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Printf("%.5f, %.5f\n", pos[0], pos[1])

	// Output: 52.32056, 4.73574
}

// ExampleDecodeGlobalPositionRef decodes a surface position from an even/odd
// message pair, using a reference point to resolve the ambiguous surface
// zones. The reference is [latitude, longitude] in degrees, typically the
// receiver location. As with DecodeGlobalPosition, the position returned is
// that of the second argument (the more recent frame) — here the odd message.
func ExampleDecodeGlobalPositionRef() {
	even := exampleMessage("8C4841753AAB238733C8CD4020B1")
	odd := exampleMessage("8C4841753A9A153237AEF0F275BE")

	evenCPR, err := even.CPR()
	if err != nil {
		fmt.Println(err)

		return
	}

	oddCPR, err := odd.CPR()
	if err != nil {
		fmt.Println(err)

		return
	}

	pos, err := adsb.DecodeGlobalPositionRef(evenCPR, oddCPR, []float64{52.0, 4.0})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Printf("%.5f, %.5f\n", pos[0], pos[1])

	// Output: 52.32056, 4.73574
}
