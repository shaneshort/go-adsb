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

import "kreklow.us/go/go-adsb/adsbtype"

// TransponderStatus is a decoded Comm-B transponder status and diagnostics
// report (BDS E,7), following the MB bit assignments of ICAO Doc 9871 Table
// A-2-231. The register reports the configuration and status of the
// transponder installation.
//
// The failure flags (DiversityFailure through LowerSquitterFailure) are true
// when a failure is reported. The input-status flags (AirGround1Active through
// ExtendedSquitterDisableActive) are true when the input is active. The
// inactive/failed flags (ACASInputInactive through SelectedControlInactive)
// are true when the function is inactive or failed. The port-select flags
// (FMCGNSSSourcePort2, IRSFMSSourcePort2, FMCSelectPort2) are true when the
// alternate port (port 2) is selected.
type TransponderStatus struct {
	SDICode adsbtype.SDI // source/side data identifier (MB bits 9-10)

	NonDiversity bool // MB bit 11: true = non-diversity transponder

	DiversityFailure     bool // MB bit 12
	UpperReceiverFailure bool // MB bit 13
	LowerReceiverFailure bool // MB bit 14
	UpperSquitterFailure bool // MB bit 15
	LowerSquitterFailure bool // MB bit 16

	AirGround1Active              bool // MB bit 17
	AirGround2Active              bool // MB bit 18
	GPSTimeMark1Active            bool // MB bit 19
	GPSTimeMark2Active            bool // MB bit 20
	ExtendedSquitterDisableActive bool // MB bit 23

	ModeSLimitingPowerOn bool // MB bit 21: true = limiting during power-on cycle
	ModeSLimiting        bool // MB bit 22: true = in limiting

	ACASInputInactive       bool // MB bit 24
	ADSBOutInactive         bool // MB bit 25
	SelectedControlInactive bool // MB bit 26

	ControlInputSelection adsbtype.CIS // MB bits 27-28

	AirDataSource         adsbtype.DSS // MB bits 29-30 (source in use)
	AltitudeAlternatePort bool         // MB bit 31: true = alternate (port B) selected
	AltitudePortAStatus   adsbtype.BST // MB bits 32-33
	AltitudePortBStatus   adsbtype.BST // MB bits 34-35

	FMCGNSSSourcePort2 bool         // MB bit 36
	FMCGNSS1BusStatus  adsbtype.BST // MB bits 37-38
	FMCGNSS2BusStatus  adsbtype.BST // MB bits 39-40

	IRSAHRSSource                 adsbtype.DSS // MB bits 41-42 (source in use)
	IRSFMSSourcePort2             bool         // MB bit 43
	IRSFMSDataConcentrator1Status adsbtype.BST // MB bits 44-45
	IRSFMSDataConcentrator2Status adsbtype.BST // MB bits 46-47

	FMCSelectPort2 bool         // MB bit 48
	FMC1BusStatus  adsbtype.BST // MB bits 49-50
	FMC2BusStatus  adsbtype.BST // MB bits 51-52

	MSPATSUCMU1Status adsbtype.BST // MB bits 53-54
	MSPATSUCMU2Status adsbtype.BST // MB bits 55-56
}

// TransponderStatus decodes the MB field as a BDS E,7 transponder status and
// diagnostics report (ICAO Doc 9871 Table A-2-231). It returns an error
// wrapping ErrNotAvailable unless the message is a Comm-B reply (DF 20 or 21).
// The register identity is not verified; use InferBDS to check that the MB
// field self-identifies as BDS E,7 (its first eight bits equal 0xE7).
func (m *Message) TransponderStatus() (*TransponderStatus, error) {
	r, err := m.commBRaw()
	if err != nil {
		return nil, err
	}

	return &TransponderStatus{
		SDICode: adsbtype.SDI(r.mbbits(9, 10)),

		NonDiversity: r.mbbits(11, 11) == 1,

		DiversityFailure:     r.mbbits(12, 12) == 1,
		UpperReceiverFailure: r.mbbits(13, 13) == 1,
		LowerReceiverFailure: r.mbbits(14, 14) == 1,
		UpperSquitterFailure: r.mbbits(15, 15) == 1,
		LowerSquitterFailure: r.mbbits(16, 16) == 1,

		AirGround1Active:              r.mbbits(17, 17) == 1,
		AirGround2Active:              r.mbbits(18, 18) == 1,
		GPSTimeMark1Active:            r.mbbits(19, 19) == 1,
		GPSTimeMark2Active:            r.mbbits(20, 20) == 1,
		ExtendedSquitterDisableActive: r.mbbits(23, 23) == 1,

		ModeSLimitingPowerOn: r.mbbits(21, 21) == 1,
		ModeSLimiting:        r.mbbits(22, 22) == 1,

		ACASInputInactive:       r.mbbits(24, 24) == 1,
		ADSBOutInactive:         r.mbbits(25, 25) == 1,
		SelectedControlInactive: r.mbbits(26, 26) == 1,

		ControlInputSelection: adsbtype.CIS(r.mbbits(27, 28)),

		AirDataSource:         adsbtype.DSS(r.mbbits(29, 30)),
		AltitudeAlternatePort: r.mbbits(31, 31) == 1,
		AltitudePortAStatus:   adsbtype.BST(r.mbbits(32, 33)),
		AltitudePortBStatus:   adsbtype.BST(r.mbbits(34, 35)),

		FMCGNSSSourcePort2: r.mbbits(36, 36) == 1,
		FMCGNSS1BusStatus:  adsbtype.BST(r.mbbits(37, 38)),
		FMCGNSS2BusStatus:  adsbtype.BST(r.mbbits(39, 40)),

		IRSAHRSSource:                 adsbtype.DSS(r.mbbits(41, 42)),
		IRSFMSSourcePort2:             r.mbbits(43, 43) == 1,
		IRSFMSDataConcentrator1Status: adsbtype.BST(r.mbbits(44, 45)),
		IRSFMSDataConcentrator2Status: adsbtype.BST(r.mbbits(46, 47)),

		FMCSelectPort2: r.mbbits(48, 48) == 1,
		FMC1BusStatus:  adsbtype.BST(r.mbbits(49, 50)),
		FMC2BusStatus:  adsbtype.BST(r.mbbits(51, 52)),

		MSPATSUCMU1Status: adsbtype.BST(r.mbbits(53, 54)),
		MSPATSUCMU2Status: adsbtype.BST(r.mbbits(55, 56)),
	}, nil
}
