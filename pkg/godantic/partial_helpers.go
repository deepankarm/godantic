package godantic

import (
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/partialjson"
)

// parsePartialJSON repairs and parses incomplete JSON.
func parsePartialJSON(data []byte) (*partialjson.ParseResult, ValidationErrors) {
	parser := partialjson.NewParser(false) // non-strict for LLM output
	parseResult, err := parser.Parse(data)
	if err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("partial JSON parse failed: %v", err),
			Type:    ErrorTypeJSONDecode,
		}}
	}
	return parseResult, nil
}

// applyAfterValidateIfComplete calls AfterValidate hook only if the state is complete.
func applyAfterValidateIfComplete[T any](obj *T, state *PartialState) ValidationErrors {
	if state.IsComplete {
		if err := callAfterValidateHook(obj); err != nil {
			return ValidationErrors{{
				Loc:     []string{},
				Message: fmt.Sprintf("AfterValidate hook failed: %v", err),
				Type:    ErrorTypeHookError,
			}}
		}
	}
	return nil
}

// unmarshalPartialCommon handles the common flow for partial JSON unmarshaling.
// This is used by both regular structs and discriminated unions.
func unmarshalPartialCommon[T any](objPtr reflect.Value, parseResult *partialjson.ParseResult) (*T, *PartialState, ValidationErrors) {
	// Apply BeforeValidate hook
	repairedData, hookErrs := applyBeforeValidateHook[[]byte](objPtr, parseResult.Repaired)
	if hookErrs != nil {
		partialState := buildPartialStateFromPaths(parseResult.Incomplete, parseResult.TruncatedAt)
		return nil, partialState, hookErrs
	}

	// Use walkParsePartial for partial JSON support
	partialResult, errs := walkParsePartial(objPtr, repairedData)

	// Build partial state from parser results
	partialState := buildPartialStateFromPaths(parseResult.Incomplete, parseResult.TruncatedAt)

	// Merge any additional incomplete paths from walker
	partialState.MergeIncompleteFields(partialResult.IncompletePaths, parseResult.TruncatedAt)

	// Return nil on JSON decode errors
	if errs.HasJSONDecodeError() {
		return nil, partialState, errs
	}

	// Get the result
	obj := objPtr.Elem().Interface().(T)

	// Apply AfterValidate hook if complete
	if hookErrs := applyAfterValidateIfComplete(&obj, partialState); hookErrs != nil {
		return &obj, partialState, hookErrs
	}

	return &obj, partialState, errs
}
