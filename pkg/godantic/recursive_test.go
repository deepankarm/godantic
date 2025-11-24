package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// TreeNodeRecursive for testing recursive validation
type TreeNodeRecursive struct {
	Name     string               `json:"name"`
	Value    int                  `json:"value"`
	Children []*TreeNodeRecursive `json:"children,omitempty"`
}

func (n *TreeNodeRecursive) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
	)
}

func (n *TreeNodeRecursive) FieldValue() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(0),
	)
}

// TestRecursiveValidation tests that recursive structs can be validated
func TestRecursiveValidation(t *testing.T) {
	validator := godantic.NewValidator[TreeNodeRecursive]()

	// Valid tree
	validTree := TreeNodeRecursive{
		Name:  "root",
		Value: 10,
		Children: []*TreeNodeRecursive{
			{Name: "child1", Value: 5},
			{Name: "child2", Value: 3, Children: []*TreeNodeRecursive{
				{Name: "grandchild", Value: 1},
			}},
		},
	}

	errs := validator.Validate(&validTree)
	if len(errs) > 0 {
		t.Errorf("Expected no validation errors, got: %v", errs)
	}

	// Invalid tree - root has invalid value
	invalidTree := TreeNodeRecursive{
		Name:  "", // Invalid: empty name (required, minLen 1)
		Value: 10,
	}

	errs = validator.Validate(&invalidTree)
	if len(errs) == 0 {
		t.Error("Expected validation errors for empty name")
	}

	// Another invalid case: negative value
	invalidTree2 := TreeNodeRecursive{
		Name:  "root",
		Value: -1, // Invalid: negative value
	}

	errs = validator.Validate(&invalidTree2)
	if len(errs) == 0 {
		t.Error("Expected validation errors for negative value")
	}
}

// LinkedListNode for testing deeply nested recursive validation
type LinkedListNode struct {
	ID   int             `json:"id"`
	Next *LinkedListNode `json:"next,omitempty"`
}

func (n *LinkedListNode) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
	)
}

// TestRecursiveValidationDeep tests deeply nested recursive validation
func TestRecursiveValidationDeep(t *testing.T) {
	validator := godantic.NewValidator[LinkedListNode]()

	// Create a linked list: 1 -> 2 -> 3 -> 4 -> 5
	node5 := &LinkedListNode{ID: 5}
	node4 := &LinkedListNode{ID: 4, Next: node5}
	node3 := &LinkedListNode{ID: 3, Next: node4}
	node2 := &LinkedListNode{ID: 2, Next: node3}
	node1 := &LinkedListNode{ID: 1, Next: node2}

	errs := validator.Validate(node1)
	if len(errs) > 0 {
		t.Errorf("Expected no validation errors for valid linked list, got: %v", errs)
	}

	// Invalid list with node having ID = 0
	invalidNode := &LinkedListNode{ID: 0}
	errs = validator.Validate(invalidNode)
	if len(errs) == 0 {
		t.Error("Expected validation error for ID = 0")
	}
}

// PersonRecursive for testing mutual reference validation
type PersonRecursive struct {
	Name       string             `json:"name"`
	BestFriend *PersonRecursive   `json:"best_friend,omitempty"`
	Friends    []*PersonRecursive `json:"friends,omitempty"`
}

func (p *PersonRecursive) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(2),
	)
}

// TestRecursiveMutualReference tests validation with mutual references
func TestRecursiveMutualReference(t *testing.T) {
	validator := godantic.NewValidator[PersonRecursive]()

	// Valid mutual friendship
	alice := &PersonRecursive{Name: "Alice"}
	bob := &PersonRecursive{Name: "Bob", BestFriend: alice}
	alice.BestFriend = bob

	errs := validator.Validate(alice)
	if len(errs) > 0 {
		t.Errorf("Expected no validation errors for mutual friends, got: %v", errs)
	}

	// Invalid - name too short
	charlie := &PersonRecursive{Name: "C"} // Too short
	errs = validator.Validate(charlie)
	if len(errs) == 0 {
		t.Error("Expected validation error for short name")
	}
}
