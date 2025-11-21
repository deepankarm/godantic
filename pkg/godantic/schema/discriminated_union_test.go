package schema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Test 1: Simple interface discriminated union

type Animal interface {
	IsAnimal()
}

type Cat struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (Cat) IsAnimal() {}

type Dog struct {
	Type  string `json:"type"`
	Breed string `json:"breed"`
}

func (Dog) IsAnimal() {}

type PetOwner struct {
	Name string `json:"name"`
	Pet  Animal `json:"pet"`
}

func (p *PetOwner) FieldPet() godantic.FieldOptions[Animal] {
	return godantic.Field(
		godantic.Required[Animal](),
		godantic.DiscriminatedUnion[Animal]("type", map[string]any{
			"cat": Cat{},
			"dog": Dog{},
		}),
	)
}

// TestInterfaceDiscriminatedUnion tests that interface fields with discriminated unions
// generate proper oneOf schemas (not just `true`)
func TestInterfaceDiscriminatedUnion(t *testing.T) {

	sg := schema.NewGenerator[PetOwner]()
	generatedSchema, err := sg.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]any
	schemaJSON, _ := json.Marshal(generatedSchema)
	json.Unmarshal(schemaJSON, &schemaMap)

	defs := schemaMap["$defs"].(map[string]any)
	petOwner := defs["PetOwner"].(map[string]any)
	props := petOwner["properties"].(map[string]any)
	petProp := props["pet"].(map[string]any)

	// Verify oneOf exists
	oneOf, ok := petProp["oneOf"].([]any)
	if !ok || len(oneOf) != 2 {
		t.Fatalf("Expected pet to have oneOf with 2 variants, got: %v", petProp)
	}

	// Verify discriminator
	discriminator := petProp["discriminator"].(map[string]any)
	if discriminator["propertyName"] != "type" {
		t.Errorf("Expected discriminator propertyName 'type', got: %v", discriminator["propertyName"])
	}

	// Verify variant schemas exist
	if _, exists := defs["Cat"]; !exists {
		t.Error("Expected Cat schema in $defs")
	}
	if _, exists := defs["Dog"]; !exists {
		t.Error("Expected Dog schema in $defs")
	}
}

// Test 2: Nested interface discriminated union (the bug we fixed)

type NestedPayload interface {
	IsNestedPayload()
}

type NestedTextPayload struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (NestedTextPayload) IsNestedPayload() {}

type NestedImagePayload struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (NestedImagePayload) IsNestedPayload() {}

type APIResponse interface {
	IsAPIResponse()
}

type NestedSuccessResponse struct {
	Status  string        `json:"status"`
	Payload NestedPayload `json:"payload"` // Nested discriminated union
}

func (NestedSuccessResponse) IsAPIResponse() {}

func (s *NestedSuccessResponse) FieldPayload() godantic.FieldOptions[NestedPayload] {
	return godantic.Field(
		godantic.Required[NestedPayload](),
		godantic.DiscriminatedUnion[NestedPayload]("type", map[string]any{
			"text":  NestedTextPayload{},
			"image": NestedImagePayload{},
		}),
	)
}

type NestedErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (NestedErrorResponse) IsAPIResponse() {}

type NestedContainer struct {
	Response APIResponse `json:"response"`
}

func (c *NestedContainer) FieldResponse() godantic.FieldOptions[APIResponse] {
	return godantic.Field(
		godantic.Required[APIResponse](),
		godantic.DiscriminatedUnion[APIResponse]("status", map[string]any{
			"success": NestedSuccessResponse{},
			"error":   NestedErrorResponse{},
		}),
	)
}

// TestNestedInterfaceDiscriminatedUnion tests the bug fix: nested discriminated unions
// where a variant struct itself contains an interface field with a discriminated union
func TestNestedInterfaceDiscriminatedUnion(t *testing.T) {

	// Test with Generator[T]
	sg := schema.NewGenerator[NestedContainer]()
	generatedSchema, err := sg.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]any
	schemaJSON, _ := json.Marshal(generatedSchema)
	json.Unmarshal(schemaJSON, &schemaMap)

	defs := schemaMap["$defs"].(map[string]any)

	// Critical check: NestedSuccessResponse.payload should have oneOf, not be `true`
	successResp := defs["NestedSuccessResponse"].(map[string]any)
	props := successResp["properties"].(map[string]any)
	payloadProp, ok := props["payload"].(map[string]any)
	if !ok {
		t.Fatalf("Expected payload to be an object schema, got: %T = %v", props["payload"], props["payload"])
	}

	oneOf, ok := payloadProp["oneOf"].([]any)
	if !ok || len(oneOf) != 2 {
		t.Fatalf("Expected payload to have oneOf with 2 variants, got: %v", payloadProp)
	}

	// Verify nested variant schemas exist
	if _, exists := defs["NestedTextPayload"]; !exists {
		t.Error("Expected NestedTextPayload schema in $defs")
	}
	if _, exists := defs["NestedImagePayload"]; !exists {
		t.Error("Expected NestedImagePayload schema in $defs")
	}
}

// Test 3: GenerateForType with nested discriminated union

type Content interface {
	IsContent()
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (TextContent) IsContent() {}

type Notification interface {
	IsNotification()
}

type EmailNotification struct {
	Channel string  `json:"channel"`
	Content Content `json:"content"`
}

func (EmailNotification) IsNotification() {}

func (e *EmailNotification) FieldContent() godantic.FieldOptions[Content] {
	return godantic.Field(
		godantic.Required[Content](),
		godantic.DiscriminatedUnion[Content]("type", map[string]any{
			"text": TextContent{},
		}),
	)
}

type NotificationConfig struct {
	Notification Notification `json:"notification"`
}

func (n *NotificationConfig) FieldNotification() godantic.FieldOptions[Notification] {
	return godantic.Field(
		godantic.DiscriminatedUnion[Notification]("channel", map[string]any{
			"email": EmailNotification{},
		}),
	)
}

// TestInterfaceFieldsWithGenerateForType tests that the GenerateForType path
// (used by gingodantic) also correctly handles nested discriminated unions
func TestInterfaceFieldsWithGenerateForType(t *testing.T) {

	// Use GenerateForType (not Generator[T])
	schemaMap, err := schema.GenerateForType(reflect.TypeOf(NotificationConfig{}))
	if err != nil {
		t.Fatalf("GenerateForType failed: %v", err)
	}

	defs := schemaMap["$defs"].(map[string]any)

	// Check EmailNotification.content has oneOf
	emailNotif := defs["EmailNotification"].(map[string]any)
	props := emailNotif["properties"].(map[string]any)
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Fatalf("Expected content to be an object schema, got: %T", props["content"])
	}

	oneOf, ok := contentProp["oneOf"].([]any)
	if !ok || len(oneOf) == 0 {
		t.Fatalf("Expected content to have oneOf, got: %v", contentProp)
	}

	// Verify variant schema exists
	if _, exists := defs["TextContent"]; !exists {
		t.Error("Expected TextContent schema in $defs")
	}
}
