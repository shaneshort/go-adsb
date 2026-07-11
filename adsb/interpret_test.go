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
	"kreklow.us/go/go-adsb/adsbtype"
)

// interpret is a test helper that interprets a hex Mode S payload.
func interpret(t *testing.T, s string, opts adsb.InterpretOptions) *adsb.Interpretation {
	t.Helper()

	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex.DecodeString(%q): %v", s, err)
	}

	in, err := adsb.InterpretModeS(b, opts)
	if err != nil {
		t.Fatalf("InterpretModeS: %v", err)
	}

	return in
}

// observation returns the first observation of the given kind, or nil.
//
//nolint:ireturn // a test helper that locates an observation by kind
func observation(in *adsb.Interpretation, kind adsb.ObservationKind) adsb.Observation {
	for _, o := range in.Observations {
		if o.ObservationKind() == kind {
			return o
		}
	}

	return nil
}

// An ADS-B identification message produces callsign and category observations.
func TestInterpretIdentification(t *testing.T) {
	in := interpret(t, "8D4840D6202CC371C32CE0576098", adsb.InterpretOptions{})

	if in.Family != adsb.FamilyADSB {
		t.Errorf("Family = %v, want ADSB", in.Family)
	}

	cs, ok := observation(in, adsb.KindCallsign).(adsb.CallsignObservation)
	if !ok {
		t.Fatal("no callsign observation")
	}

	if cs.Callsign != "KLM1023" {
		t.Errorf("Callsign = %q, want %q", cs.Callsign, "KLM1023")
	}

	if observation(in, adsb.KindCategory) == nil {
		t.Error("no category observation")
	}
}

// An ADS-B velocity message produces a velocity observation.
func TestInterpretVelocity(t *testing.T) {
	in := interpret(t, "8D485020994409940838175B284F", adsb.InterpretOptions{})

	if observation(in, adsb.KindVelocity) == nil {
		t.Error("no velocity observation")
	}
}

// An airborne position produces a CPR observation, and a warning that a
// reference is required for local decoding when none is supplied.
func TestInterpretAirbornePosition(t *testing.T) {
	in := interpret(t, "8D40621D58C382D690C8AC2863A7", adsb.InterpretOptions{})

	cpr, ok := observation(in, adsb.KindCPR).(adsb.CPRObservation)
	if !ok {
		t.Fatal("no CPR observation")
	}

	if cpr.LocalPosition != nil {
		t.Error("LocalPosition should be nil without a reference")
	}

	if observation(in, adsb.KindAltitude) == nil {
		t.Error("no altitude observation")
	}

	// An airborne position does not require a reference (it can be paired
	// globally through the lower-level API), so no warning is expected.
	if hasWarning(in, adsb.WarningPositionReferenceRequired) {
		t.Error("unexpected position-reference-required warning for an airborne position")
	}
}

// A reference position resolves the local CPR position.
func TestInterpretPositionWithReference(t *testing.T) {
	in := interpret(t, "8D40621D58C382D690C8AC2863A7",
		adsb.InterpretOptions{Reference: []float64{52, 4}})

	cpr, ok := observation(in, adsb.KindCPR).(adsb.CPRObservation)
	if !ok {
		t.Fatal("no CPR observation")
	}

	if len(cpr.LocalPosition) != 2 {
		t.Fatalf("LocalPosition = %v, want a [lat, lon] pair", cpr.LocalPosition)
	}
}

// A surface message produces a CPR observation and a surface movement
// observation.
func TestInterpretSurface(t *testing.T) {
	in := interpret(t, "8C4841753A9A153237AEF0F275BE", adsb.InterpretOptions{})

	if observation(in, adsb.KindCPR) == nil {
		t.Error("no CPR observation")
	}

	if observation(in, adsb.KindSurfaceMovement) == nil {
		t.Error("no surface movement observation")
	}

	// A surface position cannot be located without a reference.
	if !hasWarning(in, adsb.WarningPositionReferenceRequired) {
		t.Error("expected a position-reference-required warning for a surface position")
	}
}

// A surveillance reply classifies as the surveillance family with a
// surveillance observation.
func TestInterpretSurveillance(t *testing.T) {
	in := interpret(t, "5D000000000000", adsb.InterpretOptions{}) // DF11, CA 5

	if in.Family != adsb.FamilySurveillance {
		t.Errorf("Family = %v, want Surveillance", in.Family)
	}

	sv, ok := observation(in, adsb.KindSurveillance).(adsb.SurveillanceObservation)
	if !ok {
		t.Fatal("no surveillance observation")
	}

	if sv.Capability == nil {
		t.Error("expected a capability field on a DF11 reply")
	}
}

// A self-identifying Comm-B register becomes a known observation.
func TestInterpretCommBKnown(t *testing.T) {
	in := interpret(t, "A0000000200420C4820820000000", adsb.InterpretOptions{})

	if in.Family != adsb.FamilyCommB {
		t.Errorf("Family = %v, want CommB", in.Family)
	}

	cb, ok := observation(in, adsb.KindCommB).(adsb.CommBObservation)
	if !ok {
		t.Fatal("no Comm-B observation")
	}

	if cb.Confidence != adsb.ConfidenceKnown {
		t.Errorf("Confidence = %v, want Known", cb.Confidence)
	}

	if len(in.Candidates) != 0 {
		t.Errorf("a self-identifying register should not produce candidates: %v", in.Candidates)
	}
}

// An ambiguous Comm-B reply produces candidates, not facts, only when inference
// is enabled.
func TestInterpretCommBCandidates(t *testing.T) {
	// BDS 4,0 does not self-identify; without inference it stays unclassified.
	off := interpret(t, "A0000000BE85F430A40185000000", adsb.InterpretOptions{})
	if len(off.Candidates) != 0 || observation(off, adsb.KindCommB) != nil {
		t.Error("no inference requested, expected no candidates or Comm-B observation")
	}

	on := interpret(t, "A0000000BE85F430A40185000000", adsb.InterpretOptions{InferCommB: true})
	if len(on.Candidates) == 0 {
		t.Fatal("expected inferred Comm-B candidates")
	}

	for _, c := range on.Candidates {
		if c.Confidence == adsb.ConfidenceKnown {
			t.Errorf("candidate %v marked Known; inference is never a fact", c.BDS)
		}
	}
}

// A TIS-B coarse position (DF18 CF3) classifies as the TIS-B family with a
// coarse observation.
func TestInterpretTISBCoarse(t *testing.T) {
	in := interpret(t, "9340621D0B8714143E87D0000000", adsb.InterpretOptions{})

	if in.Family != adsb.FamilyTISB {
		t.Errorf("Family = %v, want TISB", in.Family)
	}

	if observation(in, adsb.KindTISBCoarse) == nil {
		t.Error("no TIS-B coarse observation")
	}
}

// ErrNotAvailable from probing inapplicable decoders is normal control flow and
// must not appear as warning noise.
func TestInterpretNoWarningNoise(t *testing.T) {
	in := interpret(t, "8D485020994409940838175B284F", adsb.InterpretOptions{})

	if len(in.Warnings) != 0 {
		t.Errorf("expected no warnings, got %+v", in.Warnings)
	}
}

// An unsupported downlink format still yields an interpretation with the
// unsupported family and recoverable metadata.
func TestInterpretUnsupportedDF(t *testing.T) {
	in := interpret(t, "98000000000000000000000000AA", adsb.InterpretOptions{}) // DF19 military

	if in.Family != adsb.FamilyUnsupported {
		t.Errorf("Family = %v, want Unsupported", in.Family)
	}

	if in.Meta.DF == nil {
		t.Error("DF metadata should still be recovered for an unsupported format")
	}
}

// A malformed payload returns an error.
func TestInterpretMalformed(t *testing.T) {
	_, err := adsb.InterpretModeS([]byte{0x8d, 0x00}, adsb.InterpretOptions{})
	if err == nil {
		t.Error("expected an error for a truncated payload")
	}
}

// A DF17 extended squitter carries a genuine 24-bit ICAO address.
func TestInterpretAddressICAO(t *testing.T) {
	in := interpret(t, "8D4840D6202CC371C32CE0576098", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressICAO {
		t.Errorf("AddressKind = %v, want ICAO", in.Meta.AddressKind)
	}

	if in.Meta.ICAO == nil || *in.Meta.ICAO != 0x4840D6 {
		t.Errorf("ICAO = %v, want 0x4840D6", in.Meta.ICAO)
	}

	if in.Meta.Address == nil || *in.Meta.Address != 0x4840D6 {
		t.Errorf("Address = %v, want 0x4840D6", in.Meta.Address)
	}
}

// A DF18 CF0 message uses an ICAO address, which is retained.
func TestInterpretAddressDF18ICAO(t *testing.T) {
	in := interpret(t, "904840D6202CC371C32CE0576098", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressICAO {
		t.Errorf("AddressKind = %v, want ICAO", in.Meta.AddressKind)
	}

	if in.Meta.ICAO == nil || *in.Meta.ICAO != 0x4840D6 {
		t.Errorf("ICAO = %v, want 0x4840D6", in.Meta.ICAO)
	}
}

// A DF18 CF1 message carries an anonymous, non-ICAO address; the interpreter
// must not label it as an ICAO address.
func TestInterpretAddressAnonymous(t *testing.T) {
	in := interpret(t, "914840D6202CC371C32CE0576098", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressNonICAO {
		t.Errorf("AddressKind = %v, want NonICAO", in.Meta.AddressKind)
	}

	if in.Meta.ICAO != nil {
		t.Errorf("ICAO = %v, want nil for a non-ICAO address", in.Meta.ICAO)
	}

	if in.Meta.Address == nil {
		t.Error("Address should still hold the raw address value")
	}
}

// A DF18 CF5 fine TIS-B message carries a non-ICAO address.
func TestInterpretAddressNonICAOTISB(t *testing.T) {
	in := interpret(t, "954840D6202CC371C32CE0576098", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressNonICAO {
		t.Errorf("AddressKind = %v, want NonICAO", in.Meta.AddressKind)
	}

	if in.Meta.ICAO != nil {
		t.Errorf("ICAO = %v, want nil", in.Meta.ICAO)
	}
}

// A DF18 CF3 coarse TIS-B message with IMF=0 uses an ICAO address.
func TestInterpretAddressCoarseICAO(t *testing.T) {
	in := interpret(t, "9340621D0B8714143E87D0000000", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressICAO {
		t.Errorf("AddressKind = %v, want ICAO", in.Meta.AddressKind)
	}

	if in.Meta.ICAO == nil || *in.Meta.ICAO != 0x40621D {
		t.Errorf("ICAO = %v, want 0x40621D", in.Meta.ICAO)
	}
}

// A DF18 CF3 coarse TIS-B message with IMF=1 carries a Mode A code and track
// file number, not an ICAO address.
func TestInterpretAddressCoarseNonICAO(t *testing.T) {
	in := interpret(t, "9340621D8B8714143E87D0000000", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressNonICAO {
		t.Errorf("AddressKind = %v, want NonICAO", in.Meta.AddressKind)
	}

	if in.Meta.ICAO != nil {
		t.Errorf("ICAO = %v, want nil for an IMF=1 address", in.Meta.ICAO)
	}
}

// A DF18 CF4 management message carries no aircraft address; its address kind is
// undetermined and it must not be labelled as an ICAO address.
func TestInterpretAddressManagement(t *testing.T) {
	in := interpret(t, "944840D6202CC371C32CE0576098", adsb.InterpretOptions{})

	if in.Meta.AddressKind != adsb.AddressUnknown {
		t.Errorf("AddressKind = %v, want Unknown", in.Meta.AddressKind)
	}

	if in.Meta.ICAO != nil {
		t.Errorf("ICAO = %v, want nil for a management message", in.Meta.ICAO)
	}

	// A management message carries no aircraft address, so the raw address value
	// is left unset rather than exposing the management field as an address.
	if in.Meta.Address != nil {
		t.Errorf("Address = %v, want nil for a management message", in.Meta.Address)
	}
}

// A known BDS 1,0 register carries its decoded data link capability payload.
func TestInterpretCommBPayloadDataLink(t *testing.T) {
	in := interpret(t, "A0000000100309B5DAACE1000000", adsb.InterpretOptions{})

	cb, ok := observation(in, adsb.KindCommB).(adsb.CommBObservation)
	if !ok {
		t.Fatal("no Comm-B observation")
	}

	if cb.BDS != adsbtype.BDS10 {
		t.Errorf("BDS = %v, want BDS10", cb.BDS)
	}

	dl, ok := cb.Payload.(adsb.DataLinkCapabilityObservation)
	if !ok {
		t.Fatalf("Payload = %T, want DataLinkCapabilityObservation", cb.Payload)
	}

	if dl.DataLinkCapability == nil {
		t.Error("decoded data link capability was discarded")
	}
}

// A known BDS 3,0 register carries its decoded ACAS resolution advisory payload.
func TestInterpretCommBPayloadACASRA(t *testing.T) {
	data := []byte{0xA0, 0x00, 0x00, 0x00, 0x30, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	in, err := adsb.InterpretModeS(data, adsb.InterpretOptions{})
	if err != nil {
		t.Fatalf("InterpretModeS: %v", err)
	}

	cb, ok := observation(in, adsb.KindCommB).(adsb.CommBObservation)
	if !ok {
		t.Fatal("no Comm-B observation")
	}

	if cb.BDS != adsbtype.BDS30 {
		t.Errorf("BDS = %v, want BDS30", cb.BDS)
	}

	ra, ok := cb.Payload.(adsb.ACASRAObservation)
	if !ok {
		t.Fatalf("Payload = %T, want ACASRAObservation", cb.Payload)
	}

	if ra.ACASRA == nil {
		t.Error("decoded ACAS RA was discarded")
	}
}

// A known BDS E,7 register carries its decoded transponder status payload.
func TestInterpretCommBPayloadTransponder(t *testing.T) {
	in := interpret(t, "A0000000E769A524C7A967000000", adsb.InterpretOptions{})

	cb, ok := observation(in, adsb.KindCommB).(adsb.CommBObservation)
	if !ok {
		t.Fatal("no Comm-B observation")
	}

	if cb.BDS != adsbtype.BDSE7 {
		t.Errorf("BDS = %v, want BDSE7", cb.BDS)
	}

	ts, ok := cb.Payload.(adsb.TransponderStatusObservation)
	if !ok {
		t.Fatalf("Payload = %T, want TransponderStatusObservation", cb.Payload)
	}

	if ts.TransponderStatus == nil {
		t.Error("decoded transponder status was discarded")
	}
}

// An invalid reference supplied for a surface position is reported only as
// malformed, not also as a missing reference: the user did provide one.
func TestInterpretSurfaceInvalidReference(t *testing.T) {
	in := interpret(t, "8C4841753A9A153237AEF0F275BE",
		adsb.InterpretOptions{Reference: []float64{999, 999}})

	if !hasWarning(in, adsb.WarningMalformed) {
		t.Error("expected a malformed-reference warning")
	}

	if hasWarning(in, adsb.WarningPositionReferenceRequired) {
		t.Error("a supplied (if invalid) reference should not also warn that one is required")
	}
}

// A reference position of the wrong length is reported as malformed.
func TestInterpretReferenceWrongLength(t *testing.T) {
	in := interpret(t, "8D40621D58C382D690C8AC2863A7",
		adsb.InterpretOptions{Reference: []float64{52}})

	if !hasWarning(in, adsb.WarningMalformed) {
		t.Error("expected a malformed-reference warning for a single-element reference")
	}
}

// A reference position outside the valid coordinate range is reported as
// malformed.
func TestInterpretReferenceOutOfRange(t *testing.T) {
	in := interpret(t, "8D40621D58C382D690C8AC2863A7",
		adsb.InterpretOptions{Reference: []float64{999, 999}})

	if !hasWarning(in, adsb.WarningMalformed) {
		t.Error("expected a malformed-reference warning for an out-of-range reference")
	}
}

// Every message family stringifies to its documented name, and an unknown value
// falls back to a diagnostic form.
func TestFamilyString(t *testing.T) {
	cases := map[adsb.MessageFamily]string{
		adsb.FamilyUnknown:      "Unknown",
		adsb.FamilyModeAC:       "Mode A/C",
		adsb.FamilySurveillance: "Surveillance",
		adsb.FamilyADSB:         "ADS-B",
		adsb.FamilyTISB:         "TIS-B",
		adsb.FamilyADSR:         "ADS-R",
		adsb.FamilyCommB:        "Comm-B",
		adsb.FamilyCommD:        "Comm-D",
		adsb.FamilyUnsupported:  "Unsupported",
		adsb.MessageFamily(200): "MessageFamily(200)",
	}

	for f, want := range cases {
		if got := f.String(); got != want {
			t.Errorf("MessageFamily(%d).String() = %q, want %q", f, got, want)
		}
	}
}

// Every confidence level stringifies, including an unknown value.
func TestConfidenceString(t *testing.T) {
	cases := map[adsb.Confidence]string{
		adsb.ConfidenceKnown:     "Known",
		adsb.ConfidenceInferred:  "Inferred",
		adsb.ConfidenceCandidate: "Candidate",
		adsb.Confidence(200):     "Confidence(200)",
	}

	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("Confidence(%d).String() = %q, want %q", c, got, want)
		}
	}
}

// Every warning code stringifies, including an unknown value.
func TestWarningCodeString(t *testing.T) {
	cases := map[adsb.WarningCode]string{
		adsb.WarningUnsupported:               "Unsupported",
		adsb.WarningAmbiguous:                 "Ambiguous",
		adsb.WarningMalformed:                 "Malformed",
		adsb.WarningPositionReferenceRequired: "PositionReferenceRequired",
		adsb.WarningDecodeFailed:              "DecodeFailed",
		adsb.WarningCode(200):                 "WarningCode(200)",
	}

	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("WarningCode(%d).String() = %q, want %q", c, got, want)
		}
	}
}

// Every observation kind stringifies, including an unknown value.
func TestObservationKindString(t *testing.T) {
	cases := map[adsb.ObservationKind]string{
		adsb.KindCallsign:           "Callsign",
		adsb.KindCategory:           "Category",
		adsb.KindAltitude:           "Altitude",
		adsb.KindIdentity:           "Identity",
		adsb.KindVelocity:           "Velocity",
		adsb.KindCPR:                "CPR",
		adsb.KindSurfaceMovement:    "SurfaceMovement",
		adsb.KindAircraftStatus:     "AircraftStatus",
		adsb.KindOperationalStatus:  "OperationalStatus",
		adsb.KindTargetState:        "TargetState",
		adsb.KindSurveillance:       "Surveillance",
		adsb.KindCommB:              "Comm-B",
		adsb.KindTISBCoarse:         "TISBCoarse",
		adsb.KindDataLinkCapability: "DataLinkCapability",
		adsb.KindACASRA:             "ACAS RA",
		adsb.KindTransponderStatus:  "TransponderStatus",
		adsb.ObservationKind(200):   "ObservationKind(200)",
	}

	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("ObservationKind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

// Every address kind stringifies, including an unknown value.
func TestAddressKindString(t *testing.T) {
	cases := map[adsb.AddressKind]string{
		adsb.AddressUnknown:   "unknown",
		adsb.AddressICAO:      "ICAO",
		adsb.AddressNonICAO:   "non-ICAO",
		adsb.AddressKind(200): "AddressKind(200)",
	}

	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("AddressKind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

// hasWarning reports whether the interpretation carries a warning of the code.
func hasWarning(in *adsb.Interpretation, code adsb.WarningCode) bool {
	for _, w := range in.Warnings {
		if w.Code == code {
			return true
		}
	}

	return false
}
