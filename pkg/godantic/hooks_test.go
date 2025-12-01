package godantic_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

type MessageWithHooks struct {
	Role    string   `json:"role"`
	Content []string `json:"content"`
}

func (m *MessageWithHooks) FieldRole() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (m *MessageWithHooks) FieldContent() godantic.FieldOptions[[]string] {
	return godantic.Field(godantic.Required[[]string]())
}

// BeforeValidate normalizes string content to array (like Python compatibility)
func (m *MessageWithHooks) BeforeValidate(raw map[string]any) error {
	if contentStr, ok := raw["content"].(string); ok {
		raw["content"] = []string{contentStr}
	}
	return nil
}

// AfterValidate ensures role is lowercase
func (m *MessageWithHooks) AfterValidate() error {
	m.Role = strings.ToLower(m.Role)
	return nil
}

func TestBeforeValidateHook(t *testing.T) {
	validator := godantic.NewValidator[MessageWithHooks]()

	t.Run("string content normalized to array", func(t *testing.T) {
		jsonStr := `{"role": "user", "content": "hello world"}`
		msg, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) > 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}

		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content item, got %d", len(msg.Content))
		}
		if msg.Content[0] != "hello world" {
			t.Errorf("expected 'hello world', got %s", msg.Content[0])
		}
	})

	t.Run("array content remains array", func(t *testing.T) {
		jsonStr := `{"role": "user", "content": ["hello", "world"]}`
		msg, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) > 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}

		if len(msg.Content) != 2 {
			t.Fatalf("expected 2 content items, got %d", len(msg.Content))
		}
	})
}

func TestAfterValidateHook(t *testing.T) {
	validator := godantic.NewValidator[MessageWithHooks]()

	jsonStr := `{"role": "USER", "content": ["hello"]}`
	msg, errs := validator.Unmarshal([]byte(jsonStr))
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user' (lowercased), got %s", msg.Role)
	}
}

type OutputTransformer struct {
	Name string `json:"name"`
}

func (o *OutputTransformer) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// BeforeSerialize adds a prefix
func (o *OutputTransformer) BeforeSerialize() error {
	o.Name = "prefix_" + o.Name
	return nil
}

// AfterSerialize wraps JSON in envelope
func (o *OutputTransformer) AfterSerialize(data []byte) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return data, err
	}

	envelope := map[string]any{
		"data":    obj,
		"version": "1.0",
	}

	return json.Marshal(envelope)
}

func TestBeforeSerializeHook(t *testing.T) {
	validator := godantic.NewValidator[OutputTransformer]()

	obj := OutputTransformer{Name: "test"}
	data, errs := validator.Marshal(&obj)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check envelope structure
	if result["version"] != "1.0" {
		t.Errorf("expected version 1.0, got %v", result["version"])
	}

	dataField, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data field to be object, got %T", result["data"])
	}

	if dataField["name"] != "prefix_test" {
		t.Errorf("expected 'prefix_test', got %v", dataField["name"])
	}
}

// Test BeforeValidate with discriminated unions

type MsgBlock interface {
	GetBlockType() string
}

type MsgTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t MsgTextBlock) GetBlockType() string { return t.Type }

func (t *MsgTextBlock) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.OneOf("text"))
}

func (t *MsgTextBlock) FieldText() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type MsgData struct {
	Role    string     `json:"role"`
	Content []MsgBlock `json:"content"`
}

func (m *MsgData) FieldRole() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (m *MsgData) FieldContent() godantic.FieldOptions[[]MsgBlock] {
	return godantic.Field(
		godantic.Required[[]MsgBlock](),
		godantic.DiscriminatedUnion[[]MsgBlock]("type", map[string]any{
			"text": MsgTextBlock{},
		}),
	)
}

// BeforeValidate transforms string content to array (Python compatibility)
func (m *MsgData) BeforeValidate(raw map[string]any) error {
	if contentStr, ok := raw["content"].(string); ok {
		raw["content"] = []map[string]any{
			{"type": "text", "text": contentStr},
		}
	}
	return nil
}

func TestBeforeValidateWithDiscriminatedUnionSlice(t *testing.T) {
	validator := godantic.NewValidator[MsgData]()

	t.Run("string content normalized to array of discriminated unions", func(t *testing.T) {
		jsonStr := `{"role": "user", "content": "hello world"}`
		msg, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) > 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}

		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(msg.Content))
		}

		textBlock, ok := msg.Content[0].(MsgTextBlock)
		if !ok {
			t.Fatalf("expected MsgTextBlock, got %T", msg.Content[0])
		}

		if textBlock.Text != "hello world" {
			t.Errorf("expected 'hello world', got %s", textBlock.Text)
		}
	})

	t.Run("array content remains array", func(t *testing.T) {
		jsonStr := `{"role": "user", "content": [{"type": "text", "text": "hello"}]}`
		msg, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) > 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}

		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(msg.Content))
		}
	})
}
