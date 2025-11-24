package godantic_bench

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ============================================================================
// Benchmark Fixtures
// ============================================================================

// Simple discriminated union (2 variants)
type SimpleAnimal interface {
	IsAnimal()
}

type SimpleDog struct {
	Species string `json:"species"`
	Breed   string `json:"breed"`
}

func (SimpleDog) IsAnimal() {}

type SimpleCat struct {
	Species string `json:"species"`
	Lives   int    `json:"lives"`
}

func (SimpleCat) IsAnimal() {}

type SimplePetOwner struct {
	Name string       `json:"name"`
	Pet  SimpleAnimal `json:"pet"`
}

func (p *SimplePetOwner) FieldPet() godantic.FieldOptions[SimpleAnimal] {
	return godantic.Field(
		godantic.Required[SimpleAnimal](),
		godantic.DiscriminatedUnion[SimpleAnimal](
			"species",
			map[string]any{
				"dog": SimpleDog{},
				"cat": SimpleCat{},
			},
		),
	)
}

// Nested discriminated union (2 levels deep)
type Payment interface {
	IsPayment()
}

type CreditCard struct {
	Type       string `json:"type"`
	CardNumber string `json:"card_number"`
	CVV        string `json:"cvv"`
}

func (CreditCard) IsPayment() {}

type BankTransfer struct {
	Type          string `json:"type"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
}

func (BankTransfer) IsPayment() {}

type Transaction struct {
	ID     string  `json:"id"`
	Method Payment `json:"method"`
}

func (t *Transaction) FieldMethod() godantic.FieldOptions[Payment] {
	return godantic.Field(
		godantic.Required[Payment](),
		godantic.DiscriminatedUnion[Payment](
			"type",
			map[string]any{
				"credit_card":   CreditCard{},
				"bank_transfer": BankTransfer{},
			},
		),
	)
}

type Order struct {
	OrderID     string      `json:"order_id"`
	Transaction Transaction `json:"transaction"`
}

func (o *Order) FieldTransaction() godantic.FieldOptions[Transaction] {
	return godantic.Field(godantic.Required[Transaction]())
}

// Many variants (10 types)
type Event interface {
	IsEvent()
}

type ClickEvent struct {
	Type string `json:"type"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}
type ScrollEvent struct {
	Type  string `json:"type"`
	Delta int    `json:"delta"`
}
type KeyEvent struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}
type MouseEvent struct {
	Type   string `json:"type"`
	Button int    `json:"button"`
}
type TouchEvent struct {
	Type    string `json:"type"`
	Touches int    `json:"touches"`
}
type ResizeEvent struct {
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
type LoadEvent struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}
type ErrorEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
type FocusEvent struct {
	Type      string `json:"type"`
	ElementID string `json:"element_id"`
}
type BlurEvent struct {
	Type      string `json:"type"`
	ElementID string `json:"element_id"`
}

func (ClickEvent) IsEvent()  {}
func (ScrollEvent) IsEvent() {}
func (KeyEvent) IsEvent()    {}
func (MouseEvent) IsEvent()  {}
func (TouchEvent) IsEvent()  {}
func (ResizeEvent) IsEvent() {}
func (LoadEvent) IsEvent()   {}
func (ErrorEvent) IsEvent()  {}
func (FocusEvent) IsEvent()  {}
func (BlurEvent) IsEvent()   {}

type EventLog struct {
	Timestamp int64 `json:"timestamp"`
	Event     Event `json:"event"`
}

func (e *EventLog) FieldEvent() godantic.FieldOptions[Event] {
	return godantic.Field(
		godantic.Required[Event](),
		godantic.DiscriminatedUnion[Event](
			"type",
			map[string]any{
				"click":  ClickEvent{},
				"scroll": ScrollEvent{},
				"key":    KeyEvent{},
				"mouse":  MouseEvent{},
				"touch":  TouchEvent{},
				"resize": ResizeEvent{},
				"load":   LoadEvent{},
				"error":  ErrorEvent{},
				"focus":  FocusEvent{},
				"blur":   BlurEvent{},
			},
		),
	)
}

// ============================================================================
// Key Benchmark: Discriminated Union Unmarshal
// Compares godantic's automatic discriminated union handling vs manual code
// ============================================================================

// Godantic: Automatic discriminated union via FieldPet()
func BenchmarkDiscriminator_Godantic(b *testing.B) {
	validator := godantic.NewValidator[SimplePetOwner]()
	data := []byte(`{"name":"John","pet":{"species":"dog","breed":"Labrador"}}`)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		result, errs := validator.Marshal(data)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
		_ = result
	}
}

// Manual: Custom UnmarshalJSON implementation
type ManualPetOwner struct {
	Name string    `json:"name"`
	Pet  ManualPet `json:"pet"`
}

type ManualPet struct {
	Species string
	Dog     *SimpleDog
	Cat     *SimpleCat
}

func (p *ManualPet) UnmarshalJSON(data []byte) error {
	var temp struct {
		Species string `json:"species"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	p.Species = temp.Species
	switch temp.Species {
	case "dog":
		var dog SimpleDog
		if err := json.Unmarshal(data, &dog); err != nil {
			return err
		}
		p.Dog = &dog
	case "cat":
		var cat SimpleCat
		if err := json.Unmarshal(data, &cat); err != nil {
			return err
		}
		p.Cat = &cat
	}
	return nil
}

func BenchmarkDiscriminator_Manual(b *testing.B) {
	data := []byte(`{"name":"John","pet":{"species":"dog","breed":"Labrador"}}`)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		var owner ManualPetOwner
		if err := json.Unmarshal(data, &owner); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}

// ============================================================================
// Supporting Benchmark: Validator Creation Cost
// Shows one-time setup cost (validators should be reused in production)
// ============================================================================

func BenchmarkDiscriminator_ValidatorCreation(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = godantic.NewValidator[SimplePetOwner]()
	}
}
