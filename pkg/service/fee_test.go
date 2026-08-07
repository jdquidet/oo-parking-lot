package service

import (
	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"testing"
	"time"
)

func TestCalculateBaseFee(t *testing.T) {
	tests := []struct {
		name     string
		slot     domain.SlotSize
		duration time.Duration
		expected float64
	}{
		{
			name:     "Under 3h (SP): Flat rate 40",
			slot:     domain.SlotSP,
			duration: 2 * time.Hour,
			expected: 40.0,
		},
		{
			name:     "Exact 3h (MP): Flat rate 40",
			slot:     domain.SlotMP,
			duration: 3 * time.Hour,
			expected: 40.0,
		},
		{
			name:     "Round up 6.4h (SP): 40+(4*20) = 120",
			slot:     domain.SlotSP,
			duration: 6*time.Hour + 24*time.Minute,
			expected: 120.0,
		},
		{
			name:     "Exact 5h (LP): 40+(2*100) = 240",
			slot:     domain.SlotLP,
			duration: 5 * time.Hour,
			expected: 240.0,
		},
		{
			name:     "Exact 24h: Flat chunk 5000",
			slot:     domain.SlotSP,
			duration: 24 * time.Hour,
			expected: 5000.0,
		},
		{
			name:     "Exact 26h (MP): 5000+(2*60) = 5120",
			slot:     domain.SlotMP,
			duration: 26 * time.Hour,
			expected: 5120.0,
		},
		{
			name:     "Exact 49h (LP): (2*5000)+100 = 10100",
			slot:     domain.SlotLP,
			duration: 49 * time.Hour,
			expected: 10100.0,
		},
	}

	now := time.Now()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := CalculateBaseFee(tt.slot, now, now.Add(tt.duration))
			if fee != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, fee)
			}
		})
	}
}
