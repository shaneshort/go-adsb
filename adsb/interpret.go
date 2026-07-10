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

package adsb

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"kreklow.us/go/go-adsb/adsbtype"
	"kreklow.us/go/go-adsb/beast"
)

// Beast frame type bytes.
const (
	beastModeAC     = 0x31 // type 1: Mode A/C
	beastModeSShort = 0x32 // type 2: Mode S short
	beastModeSLong  = 0x33 // type 3: Mode S long
)

// referenceLen is the required length of an InterpretOptions reference position.
const referenceLen = 2

// Coordinate bounds for validating a reference position, in degrees.
const (
	maxLatitude  = 90
	maxLongitude = 180
)

// DF18 control fields whose address interpretation does not depend on the IMF
// flag.
const (
	adsbICAOCF      = 0 // ADS-B message with a 24-bit ICAO address
	adsbAnonymousCF = 1 // ADS-B message with an anonymous, non-ICAO address
)

// tisbManagementCF is the DF18 control field of a TIS-B/ADS-R management
// message, whose ME contents are undefined by DO-260B.
const tisbManagementCF = 4

// InterpretOptions configures the single-frame interpretation layer.
type InterpretOptions struct {
	// Reference is an optional receiver/reference position as [lat, lon]
	// degrees. When present, local CPR positions are resolved where possible.
	Reference []float64

	// InferCommB enables heuristic BDS inference for DF20/21 replies. Inferred
	// registers are reported as candidates, never as facts.
	InferCommB bool

	// IncludeRaw keeps the raw payload hex in the result metadata.
	IncludeRaw bool
}

// MessageFamily is a high-level classification of a message.
type MessageFamily uint8

// Message families.
const (
	FamilyUnknown MessageFamily = iota
	FamilyModeAC
	FamilySurveillance
	FamilyADSB
	FamilyTISB
	FamilyADSR
	FamilyCommB
	FamilyCommD
	FamilyUnsupported
)

// String returns a human-readable name for the message family.
func (f MessageFamily) String() string {
	switch f {
	case FamilyUnknown:
		return "Unknown"
	case FamilyModeAC:
		return "Mode A/C"
	case FamilySurveillance:
		return "Surveillance"
	case FamilyADSB:
		return "ADS-B"
	case FamilyTISB:
		return "TIS-B"
	case FamilyADSR:
		return "ADS-R"
	case FamilyCommB:
		return "Comm-B"
	case FamilyCommD:
		return "Comm-D"
	case FamilyUnsupported:
		return "Unsupported"
	default:
		return fmt.Sprintf("MessageFamily(%d)", uint8(f))
	}
}

// AddressKind describes how the 24-bit address field of a message is to be
// interpreted.
type AddressKind uint8

// Address kinds.
const (
	AddressUnknown AddressKind = iota // the interpretation is undetermined
	AddressICAO                       // a 24-bit ICAO aircraft address
	AddressNonICAO                    // an anonymous or Mode A + track file address
)

// String returns a human-readable name for the address kind.
func (k AddressKind) String() string {
	switch k {
	case AddressUnknown:
		return "unknown"
	case AddressICAO:
		return "ICAO"
	case AddressNonICAO:
		return "non-ICAO"
	default:
		return fmt.Sprintf("AddressKind(%d)", uint8(k))
	}
}

// Confidence reports how certain a Comm-B register identification is.
type Confidence uint8

// Confidence levels.
const (
	ConfidenceKnown     Confidence = iota // the register self-identifies
	ConfidenceInferred                    // a single plausible register
	ConfidenceCandidate                   // one of several plausible registers
)

// String returns a human-readable name for the confidence level.
func (c Confidence) String() string {
	switch c {
	case ConfidenceKnown:
		return "Known"
	case ConfidenceInferred:
		return "Inferred"
	case ConfidenceCandidate:
		return "Candidate"
	default:
		return fmt.Sprintf("Confidence(%d)", uint8(c))
	}
}

// WarningCode categorises a Warning.
type WarningCode uint8

// Warning codes.
const (
	WarningUnsupported WarningCode = iota
	WarningAmbiguous
	WarningMalformed
	WarningPositionReferenceRequired
	WarningDecodeFailed
)

// String returns a human-readable name for the warning code.
func (c WarningCode) String() string {
	switch c {
	case WarningUnsupported:
		return "Unsupported"
	case WarningAmbiguous:
		return "Ambiguous"
	case WarningMalformed:
		return "Malformed"
	case WarningPositionReferenceRequired:
		return "PositionReferenceRequired"
	case WarningDecodeFailed:
		return "DecodeFailed"
	default:
		return fmt.Sprintf("WarningCode(%d)", uint8(c))
	}
}

// Warning records a non-fatal issue encountered while interpreting a message.
type Warning struct {
	Code    WarningCode
	Message string
	Err     error
}

// Candidate is a plausible but unconfirmed Comm-B register identification from
// heuristic inference. Payload is nil: candidate registers are not decoded
// blindly, since more than one may match.
type Candidate struct {
	BDS        adsbtype.BDS
	Confidence Confidence
	Payload    Observation
	Reason     string
}

// Meta holds message metadata. Pointer fields are nil when the value is not
// known or not applicable to the message.
type Meta struct {
	RawHex    string
	BeastType byte

	Timestamp *time.Duration
	Signal    *uint8

	DF *adsbtype.DF
	TC *uint8
	CF *adsbtype.CF

	// Address is the 24-bit aircraft address carried by the message. It is nil
	// when the message carries no aircraft address (a DF18 management or
	// reserved control field). AddressKind reports how to interpret it; IMF is
	// the ICAO/Mode A flag for TIS-B and ADS-R messages that define it. ICAO is
	// set only when the address is a genuine 24-bit ICAO address, so callers can
	// rely on a non-nil ICAO being a real aircraft address.
	Address     *uint64
	AddressKind AddressKind
	IMF         *bool
	ICAO        *uint64

	ModeAC *ModeAC
}

// Interpretation is the classified result of interpreting a single frame.
type Interpretation struct {
	Meta         Meta
	Family       MessageFamily
	Observations []Observation
	Candidates   []Candidate
	Warnings     []Warning
}

// observe appends an observation.
func (in *Interpretation) observe(o Observation) {
	in.Observations = append(in.Observations, o)
}

// warn appends a warning.
func (in *Interpretation) warn(code WarningCode, msg string, err error) {
	in.Warnings = append(in.Warnings, Warning{Code: code, Message: msg, Err: err})
}

// probe runs a decoder and dispatches on the result: on success it calls add,
// on ErrNotAvailable it does nothing (a field not carried by this message is
// normal control flow), and on any other error it records a decode-failed
// warning.
func probe[T any](in *Interpretation, name string, fn func() (T, error), add func(T)) {
	v, err := fn()

	switch {
	case err == nil:
		add(v)
	case errors.Is(err, ErrNotAvailable):
	default:
		in.warn(WarningDecodeFailed, name, err)
	}
}

// probeField runs a decoder and returns a pointer to the value, or nil when the
// field is not available.
func probeField[T any](fn func() (T, error)) *T {
	v, err := fn()
	if err != nil {
		return nil
	}

	return &v
}

// probePtr runs a decoder that already returns a pointer, returning nil when the
// field is not available.
func probePtr[T any](fn func() (*T, error)) *T {
	v, err := fn()
	if err != nil {
		return nil
	}

	return v
}

// InterpretModeS interprets a single raw Mode S payload.
func InterpretModeS(data []byte, opts InterpretOptions) (*Interpretation, error) {
	msg := new(Message)

	err := msg.UnmarshalBinary(data)
	if err != nil && !errors.Is(err, ErrUnsupported) {
		return nil, newError(err, "error interpreting Mode S data")
	}

	// An unsupported downlink format still stores the raw message, so its
	// metadata can be recovered and the format reported as unsupported.
	in, err := InterpretMessage(msg, opts)
	if err != nil {
		return nil, err
	}

	if opts.IncludeRaw {
		in.Meta.RawHex = hex.EncodeToString(data)
	}

	return in, nil
}

// InterpretMessage interprets an already-parsed Mode S message.
func InterpretMessage(msg *Message, opts InterpretOptions) (*Interpretation, error) {
	if msg == nil {
		return nil, newError(nil, "nil message")
	}

	df, err := msg.raw.DF()
	if err != nil {
		return nil, newError(err, "error interpreting message")
	}

	in := &Interpretation{Family: FamilyUnknown}

	dfv := adsbtype.DF(df)
	in.Meta.DF = &dfv
	interpretAddress(msg, df, in)

	if len(opts.Reference) != 0 && !referenceValid(opts.Reference) {
		in.warn(WarningMalformed,
			"reference position must be [latitude, longitude] within valid coordinate ranges", nil)
	}

	switch df {
	case 17, 18:
		interpretExtendedSquitter(msg, df, opts, in)
	case 0, 4, 5, 11, 16:
		in.Family = FamilySurveillance
		interpretSurveillance(msg, in)
	case 20, 21:
		in.Family = FamilyCommB
		interpretSurveillance(msg, in)
		interpretCommB(msg, opts, in)
	case 24:
		in.Family = FamilyCommD
	default:
		in.Family = FamilyUnsupported
	}

	return in, nil
}

// interpretAddress records the message address and classifies it. For DF18 the
// address is not always a 24-bit ICAO address: the anonymous and non-ICAO
// control fields, and IMF=1 TIS-B/ADS-R messages, carry a Mode A code with a
// track file number instead. ICAO is populated only when the address is
// genuinely ICAO, so it is never a mislabelled non-ICAO value.
func interpretAddress(msg *Message, df uint64, in *Interpretation) {
	addr, err := msg.ICAO()
	if err != nil {
		return
	}

	in.Meta.IMF = probeField(msg.IMF)

	kind := addressKind(msg, df)
	in.Meta.AddressKind = kind

	// A management or reserved control field carries no aircraft address, so the
	// raw value is not exposed as one.
	if kind == AddressUnknown {
		return
	}

	in.Meta.Address = &addr

	if kind == AddressICAO {
		icao := addr
		in.Meta.ICAO = &icao
	}
}

// addressKind classifies the address field of a message. Every downlink format
// other than DF18 carries a 24-bit ICAO address; DF18 depends on its control
// field and, for the TIS-B and ADS-R formats, the IMF flag.
func addressKind(msg *Message, df uint64) AddressKind {
	if df != 18 {
		return AddressICAO
	}

	cf, err := msg.raw.CF()
	if err != nil {
		return AddressUnknown
	}

	switch cf {
	case adsbICAOCF:
		return AddressICAO
	case adsbAnonymousCF, tisbFineNonICAOCF:
		return AddressNonICAO
	case tisbFineCF, tisbCoarseCF, adsrCF:
		return imfAddressKind(msg)
	default: // management (control field 4) and reserved control fields
		return AddressUnknown
	}
}

// imfAddressKind classifies a TIS-B or ADS-R address from its IMF flag: IMF=0 is
// a 24-bit ICAO address, IMF=1 is a Mode A code with a track file number. The
// address kind is undetermined when the message type carries no IMF flag.
func imfAddressKind(msg *Message) AddressKind {
	imf, err := msg.IMF()
	if err != nil {
		return AddressUnknown
	}

	if imf {
		return AddressNonICAO
	}

	return AddressICAO
}

// referenceValid reports whether a reference position is a [latitude, longitude]
// pair within valid coordinate ranges.
func referenceValid(ref []float64) bool {
	return len(ref) == referenceLen &&
		ref[0] >= -maxLatitude && ref[0] <= maxLatitude &&
		ref[1] >= -maxLongitude && ref[1] <= maxLongitude
}

// interpretExtendedSquitter interprets a DF17/18 extended squitter, dispatching
// on the type code. DF18 refines the family from its control field.
func interpretExtendedSquitter(msg *Message, df uint64, opts InterpretOptions, in *Interpretation) {
	in.Family = FamilyADSB

	if df == 18 {
		interpretControlField(msg, in)
	}

	tc, err := msg.raw.ESType()
	if err != nil {
		// DF18 control field 3 (coarse TIS-B) and 4 (management) have no
		// standard type code.
		interpretTISBCoarse(msg, opts, in)

		return
	}

	tcv := asU8(tc)
	in.Meta.TC = &tcv

	switch {
	case tc >= 1 && tc <= 4:
		probe(in, "callsign", msg.Call, func(s string) {
			if s != "" {
				in.observe(CallsignObservation{Callsign: s})
			}
		})
		probe(in, "category", msg.Category, func(c adsbtype.AcCat) {
			in.observe(CategoryObservation{Category: c})
		})
	case tc >= surfacePosTypeLo && tc <= surfacePosTypeHi:
		interpretCPR(msg, opts, in)
		probe(in, "surface movement", msg.SurfaceMovement, func(s *SurfaceMovement) {
			in.observe(SurfaceMovementObservation{SurfaceMovement: s})
		})
	case (tc >= airPosTypeLo && tc <= airPosTypeHi) || (tc >= gnssPosTypeLo && tc <= gnssPosTypeHi):
		interpretCPR(msg, opts, in)
		interpretAltitude(msg, in)
	case tc == velocityType:
		probe(in, "velocity", msg.Velocity, func(v *Velocity) {
			in.observe(VelocityObservation{Velocity: v})
		})
	case tc == aircraftStatusType:
		probe(in, "aircraft status", msg.AircraftStatus, func(a *AircraftStatus) {
			in.observe(AircraftStatusObservation{AircraftStatus: a})
		})
	case tc == targetStateType:
		probe(in, "target state", msg.TargetState, func(t *TargetState) {
			in.observe(TargetStateObservation{TargetState: t})
		})
	case tc == opStatusType:
		probe(in, "operational status", msg.OperationalStatus, func(o *OperationalStatus) {
			in.observe(OperationalStatusObservation{OperationalStatus: o})
		})
	}
}

// interpretControlField records the DF18 control field and refines the family.
func interpretControlField(msg *Message, in *Interpretation) {
	cf, err := msg.raw.CF()
	if err != nil {
		return
	}

	cfv := adsbtype.CF(cf)
	in.Meta.CF = &cfv
	in.Family = tisbFamily(cf)
}

// tisbFamily maps a DF18 control field to a message family.
func tisbFamily(cf uint64) MessageFamily {
	switch cf {
	case tisbFineCF, tisbCoarseCF, tisbManagementCF, tisbFineNonICAOCF:
		return FamilyTISB
	case adsrCF:
		return FamilyADSR
	default:
		return FamilyADSB
	}
}

// interpretTISBCoarse probes for a TIS-B coarse position (DF18 CF3).
func interpretTISBCoarse(msg *Message, opts InterpretOptions, in *Interpretation) {
	probe(in, "TIS-B coarse position", msg.TISBCoarsePosition, func(p *TISBCoarsePosition) {
		in.Family = FamilyTISB
		in.observe(TISBCoarseObservation{TISBCoarsePosition: p})
		interpretCPRValue(p.CPR, opts, in)
	})
}

// interpretCPR probes for a compact position report.
func interpretCPR(msg *Message, opts InterpretOptions, in *Interpretation) {
	probe(in, "position", msg.CPR, func(c *CPR) {
		interpretCPRValue(c, opts, in)
	})
}

// interpretCPRValue records a CPR observation, resolving the local position
// when a reference is available.
func interpretCPRValue(c *CPR, opts InterpretOptions, in *Interpretation) {
	obs := CPRObservation{CPR: c}

	if referenceValid(opts.Reference) {
		pos, err := c.DecodeLocal(opts.Reference)
		if err == nil {
			obs.LocalPosition = pos
		}
	} else if c.Surface && len(opts.Reference) == 0 {
		// A surface position spans only a 90-degree zone and cannot be located
		// without a reference; an airborne position can be paired globally
		// through the lower-level API instead. An invalid (rather than absent)
		// reference is already reported once as WarningMalformed, so it is not
		// repeated here.
		in.warn(WarningPositionReferenceRequired,
			"a reference position is required to decode a surface position locally", nil)
	}

	in.observe(obs)
}

// interpretAltitude probes for an altitude and its source.
func interpretAltitude(msg *Message, in *Interpretation) {
	probe(in, "altitude", msg.Alt, func(alt int64) {
		src, serr := msg.AltitudeSource()
		if serr != nil {
			src = AltitudeBarometric
		}

		in.observe(AltitudeObservation{Altitude: alt, Source: src})
	})
}

// interpretSurveillance records the Mode S surveillance and protocol fields, the
// altitude and the Mode A identity carried by a message.
func interpretSurveillance(msg *Message, in *Interpretation) {
	obs := SurveillanceObservation{
		FlightStatus:     probeField(msg.FlightStatus),
		DownlinkRequest:  probeField(msg.DownlinkRequest),
		UtilityMessage:   probePtr(msg.UtilityMessage),
		VerticalStatus:   probeField(msg.VerticalStatus),
		SensitivityLevel: probeField(msg.SensitivityLevel),
		ReplyInformation: probeField(msg.ReplyInformation),
		Capability:       probeField(msg.Capability),
		CrossLinkCapable: probeField(msg.CrossLinkCapability),
	}

	if obs != (SurveillanceObservation{}) {
		in.observe(obs)
	}

	interpretAltitude(msg, in)

	probe(in, "identity", msg.Sqk, func(sqk []byte) {
		in.observe(IdentityObservation{Squawk: sqk})
	})
}

// interpretCommB classifies a DF20/21 Comm-B reply. Self-identifying registers
// become known observations; heuristic inference (when enabled) produces
// candidates rather than facts.
func interpretCommB(msg *Message, opts InterpretOptions, in *Interpretation) {
	if commBSelfIdentified(msg, in) {
		return
	}

	if !opts.InferCommB {
		return
	}

	candidates, err := msg.InferBDS()
	if err != nil {
		return
	}

	for _, bds := range candidates {
		in.Candidates = append(in.Candidates, Candidate{
			BDS:        bds,
			Confidence: ConfidenceCandidate,
			Reason:     "matched BDS inference heuristics",
		})
	}

	if len(candidates) > 1 {
		in.warn(WarningAmbiguous, "multiple Comm-B register candidates", nil)
	}
}

// commBSelfIdentified emits a known Comm-B observation when the MB field carries
// a self-identifying register code in its first eight bits, returning whether
// one was found.
func commBSelfIdentified(msg *Message, in *Interpretation) bool {
	switch msg.raw.mbbits(1, 8) {
	case bds10Code:
		probe(in, "data link capability", msg.DataLinkCapability, func(dl *DataLinkCapability) {
			in.observe(CommBObservation{
				BDS: adsbtype.BDS10, Confidence: ConfidenceKnown,
				Payload: DataLinkCapabilityObservation{DataLinkCapability: dl},
			})
		})
	case bds20Code:
		probe(in, "callsign", msg.Call, func(s string) {
			in.observe(CommBObservation{
				BDS: adsbtype.BDS20, Confidence: ConfidenceKnown,
				Payload: CallsignObservation{Callsign: s},
			})
		})
	case bds30Code:
		probe(in, "ACAS resolution advisory", msg.ACASRA, func(ra *ACASRA) {
			in.observe(CommBObservation{
				BDS: adsbtype.BDS30, Confidence: ConfidenceKnown,
				Payload: ACASRAObservation{ACASRA: ra},
			})
		})
	case bdsE7Code:
		probe(in, "transponder status", msg.TransponderStatus, func(ts *TransponderStatus) {
			in.observe(CommBObservation{
				BDS: adsbtype.BDSE7, Confidence: ConfidenceKnown,
				Payload: TransponderStatusObservation{TransponderStatus: ts},
			})
		})
	default:
		return false
	}

	return true
}

// InterpretBeastFrame interprets a single Beast frame, preserving its metadata
// (type, timestamp and signal level) alongside the decoded message.
func InterpretBeastFrame(frame *beast.Frame, opts InterpretOptions) (*Interpretation, error) {
	if frame == nil {
		return nil, newError(nil, "nil frame")
	}

	ft, err := frame.Type()
	if err != nil {
		return nil, newError(err, "error interpreting Beast frame")
	}

	in, err := interpretBeastPayload(frame, ft, opts)
	if err != nil {
		return nil, err
	}

	in.Meta.BeastType = ft

	ts, terr := frame.Timestamp()
	if terr == nil {
		in.Meta.Timestamp = &ts
	}

	sig, serr := frame.Signal()
	if serr == nil {
		in.Meta.Signal = &sig
	}

	return in, nil
}

// interpretBeastPayload interprets the message payload of a Beast frame
// according to its type.
func interpretBeastPayload(frame *beast.Frame, ft byte, opts InterpretOptions) (*Interpretation, error) {
	switch ft {
	case beastModeAC:
		data, err := frame.ModeAC()
		if err != nil {
			return nil, newError(err, "error interpreting Mode A/C frame")
		}

		ma, err := DecodeModeAC(data)
		if err != nil {
			return nil, newError(err, "error interpreting Mode A/C frame")
		}

		in := &Interpretation{Family: FamilyModeAC}
		in.Meta.ModeAC = ma

		if opts.IncludeRaw {
			in.Meta.RawHex = hex.EncodeToString(data)
		}

		return in, nil
	case beastModeSShort, beastModeSLong:
		data, err := frame.ModeS()
		if err != nil {
			return nil, newError(err, "error interpreting Mode S frame")
		}

		return InterpretModeS(data, opts)
	default:
		return &Interpretation{Family: FamilyUnsupported}, nil
	}
}
