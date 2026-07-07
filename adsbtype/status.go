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

package adsbtype

import "fmt"

// notUsed is shared by the reserved/unused values of the transponder status
// subfields (BDS E,7).
const notUsed = "Not used"

// noData is shared by the "no data or not used" values of the transponder
// status source and bus subfields (BDS E,7).
const noData = "No data or not used"

// SDI is the source/side data identifier reported in the transponder status
// and diagnostics register (BDS E,7).
type SDI uint64

// Source/side data identifier values.
const (
	SDI0 SDI = 0 // Not used
	SDI1 SDI = 1 // Side 1
	SDI2 SDI = 2 // Side 2
	SDI3 SDI = 3 // Not used
)

var mSDI = map[SDI]string{
	SDI0: notUsed,
	SDI1: "Side 1",
	SDI2: "Side 2",
	SDI3: notUsed,
}

// String representation of SDI.
func (c SDI) String() string {
	if str, ok := mSDI[c]; ok {
		return str
	}

	return fmt.Sprintf("Unknown value %d", c)
}

// CIS is the control input selection reported in the transponder status and
// diagnostics register (BDS E,7).
type CIS uint64

// Control input selection values.
const (
	CIS0 CIS = 0 // Burst time
	CIS1 CIS = 1 // Port A or 1
	CIS2 CIS = 2 // Port B or 2
	CIS3 CIS = 3 // Port C or 3
)

var mCIS = map[CIS]string{
	CIS0: "Burst time",
	CIS1: "Port A or 1",
	CIS2: "Port B or 2",
	CIS3: "Port C or 3",
}

// String representation of CIS.
func (c CIS) String() string {
	if str, ok := mCIS[c]; ok {
		return str
	}

	return fmt.Sprintf("Unknown value %d", c)
}

// DSS is a multiple data source reporting selection (which source is in use)
// in the transponder status and diagnostics register (BDS E,7).
type DSS uint64

// Data source reporting selection values.
const (
	DSS0 DSS = 0 // No data or not used
	DSS1 DSS = 1 // Source 1 in use
	DSS2 DSS = 2 // Source 2 in use
	DSS3 DSS = 3 // Source 3 in use
)

var mDSS = map[DSS]string{
	DSS0: noData,
	DSS1: "Source 1 in use",
	DSS2: "Source 2 in use",
	DSS3: "Source 3 in use",
}

// String representation of DSS.
func (c DSS) String() string {
	if str, ok := mDSS[c]; ok {
		return str
	}

	return fmt.Sprintf("Unknown value %d", c)
}

// BST is a bus or port status in the transponder status and diagnostics
// register (BDS E,7).
type BST uint64

// Bus/port status values.
const (
	BST0 BST = 0 // No data or not used
	BST1 BST = 1 // Active
	BST2 BST = 2 // Inactive
	BST3 BST = 3 // Fail
)

var mBST = map[BST]string{
	BST0: noData,
	BST1: "Active",
	BST2: "Inactive",
	BST3: "Fail",
}

// String representation of BST.
func (c BST) String() string {
	if str, ok := mBST[c]; ok {
		return str
	}

	return fmt.Sprintf("Unknown value %d", c)
}
