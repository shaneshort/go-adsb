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
	"fmt"

	"kreklow.us/go/go-adsb/adsbtype"
)

// ObservationKind identifies the concrete type of an Observation, allowing a
// caller to switch on the kind before asserting the concrete type.
type ObservationKind uint8

// Observation kinds.
const (
	KindCallsign ObservationKind = iota
	KindCategory
	KindAltitude
	KindIdentity
	KindVelocity
	KindCPR
	KindSurfaceMovement
	KindAircraftStatus
	KindOperationalStatus
	KindTargetState
	KindSurveillance
	KindCommB
	KindTISBCoarse
	KindDataLinkCapability
	KindACASRA
	KindTransponderStatus
	KindSelectedVerticalIntention
	KindTrackAndTurn
	KindHeadingAndSpeed
	KindCommonUsageGICB
)

// Names of the observation kinds.
var mObservationKind = map[ObservationKind]string{
	KindCallsign:                  "Callsign",
	KindCategory:                  "Category",
	KindAltitude:                  "Altitude",
	KindIdentity:                  "Identity",
	KindVelocity:                  "Velocity",
	KindCPR:                       "CPR",
	KindSurfaceMovement:           "SurfaceMovement",
	KindAircraftStatus:            "AircraftStatus",
	KindOperationalStatus:         "OperationalStatus",
	KindTargetState:               "TargetState",
	KindSurveillance:              "Surveillance",
	KindCommB:                     "Comm-B",
	KindTISBCoarse:                "TISBCoarse",
	KindDataLinkCapability:        "DataLinkCapability",
	KindACASRA:                    "ACAS RA",
	KindTransponderStatus:         "TransponderStatus",
	KindSelectedVerticalIntention: "SelectedVerticalIntention",
	KindTrackAndTurn:              "TrackAndTurn",
	KindHeadingAndSpeed:           "HeadingAndSpeed",
	KindCommonUsageGICB:           "CommonUsageGICB",
}

// String returns a human-readable name for the observation kind.
func (k ObservationKind) String() string {
	if str, ok := mObservationKind[k]; ok {
		return str
	}

	return fmt.Sprintf("ObservationKind(%d)", uint8(k))
}

// Observation is a single decoded fact recovered from a message by the
// interpretation layer. Concrete observation types wrap the existing decoder
// results rather than duplicating their fields; assert the concrete type after
// switching on ObservationKind.
type Observation interface {
	ObservationKind() ObservationKind
}

// CallsignObservation is the aircraft identification (callsign) from an
// identification message or a BDS 2,0 Comm-B reply.
type CallsignObservation struct {
	Callsign string
}

// ObservationKind implements Observation.
func (CallsignObservation) ObservationKind() ObservationKind { return KindCallsign }

// CategoryObservation is the ADS-B emitter category from an identification
// message.
type CategoryObservation struct {
	Category adsbtype.AcCat
}

// ObservationKind implements Observation.
func (CategoryObservation) ObservationKind() ObservationKind { return KindCategory }

// AltitudeObservation is a decoded altitude and its reference frame.
type AltitudeObservation struct {
	Altitude int64
	Source   AltitudeSource
}

// ObservationKind implements Observation.
func (AltitudeObservation) ObservationKind() ObservationKind { return KindAltitude }

// IdentityObservation is a Mode A identity (squawk) code.
type IdentityObservation struct {
	Squawk []byte // four octal digits
}

// ObservationKind implements Observation.
func (IdentityObservation) ObservationKind() ObservationKind { return KindIdentity }

// VelocityObservation is a decoded airborne velocity report.
type VelocityObservation struct {
	*Velocity
}

// ObservationKind implements Observation.
func (VelocityObservation) ObservationKind() ObservationKind { return KindVelocity }

// CPRObservation is an encoded compact position report. LocalPosition holds the
// decoded [latitude, longitude] only when a reference position was supplied to
// the interpreter and the local decode succeeded; the interpretation layer is
// stateless and does not perform global even/odd pair decoding.
type CPRObservation struct {
	CPR           *CPR
	LocalPosition []float64
}

// ObservationKind implements Observation.
func (CPRObservation) ObservationKind() ObservationKind { return KindCPR }

// SurfaceMovementObservation is a decoded surface movement (ground speed and
// track) report.
type SurfaceMovementObservation struct {
	*SurfaceMovement
}

// ObservationKind implements Observation.
func (SurfaceMovementObservation) ObservationKind() ObservationKind { return KindSurfaceMovement }

// AircraftStatusObservation is a decoded aircraft status message (emergency or
// TCAS resolution advisory).
type AircraftStatusObservation struct {
	*AircraftStatus
}

// ObservationKind implements Observation.
func (AircraftStatusObservation) ObservationKind() ObservationKind { return KindAircraftStatus }

// OperationalStatusObservation is a decoded aircraft operational status
// message.
type OperationalStatusObservation struct {
	*OperationalStatus
}

// ObservationKind implements Observation.
func (OperationalStatusObservation) ObservationKind() ObservationKind {
	return KindOperationalStatus
}

// TargetStateObservation is a decoded target state and status message.
type TargetStateObservation struct {
	*TargetState
}

// ObservationKind implements Observation.
func (TargetStateObservation) ObservationKind() ObservationKind { return KindTargetState }

// TISBCoarseObservation is a decoded TIS-B coarse airborne position message.
type TISBCoarseObservation struct {
	*TISBCoarsePosition
}

// ObservationKind implements Observation.
func (TISBCoarseObservation) ObservationKind() ObservationKind { return KindTISBCoarse }

// SurveillanceObservation groups the Mode S surveillance and protocol subfields
// present in a message. Each field is a pointer that is nil when the field is
// not carried by the message's downlink format.
type SurveillanceObservation struct {
	FlightStatus     *adsbtype.FS
	DownlinkRequest  *adsbtype.DR
	UtilityMessage   *UtilityMessage
	VerticalStatus   *adsbtype.VS
	SensitivityLevel *adsbtype.SL
	ReplyInformation *adsbtype.RI
	Capability       *adsbtype.CA
	CrossLinkCapable *adsbtype.CC
}

// ObservationKind implements Observation.
func (SurveillanceObservation) ObservationKind() ObservationKind { return KindSurveillance }

// CommBObservation is a Comm-B register that identified itself through the
// register code in its MB field. The interpretation layer emits it only with
// Confidence set to ConfidenceKnown, and Payload holds the decoded register
// content as a nested observation. A register recovered by heuristic inference
// is reported as a Candidate instead, never as an observation, so a consumer of
// inferred registers reads Interpretation.Candidates.
type CommBObservation struct {
	BDS        adsbtype.BDS
	Confidence Confidence
	Payload    Observation
}

// ObservationKind implements Observation.
func (CommBObservation) ObservationKind() ObservationKind { return KindCommB }

// DataLinkCapabilityObservation is a decoded BDS 1,0 data link capability
// report, carried as the payload of a self-identifying Comm-B reply.
type DataLinkCapabilityObservation struct {
	*DataLinkCapability
}

// ObservationKind implements Observation.
func (DataLinkCapabilityObservation) ObservationKind() ObservationKind {
	return KindDataLinkCapability
}

// ACASRAObservation is a decoded BDS 3,0 ACAS resolution advisory, carried as
// the payload of a self-identifying Comm-B reply.
type ACASRAObservation struct {
	*ACASRA
}

// ObservationKind implements Observation.
func (ACASRAObservation) ObservationKind() ObservationKind { return KindACASRA }

// TransponderStatusObservation is a decoded BDS E,7 transponder status report,
// carried as the payload of a self-identifying Comm-B reply.
type TransponderStatusObservation struct {
	*TransponderStatus
}

// ObservationKind implements Observation.
func (TransponderStatusObservation) ObservationKind() ObservationKind {
	return KindTransponderStatus
}

// SelectedVerticalIntentionObservation is a decoded BDS 4,0 selected vertical
// intention report, carried as the payload of an inferred Comm-B reply.
type SelectedVerticalIntentionObservation struct {
	*SelectedVerticalIntention
}

// ObservationKind implements Observation.
func (SelectedVerticalIntentionObservation) ObservationKind() ObservationKind {
	return KindSelectedVerticalIntention
}

// TrackAndTurnObservation is a decoded BDS 5,0 track and turn report, carried
// as the payload of an inferred Comm-B reply.
type TrackAndTurnObservation struct {
	*TrackAndTurn
}

// ObservationKind implements Observation.
func (TrackAndTurnObservation) ObservationKind() ObservationKind { return KindTrackAndTurn }

// HeadingAndSpeedObservation is a decoded BDS 6,0 heading and speed report,
// carried as the payload of an inferred Comm-B reply.
type HeadingAndSpeedObservation struct {
	*HeadingAndSpeed
}

// ObservationKind implements Observation.
func (HeadingAndSpeedObservation) ObservationKind() ObservationKind { return KindHeadingAndSpeed }

// CommonUsageGICBObservation is a decoded BDS 1,7 common usage GICB capability
// report, carried as the payload of an inferred Comm-B reply. Registers holds
// the GICB registers reported as available, in ascending register order; it is
// empty when the report lists none.
type CommonUsageGICBObservation struct {
	Registers []adsbtype.BDS
}

// ObservationKind implements Observation.
func (CommonUsageGICBObservation) ObservationKind() ObservationKind { return KindCommonUsageGICB }
