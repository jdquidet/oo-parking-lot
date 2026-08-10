package domain

import (
	"errors"
	"testing"
)

func TestValidateLicensePlate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "trims and uppercases", input: "  abc-123  ", expected: "ABC-123"},
		{name: "accepts one alphanumeric group", input: "a1b2c3", expected: "A1B2C3"},
		{name: "accepts multiple groups", input: "ab-12-cd34", expected: "AB-12-CD34"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateLicensePlate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestValidateLicensePlateRejectsInvalidPlates(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "spaces only", input: "   "},
		{name: "internal space", input: "ABC 123"},
		{name: "punctuation", input: "ABC.123"},
		{name: "leading dash", input: "-ABC123"},
		{name: "trailing dash", input: "ABC123-"},
		{name: "consecutive dashes", input: "ABC--123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateLicensePlate(tt.input)
			if !errors.Is(err, ErrInvalidLicensePlate) {
				t.Fatalf("expected ErrInvalidLicensePlate, got %v", err)
			}
			if got != "" {
				t.Errorf("expected empty valid plate, got %q", got)
			}
		})
	}
}
