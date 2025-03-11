package hash

import (
	"fmt"
	"hash"

	"github.com/davecgh/go-spew/spew"
)

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	fmt.Fprintf(hasher, "%v", ForHash(objectToWrite))
}

var prettyPrintConfig = &spew.ConfigState{
	Indent:                  "  ",
	DisableMethods:          true,
	DisablePointerAddresses: true,
	DisableCapacities:       true,
}

// The config MUST NOT be changed because that could change the result of a hash operation
var prettyPrintConfigForHash = &spew.ConfigState{
	Indent:                  " ",
	SortKeys:                true,
	DisableMethods:          true,
	SpewKeys:                true,
	DisablePointerAddresses: true,
	DisableCapacities:       true,
}

// Pretty wrap the spew.Sdump with Indent, and disabled methods like error() and String()
// The output may change over time, so for guaranteed output please take more direct control
func Pretty(a interface{}) string {
	return prettyPrintConfig.Sdump(a)
}

// ForHash keeps the original Spew.Sprintf format to ensure the same checksum
func ForHash(a interface{}) string {
	return prettyPrintConfigForHash.Sprintf("%#v", a)
}

// OneLine outputs the object in one line
func OneLine(a interface{}) string {
	return prettyPrintConfig.Sprintf("%#v", a)
}
