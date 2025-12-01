package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// unmarshalPartialDiscriminatedUnion handles partial JSON for discriminated unions.
func (v *Validator[T]) unmarshalPartialDiscriminatedUnion(data []byte, cfg *discriminatorConfig) (*T, *PartialState, ValidationErrors) {
	// Parse and repair the partial JSON first
	parseResult, parseErrs := parsePartialJSON(data)
	if parseErrs != nil {
		return nil, &PartialState{IsComplete: false}, parseErrs
	}

	// Try to determine the concrete type from discriminator
	instance, errs := newUnionFromJSONPartial[T](parseResult.Repaired, cfg, parseResult.Incomplete)
	if errs != nil {
		// If discriminator is incomplete or missing, we can't determine the type yet
		partialState := buildPartialStateFromPaths(parseResult.Incomplete, parseResult.TruncatedAt)

		// Add discriminator as incomplete field
		partialState.IncompleteFields = append([]IncompleteField{{
			Path:     []string{cfg.field},
			JSONPath: cfg.field,
			Reason:   "discriminator_incomplete",
		}}, partialState.IncompleteFields...)
		partialState.IsComplete = false

		return nil, partialState, errs
	}

	// Use common partial marshal flow
	result, state, errs := unmarshalPartialCommon[T](instance.ptr, parseResult)
	if result == nil {
		return nil, state, errs
	}

	// Convert to interface type T
	finalResult := instance.Result()
	return &finalResult, state, errs
}

// newUnionFromJSONPartial creates a union instance from potentially partial JSON.
// It handles the case where the discriminator field might be incomplete.
func newUnionFromJSONPartial[T any](repairedData []byte, cfg *discriminatorConfig, incompletePaths [][]string) (*unionInstance[T], ValidationErrors) {
	var peek map[string]any
	if err := json.Unmarshal(repairedData, &peek); err != nil {
		return nil, ValidationErrors{{Message: fmt.Sprintf("JSON unmarshal failed: %v", err), Type: ErrorTypeJSONDecode}}
	}

	// Check if discriminator field is incomplete
	discIncomplete := false
	for _, path := range incompletePaths {
		if len(path) == 1 && path[0] == cfg.field {
			discIncomplete = true
			break
		}
	}

	discValue, ok := peek[cfg.field]
	if !ok || discIncomplete {
		// Discriminator is missing or incomplete - can't determine type yet
		discValueStr := ""
		if ok {
			discValueStr = fmt.Sprintf("%v", discValue)
		}

		// Try to look up the type anyway (might be partial like "do" -> "dog")
		// But if it fails, that's expected
		concreteType, validationErr := cfg.lookupConcreteType(discValueStr)
		if validationErr != nil || concreteType == nil {
			return nil, ValidationErrors{{
				Loc:     []string{cfg.field},
				Message: fmt.Sprintf("discriminator field '%s' is incomplete or missing", cfg.field),
				Type:    ErrorTypeDiscriminatorMissing,
			}}
		}
		// If we found a match (partial discriminator), use it
		elemType := reflectutil.UnwrapPointer(concreteType)
		return &unionInstance[T]{ptr: reflect.New(elemType), concreteType: concreteType}, nil
	}

	// Discriminator is complete - proceed normally
	concreteType, validationErr := cfg.lookupConcreteType(fmt.Sprintf("%v", discValue))
	if validationErr != nil {
		return nil, ValidationErrors{*validationErr}
	}

	elemType := reflectutil.UnwrapPointer(concreteType)
	return &unionInstance[T]{ptr: reflect.New(elemType), concreteType: concreteType}, nil
}
