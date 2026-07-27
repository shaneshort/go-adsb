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

// Package adsb decodes Mode S and ADS-B transponder messages.
//
// # Interpreting a message
//
// InterpretModeS, InterpretMessage and InterpretBeastFrame are the entry points
// for a caller that wants a classified result from a single frame. Each returns
// an *Interpretation carrying the message Family, Meta metadata, typed
// Observations, Comm-B Candidates and Warnings. Meta.DF and Meta.TC discriminate
// the message, so the caller does not dispatch on the downlink format and type
// code itself. See ExampleInterpretModeS and ExampleInterpretation_observations.
//
// The layer is stateless: it interprets one frame and holds nothing between
// calls.
//
// # Direct field access
//
// Message and RawMessage are the lower-level API, for a caller that wants named
// fields or arbitrary bit ranges directly. A Message method returns an error
// wrapping ErrNotAvailable when the field is not part of the received message
// format. That is normal control flow rather than a failure, and is checked with
// errors.Is(err, ErrNotAvailable). RawMessage exposes the raw Mode S fields such
// as AA, AC, ME and MB.
//
// # Comm-B
//
// DF20/21 replies carry no register identifier. A register that self-identifies
// is decoded and reported as an Observation with ConfidenceKnown. With
// InterpretOptions.InferCommB set, the remaining registers are inferred: a sole
// match becomes a Candidate with ConfidenceInferred and the register decoded
// into Candidate.Payload, while several matches become Candidates with
// ConfidenceCandidate, a nil Payload and a WarningAmbiguous. An ambiguous
// register is never decoded, since decoding the wrong one produces plausible but
// false values. See ExampleInterpretation_commBCandidates.
//
// # Position
//
// A CPRObservation always carries the encoded position. Local decoding needs a
// reference position supplied through InterpretOptions.Reference; global even
// and odd pair decoding is available through DecodeGlobalPosition. See
// ExampleInterpretModeS_withReference, ExampleCPR_DecodeLocal and
// ExampleDecodeGlobalPositionRef.
//
// # Scope
//
// The interpretation layer excludes DF24 Comm-D/ELM high-level assembly, the
// DF19 and DF22 military formats, vendor and private formats, and UAT (978 MHz).
package adsb

import "fmt"

// Public error variables.
var (
	errNotAvailable = newError(nil, "field not available")
	errUnsupported  = newError(nil, "format unsupported")

	// ErrNotAvailable is used to indicate that a field is not part of the
	// specification for the message format received. Each field error wraps
	// ErrNotAvailable, making it accessible by calling
	// errors.Is(err, adsb.ErrNotAvailable).
	ErrNotAvailable = errNotAvailable

	// ErrUnsupported is returned when the Downlink Format of a message
	// is not supported by Message. The error may be wrapped and should be
	// checked with errors.Is().
	ErrUnsupported = errUnsupported
)

// adsbError is the error type for the adsb library.
type adsbError struct {
	msg  string // error message string from this library
	werr error  // wrapped error from downstream function
}

// Error returns the string value of an error.
func (e adsbError) Error() string {
	if e.werr == nil {
		return e.msg
	}

	return e.msg + ": " + e.werr.Error()
}

// Unwrap returns an underlying error if applicable.
func (e adsbError) Unwrap() error {
	return e.werr
}

// newError returns a new adsbError.
func newError(w error, m string) adsbError {
	return adsbError{
		msg:  m,
		werr: w,
	}
}

// newErrorf returns a new adsbError with a Printf-style message.
func newErrorf(w error, m string, v ...any) adsbError {
	return adsbError{
		msg:  fmt.Sprintf(m, v...),
		werr: w,
	}
}

// notAvailable returns a pre-built error reporting that a named field is not
// carried by the message format. The result is declared as an error, not as an
// adsbError, so that returning it does not box the value: probing a field that
// a message does not carry is the normal outcome of interpreting a frame, not
// an exception, and must not allocate. Every not-available error whose message
// is fixed is built once through this function, at package initialisation.
//
// A field that the message format cannot carry returns one of these pre-built
// errors, because that is normal control flow on every probe. A field the
// format does carry but whose encoding is reserved or unsupported keeps a
// formatted diagnostic built by newErrorf, because the offending value is the
// point of the message and the path is cold.
func notAvailable(field string) error {
	return newError(ErrNotAvailable, "error retrieving "+field)
}
