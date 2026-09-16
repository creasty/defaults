package defaults

import (
	"encoding"
	"encoding/json"
)

// unmarshalTag offers value to target's own unmarshalers, UnmarshalText first and UnmarshalJSON
// second, and reports whether one took it.
//
// An empty value is offered to neither: it asks for the target's zero value rather than for anything
// to be parsed. "{}" and "[]" are kept from UnmarshalJSON, which would read them as an empty object
// or array, while UnmarshalText, which has no such reading of them, is offered them like any other.
//
// When neither took it, the error is the rejection to report: UnmarshalText's, since it was asked
// first, or UnmarshalJSON's when UnmarshalText was not offered the value or does not exist, and nil
// when neither was offered anything.
//
// target is what an unmarshaler is implemented on, so the caller passes a pointer to the value rather
// than the value. Taking that rather than a reflect.Value keeps reflect out of the contract, and
// leaves the caller making field.Addr().Interface() once for both type assertions rather than twice.
func unmarshalTag(target any, value string) (bool, error) {
	var textErr, jsonErr error

	asText, ok := target.(encoding.TextUnmarshaler)
	if ok && value != "" {
		if textErr = asText.UnmarshalText([]byte(value)); textErr == nil {
			return true, nil
		}
	}

	asJSON, ok := target.(json.Unmarshaler)
	if ok && value != "" && value != "{}" && value != "[]" {
		if jsonErr = asJSON.UnmarshalJSON([]byte(value)); jsonErr == nil {
			return true, nil
		}
	}

	// UnmarshalText is offered the value first, so if both rejected it, its rejection is reported.
	if textErr != nil {
		return false, textErr
	}
	return false, jsonErr
}
