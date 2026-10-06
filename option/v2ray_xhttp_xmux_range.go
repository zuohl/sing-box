package option

import (
	"strconv"
	"strings"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
)

// V2RayXHTTPRange is a range option used by XHTTP fields (x_padding_bytes,
// sc_max_each_post_bytes, xmux limits, etc.). In sing-box options these ranges
// are canonically expressed as "min-max" strings (e.g. "100-1000"), but external
// configs and subscription providers also author them as bare numbers (100) or
// two-element arrays ([100, 1000]).
// Accepting all three spellings keeps those configs loadable without changing
// what they mean:
//
//	"600-900"   our canonical string form
//	"900"       a single integer, equivalent to "900-900"
//	600         a bare JSON number, likewise
//	[600, 900]  the Xray / sing-box-extended array form
//
// The zero value marshals back as the empty string, so an unset field stays
// absent from a re-serialized config.
type V2RayXHTTPRange string

// XmuxRange is kept as an alias for backwards compatibility with existing code.
type XmuxRange = V2RayXHTTPRange

// MarshalJSON emits the canonical string form.
func (r V2RayXHTTPRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// UnmarshalJSON accepts the string, bare-number and [min,max] array spellings,
// normalizing all of them to the "min-max" string form.
func (r *V2RayXHTTPRange) UnmarshalJSON(content []byte) error {
	var stringValue string
	if err := json.Unmarshal(content, &stringValue); err == nil {
		*r = V2RayXHTTPRange(strings.TrimSpace(stringValue))
		return nil
	}
	var numberValue int
	if err := json.Unmarshal(content, &numberValue); err == nil {
		*r = V2RayXHTTPRange(strconv.Itoa(numberValue))
		return nil
	}
	var arrayValue []int
	if err := json.Unmarshal(content, &arrayValue); err != nil {
		return E.New("invalid xhttp range: expected \"min-max\", a number, or [min,max]")
	}
	switch len(arrayValue) {
	case 1:
		*r = V2RayXHTTPRange(strconv.Itoa(arrayValue[0]))
	case 2:
		*r = V2RayXHTTPRange(strconv.Itoa(arrayValue[0]) + "-" + strconv.Itoa(arrayValue[1]))
	default:
		return E.New("invalid xhttp range: array form takes 1 or 2 elements, got ", len(arrayValue))
	}
	return nil
}
