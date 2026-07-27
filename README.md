# Overview
[![PkgGoDev](https://pkg.go.dev/badge/kreklow.us/go/go-adsb)](https://pkg.go.dev/kreklow.us/go/go-adsb)
![License](https://img.shields.io/github/license/cjkreklow/go-adsb)
![Version](https://img.shields.io/github/v/tag/cjkreklow/go-adsb)
![Status](https://github.com/cjkreklow/go-adsb/actions/workflows/push.yml/badge.svg?branch=main)
[![codecov](https://codecov.io/gh/cjkreklow/go-adsb/branch/main/graph/badge.svg)](https://codecov.io/gh/cjkreklow/go-adsb)

`go-adsb` is a Go module that includes packages for working with ADS-B and
Mode S aircraft transponder data.

## beast
The `beast` package is a low-level library for handling data in [Mode S
Beast format](https://wiki.jetvision.de/wiki/Mode-S_Beast:Data_Output_Formats),
as provided by common software such as
[dump1090](https://github.com/flightaware/dump1090).
`Decoder` provides a consumer for an `io.Reader` such as
[net.Conn](https://golang.org/pkg/net/#Conn), which will then parse a Beast
stream into individual frames. These frames are passed to a
[BinaryUnmarshaler](https://golang.org/pkg/encoding/#BinaryUnmarshaler) via
`Decode`. The provided `Frame` is a BinaryUnmarshaler that provides methods
to extract the Beast data such as timestamp and signal level, as well as the
enclosed Mode S or ADS-B data.

## adsb
The `adsb` package is a library for decoding Mode S and ADS-B transponder
messages. `InterpretModeS`, `InterpretMessage` and `InterpretBeastFrame` are
the entry points for decoding a frame. They return an `*Interpretation`
holding the message family, metadata, typed `Observation` values, Comm-B
`Candidate`s and `Warning`s, so a caller does not dispatch on every downlink
format, type code, control field and BDS register method itself. The layer is
stateless: it reuses the existing decoders, probes only what a message can
carry, and does not track aircraft state.

Underneath it, `Message` and `RawMessage` are the lower-level API. `Message`
provides functions to retrieve decoded values such as altitude and callsign
from the encoded data, and `RawMessage` provides access to arbitrary bit
sequences and named message fields.

Both `Message` and `RawMessage` designed to accept a `beast.Frame` to
provide a complete solution for decoding usable values from an incoming data
stream.

## adsbtype
The `adsbtype` package provides constants for Mode S and ADS-B data fields
that have fixed values. Converting the value to a provided data type allows
the text description of the value to be returned via the `%s` operator in
Printf-style operations.

# Usage
See the documentation on [pkg.go.dev](https://pkg.go.dev/kreklow.us/go/go-adsb)
for import paths and usage information, including runnable examples for the
interpretation layer.

## Install
```
go get kreklow.us/go/go-adsb
```

## Interpreting a hex Mode S payload
`InterpretModeS` classifies a single raw payload. Observations are typed;
switch over `ObservationKind` (or the concrete type) to consume them.

```go
in, err := adsb.InterpretModeS(payload, adsb.InterpretOptions{})
if err != nil {
	// malformed payload
}
for _, o := range in.Observations {
	switch obs := o.(type) {
	case adsb.CallsignObservation:
		fmt.Println("callsign:", obs.Callsign)
	case adsb.AltitudeObservation:
		fmt.Println("altitude:", obs.Altitude)
	}
}
```

## Interpreting a Beast stream
`InterpretBeastFrame` preserves the Beast metadata (type, timestamp and signal)
and interprets the enclosed Mode S or Mode A/C payload.

```go
dec := beast.NewDecoder(conn)
for {
	var f beast.Frame
	if err := dec.Decode(&f); err != nil {
		break
	}
	in, err := adsb.InterpretBeastFrame(&f, adsb.InterpretOptions{})
	if err != nil {
		continue
	}
	// use in.Family, in.Meta, in.Observations
}
```

## Retrieving a single field
For a caller that wants one named field rather than a classified result, the
`Message` methods decode it directly. A field that the received message format
does not carry returns an error wrapping `ErrNotAvailable`, which is normal
control flow rather than a failure.

```go
msg := new(adsb.Message)
if err := msg.UnmarshalBinary(payload); err != nil {
	// malformed payload
}
alt, err := msg.Alt()
switch {
case errors.Is(err, adsb.ErrNotAvailable):
	// this message format carries no altitude
case err != nil:
	// decoding failed
default:
	fmt.Println("altitude:", alt)
}
```

## Comm-B ambiguity
DF20/21 replies carry no register identifier. Registers that self-identify
become `Observations` with `ConfidenceKnown`. When `InterpretOptions.InferCommB`
is set, the remaining registers are inferred and reported as `Candidates`, never
as facts. A sole matching register is reported with `ConfidenceInferred` and its
decoded value in `Candidate.Payload`; several matching registers are reported
with `ConfidenceCandidate`, a nil `Payload` and a `WarningAmbiguous`, because an
ambiguous register is never decoded. An inferred register is not guaranteed to
be correct.

## Addresses
For DF18 the 24-bit address field is not always an ICAO aircraft address:
anonymous, non-ICAO and IMF=1 TIS-B/ADS-R messages carry a Mode A code with a
track file number instead. `Meta.AddressKind` reports the interpretation and
`Meta.ICAO` is populated only for a genuine ICAO address, so a non-nil `ICAO`
can be relied upon. `Meta.Address` holds the 24-bit aircraft address and is nil
when the message carries none (a DF18 management or reserved control field).

## Position decoding
The interpretation layer is stateless. A `CPRObservation` always exposes the
encoded position. Local decoding requires a receiver/reference position via
`InterpretOptions.Reference`; global even/odd pair decoding is available through
the lower-level `DecodeGlobalPosition` API, and stateful per-aircraft tracking
is out of scope for this layer.

## Scope
The interpretation layer intentionally excludes:

- DF24 Comm-D / ELM high-level assembly
- DF19 / DF22 military formats
- vendor and private formats
- UAT (978 MHz)

# About
`go-adsb` is maintained by Collin Kreklow. The source code is licensed under
the terms of the MIT license, see `LICENSE.txt` for further information.
