package main

import (
	"fmt"

	"github.com/deepankarm/godantic/pkg/godantic"
)

type PaymentType string

const (
	PaymentTypeCreditCard   PaymentType = "credit_card"
	PaymentTypePayPal       PaymentType = "paypal"
	PaymentTypeBankTransfer PaymentType = "bank_transfer"
)

// PaymentMethod is the interface all payment types must implement
type PaymentMethod interface {
	GetType() PaymentType
}

// CreditCardPayment represents a credit card payment
type CreditCardPayment struct {
	Type       PaymentType `json:"type"`
	CardNumber string      `json:"card_number"`
}

func (c CreditCardPayment) GetType() PaymentType { return c.Type }

func (c *CreditCardPayment) FieldType() godantic.FieldOptions[PaymentType] {
	return godantic.Field(godantic.Const(PaymentTypeCreditCard))
}

// PayPalPayment represents a PayPal payment
type PayPalPayment struct {
	Type  PaymentType `json:"type"`
	Email string      `json:"email"`
}

func (p PayPalPayment) GetType() PaymentType { return p.Type }

func (p *PayPalPayment) FieldType() godantic.FieldOptions[PaymentType] {
	return godantic.Field(godantic.Const(PaymentTypePayPal))
}

// BankTransferPayment represents a bank transfer payment
type BankTransferPayment struct {
	Type        PaymentType `json:"type"`
	AccountName string      `json:"account_name"`
}

func (b BankTransferPayment) GetType() PaymentType { return b.Type }

func (b *BankTransferPayment) FieldType() godantic.FieldOptions[PaymentType] {
	return godantic.Field(godantic.Const(PaymentTypeBankTransfer))
}

func main() {
	// Create validator with discriminator configuration
	validator := godantic.NewValidator[PaymentMethod](
		godantic.WithDiscriminatorTyped("type", map[PaymentType]any{
			PaymentTypeCreditCard:   CreditCardPayment{},
			PaymentTypePayPal:       PayPalPayment{},
			PaymentTypeBankTransfer: BankTransferPayment{},
		}),
	)

	// Valid credit card payment
	creditCardJSON := `{"type": "credit_card", "card_number": "4532015112830366"}`
	payment, errs := validator.ValidateJSON([]byte(creditCardJSON))
	if errs != nil {
		fmt.Printf("Validation failed: %v\n", errs)
	} else {
		if cc, ok := (*payment).(CreditCardPayment); ok {
			fmt.Printf("Credit card payment: %s\n", cc.CardNumber)
		}
	}

	// Valid PayPal payment
	paypalJSON := `{"type": "paypal", "email": "user@example.com"}`
	payment, errs = validator.ValidateJSON([]byte(paypalJSON))
	if errs != nil {
		fmt.Printf("Validation failed: %v\n", errs)
	} else {
		if pp, ok := (*payment).(PayPalPayment); ok {
			fmt.Printf("PayPal payment: %s\n", pp.Email)
		}
	}

	// Invalid payment type
	invalidJSON := `{"type": "cryptocurrency", "wallet": "0x1234"}`
	_, errs = validator.ValidateJSON([]byte(invalidJSON))
	if errs != nil {
		fmt.Printf("Invalid type error: %s\n", errs[0].Message)
	}

	// Process multiple payments with type switching
	payments := []string{
		`{"type": "credit_card", "card_number": "5425233430109903"}`,
		`{"type": "paypal", "email": "customer@domain.com"}`,
		`{"type": "bank_transfer", "account_name": "John Doe"}`,
	}

	for _, paymentJSON := range payments {
		payment, errs := validator.ValidateJSON([]byte(paymentJSON))
		if errs != nil {
			continue
		}

		switch p := (*payment).(type) {
		case CreditCardPayment:
			fmt.Printf("Processing credit card: %s\n", p.CardNumber)
		case PayPalPayment:
			fmt.Printf("Processing PayPal: %s\n", p.Email)
		case BankTransferPayment:
			fmt.Printf("Processing bank transfer: %s\n", p.AccountName)
		}
	}
}
