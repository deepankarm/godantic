package godanticlint

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the main analyzer that checks Field{X}() methods correspond to struct fields
var Analyzer = &analysis.Analyzer{
	Name:     "godanticlint",
	Doc:      "checks that Field{X}() methods correspond to actual struct fields",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Filter for function declarations
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)

		// Check for nolint directive
		if hasNoLint(fn) {
			return
		}

		// Skip if not a method (no receiver)
		if fn.Recv == nil || len(fn.Recv.List) == 0 {
			return
		}

		// Skip if method name doesn't start with "Field"
		if !strings.HasPrefix(fn.Name.Name, "Field") {
			return
		}

		// Extract field name by stripping "Field" prefix
		fieldName := fn.Name.Name[5:] // "Field" is 5 characters
		if fieldName == "" {
			return // Skip if method is just "Field" with no suffix
		}

		// Get receiver type
		recv := fn.Recv.List[0]
		recvType := pass.TypesInfo.TypeOf(recv.Type)
		if recvType == nil {
			return
		}

		// Handle pointer receivers
		if ptrType, ok := recvType.(*types.Pointer); ok {
			recvType = ptrType.Elem()
		}

		// Skip if receiver is not a struct type
		namedType, ok := recvType.(*types.Named)
		if !ok {
			return
		}

		structType, ok := namedType.Underlying().(*types.Struct)
		if !ok {
			return
		}

		// Check return type - must be FieldOptions[T]
		returnInfo, returnErr := checkReturnType(pass, fn)
		if returnErr != "" {
			pass.Reportf(fn.Name.Pos(), returnErr)
			return
		}

		// Check if struct has a field with the extracted name (including embedded fields recursively)
		actualField := findFieldInStruct(structType, fieldName)

		// Report error if field not found
		if actualField == nil {
			// Try to suggest similar field names
			suggestions := findSimilarFields(structType, fieldName)
			msg := fmt.Sprintf("method %s() does not correspond to any field on %s", fn.Name.Name, namedType.Obj().Name())
			if len(suggestions) > 0 {
				msg += fmt.Sprintf(" (did you mean %s?)", strings.Join(suggestions, ", "))
			}
			pass.Reportf(fn.Name.Pos(), msg)
			return
		}

		// Type checking: verify type parameter matches field type
		checkTypeMatch(pass, fn, actualField, returnInfo.typeArg)
	})

	return nil, nil
}

// fieldOptionsInfo holds info about a FieldOptions[T] return type
type fieldOptionsInfo struct {
	typeArg types.Type
}

// checkReturnType verifies that the method returns FieldOptions[T].
// Returns the type info and an error message (empty if valid).
func checkReturnType(pass *analysis.Pass, fn *ast.FuncDecl) (*fieldOptionsInfo, string) {
	fnObj := pass.TypesInfo.ObjectOf(fn.Name)
	if fnObj == nil {
		return nil, ""
	}

	sig, ok := fnObj.Type().(*types.Signature)
	if !ok {
		return nil, ""
	}

	results := sig.Results()
	if results.Len() != 1 {
		return nil, fmt.Sprintf("method %s() must return exactly one value of type FieldOptions[T]", fn.Name.Name)
	}

	returnType := results.At(0).Type()

	namedReturnType, ok := returnType.(*types.Named)
	if !ok {
		return nil, fmt.Sprintf("method %s() must return FieldOptions[T], got %s", fn.Name.Name, returnType)
	}

	// Must be named "FieldOptions"
	if namedReturnType.Obj().Name() != "FieldOptions" {
		return nil, fmt.Sprintf("method %s() must return FieldOptions[T], got %s", fn.Name.Name, returnType)
	}

	typeArgs := namedReturnType.TypeArgs()
	if typeArgs.Len() != 1 {
		return nil, fmt.Sprintf("method %s() must return FieldOptions[T] with one type parameter", fn.Name.Name)
	}

	return &fieldOptionsInfo{typeArg: typeArgs.At(0)}, ""
}

// checkTypeMatch verifies that the FieldOptions type parameter matches the field type
func checkTypeMatch(pass *analysis.Pass, fn *ast.FuncDecl, field *types.Var, typeArg types.Type) {
	fieldType := field.Type()

	if types.Identical(typeArg, fieldType) {
		return // Types match
	}

	// Check for pointer vs value type mismatch
	if ptrType, ok := fieldType.(*types.Pointer); ok {
		if types.Identical(typeArg, ptrType.Elem()) {
			pass.Reportf(fn.Name.Pos(), "method %s() returns FieldOptions[%s] but field %s has type %s (pointer mismatch)",
				fn.Name.Name, typeArg, field.Name(), fieldType)
			return
		}
	}
	if ptrType, ok := typeArg.(*types.Pointer); ok {
		if types.Identical(ptrType.Elem(), fieldType) {
			pass.Reportf(fn.Name.Pos(), "method %s() returns FieldOptions[%s] but field %s has type %s (pointer mismatch)",
				fn.Name.Name, typeArg, field.Name(), fieldType)
			return
		}
	}

	// Type mismatch
	pass.Reportf(fn.Name.Pos(), "method %s() returns FieldOptions[%s] but field %s has type %s",
		fn.Name.Name, typeArg, field.Name(), fieldType)
}

// findFieldInStruct recursively searches for a field by name, including embedded structs
func findFieldInStruct(structType *types.Struct, fieldName string) *types.Var {
	return findFieldInStructRecursive(structType, fieldName, make(map[*types.Struct]bool))
}

func findFieldInStructRecursive(structType *types.Struct, fieldName string, visited map[*types.Struct]bool) *types.Var {
	// Prevent infinite recursion with cyclic types
	if visited[structType] {
		return nil
	}
	visited[structType] = true

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)

		// Direct field match
		if field.Name() == fieldName {
			return field
		}

		// Check embedded fields (anonymous fields) recursively
		if field.Embedded() {
			embeddedType := field.Type()
			// Handle pointer to embedded type
			if ptrType, ok := embeddedType.(*types.Pointer); ok {
				embeddedType = ptrType.Elem()
			}
			// Check if embedded type is a named struct
			if embeddedNamed, ok := embeddedType.(*types.Named); ok {
				if embeddedStruct, ok := embeddedNamed.Underlying().(*types.Struct); ok {
					if found := findFieldInStructRecursive(embeddedStruct, fieldName, visited); found != nil {
						return found
					}
				}
			}
		}
	}
	return nil
}

// findSimilarFields finds field names similar to the given name (simple Levenshtein-like check)
func findSimilarFields(structType *types.Struct, targetName string) []string {
	var suggestions []string
	targetLower := strings.ToLower(targetName)

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		fieldName := field.Name()
		fieldLower := strings.ToLower(fieldName)

		// Simple similarity check: same length and similar characters
		if len(fieldLower) == len(targetLower) {
			diffs := 0
			for j := 0; j < len(fieldLower) && j < len(targetLower); j++ {
				if fieldLower[j] != targetLower[j] {
					diffs++
				}
			}
			// If only 1-2 character differences, suggest it
			if diffs > 0 && diffs <= 2 {
				suggestions = append(suggestions, fieldName)
			}
		}

		// Check if target is a substring of field name or vice versa
		if strings.Contains(fieldLower, targetLower) || strings.Contains(targetLower, fieldLower) {
			if fieldName != targetName {
				suggestions = append(suggestions, fieldName)
			}
		}
	}

	// Limit to 3 suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// hasNoLint checks if the function has a nolint:godanticlint directive
func hasNoLint(fn *ast.FuncDecl) bool {
	if fn.Doc == nil {
		return false
	}
	text := fn.Doc.Text()
	return strings.Contains(text, "nolint:godanticlint") || strings.Contains(text, "nolint:all")
}
