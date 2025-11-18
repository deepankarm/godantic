package schema_test

import (
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Test enum constraints from type-level validation
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderConfirmed OrderStatus = "confirmed"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
)

// Type-level validation for OrderStatus
func (OrderStatus) FieldOrderStatus() godantic.FieldOptions[OrderStatus] {
	return godantic.Field(
		godantic.Required[OrderStatus](),
		godantic.OneOf(OrderPending, OrderConfirmed, OrderShipped, OrderDelivered),
		godantic.Description[OrderStatus]("Current status of the order"),
	)
}

type Order struct {
	ID     string
	Status OrderStatus // Uses type-level validation
}

func (o *Order) FieldID() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func TestEnumConstraintsInSchema(t *testing.T) {
	t.Run("enum from type-level validation should be in schema", func(t *testing.T) {
		sg := schema.NewGenerator[Order]()
		schemaJSON, err := sg.GenerateJSON()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Check that enum constraint is present
		if !strings.Contains(schemaJSON, `"enum"`) {
			t.Error("schema should contain enum constraint from OrderStatus type")
		}

		// Check that all enum values are present
		expectedValues := []string{
			`"pending"`,
			`"confirmed"`,
			`"shipped"`,
			`"delivered"`,
		}
		for _, val := range expectedValues {
			if !strings.Contains(schemaJSON, val) {
				t.Errorf("schema should contain enum value %s", val)
			}
		}

		// Check that description is present
		if !strings.Contains(schemaJSON, "Current status of the order") {
			t.Error("schema should contain description from type-level validation")
		}

		// Verify the Status field is marked as required
		if !strings.Contains(schemaJSON, `"required"`) {
			t.Error("schema should mark Status as required")
		}
	})
}
