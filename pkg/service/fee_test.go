package service

import (
	"errors"
	"testing"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

var baseTime = time.Date(2026, time.August, 11, 17, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

func TestComputeBaseFee(t *testing.T) {
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
			name:     "Zero or negative duration",
			slot:     domain.SlotSP,
			duration: -1 * time.Hour,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := computeBaseFee(tt.slot, baseTime, baseTime.Add(tt.duration))
			if fee != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, fee)
			}
		})
	}
}

func TestCalculateFee_StandaloneSessions(t *testing.T) {
	tests := []struct {
		name        string
		slot        domain.SlotSize
		entryTime   time.Time
		exitTime    time.Time
		lastSession *domain.ParkingSession
		expected    float64
		expectedErr error
	}{
		{
			name:        "No last session: computes base fee",
			slot:        domain.SlotSP,
			entryTime:   baseTime,
			exitTime:    baseTime.Add(2 * time.Hour),
			lastSession: nil,
			expected:    40.0,
		},
		{
			name:      "Last session missing exit time: computes base fee",
			slot:      domain.SlotMP,
			entryTime: baseTime,
			exitTime:  baseTime.Add(4 * time.Hour),
			lastSession: &domain.ParkingSession{
				EntryTime: baseTime.Add(-10 * time.Hour),
				ExitTime:  nil,
			},
			expected: 100.0, // 40 + (1 * 60)
		},
		{
			name:      "Re-entry > 1 hour: computes standalone base fee",
			slot:      domain.SlotSP,
			entryTime: baseTime.Add(2 * time.Hour), // 1.5h gap from previous session
			exitTime:  baseTime.Add(4 * time.Hour),
			lastSession: &domain.ParkingSession{
				EntryTime:       baseTime,
				ExitTime:        timePtr(baseTime.Add(30 * time.Minute)),
				TotalFeeCharged: 40.0,
			},
			expected: 40.0,
		},
		{
			name:      "Negative time gap anomaly: computes base fee",
			slot:      domain.SlotSP,
			entryTime: baseTime.Add(-5 * time.Minute), // 1.5h gap from previous session
			exitTime:  baseTime.Add(2 * time.Hour),
			lastSession: &domain.ParkingSession{
				EntryTime:       baseTime.Add(-10 * time.Hour),
				ExitTime:        timePtr(baseTime),
				TotalFeeCharged: 180.0,
			},
			expected: 40.0,
		},
		{
			name:        "Returns ErrInvalidTimeOrder when exit time is before entry time",
			slot:        domain.SlotSP,
			entryTime:   baseTime.Add(2 * time.Hour),
			exitTime:    baseTime, // Exit before entry
			lastSession: nil,
			expectedErr: ErrInvalidTimeOrder,
		},
		{
			name:        "Returns ErrUnknownSlotSize when slot size is unknown",
			slot:        domain.SlotSize(99),
			entryTime:   baseTime,
			exitTime:    baseTime.Add(2 * time.Hour),
			lastSession: nil,
			expectedErr: ErrUnknownSlotSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee, err := CalculateFee(tt.slot, tt.entryTime, tt.exitTime, tt.lastSession)
			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fee != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, fee)
			}
		})
	}
}

func TestCalculateFee_ContinuedSessions(t *testing.T) {
	tests := []struct {
		name        string
		slot        domain.SlotSize
		entryTime   time.Time
		exitTime    time.Time
		lastSession *domain.ParkingSession
		expected    float64
	}{
		{
			name:      "Quick re-entry within 30m (Same SP slot): 4h + 30m + 1h",
			slot:      domain.SlotSP,
			entryTime: baseTime.Add(4*time.Hour + 30*time.Minute),
			exitTime:  baseTime.Add(5*time.Hour + 30*time.Minute),
			lastSession: &domain.ParkingSession{
				EntryTime:       baseTime,
				ExitTime:        timePtr(baseTime.Add(4 * time.Hour)),
				TotalFeeCharged: 60.0,
			},
			expected: 40.0,
		},
		{
			name:      "Quick re-entry with slot switch (SP -> LP): SP(4h) + LP(30m+1h)",
			slot:      domain.SlotLP,
			entryTime: baseTime.Add(4*time.Hour + 30*time.Minute),
			exitTime:  baseTime.Add(5*time.Hour + 30*time.Minute),
			lastSession: &domain.ParkingSession{
				EntryTime:       baseTime,
				ExitTime:        timePtr(baseTime.Add(4 * time.Hour)),
				TotalFeeCharged: 60.0,
			},
			expected: 200.0,
		},
		{
			name:      "24h+ last session with 24h+ continued session  (SP -> LP): SP(25h) + LP(30m+25h)",
			slot:      domain.SlotLP,
			entryTime: baseTime.Add(25*time.Hour + 30*time.Minute),
			exitTime:  baseTime.Add(50*time.Hour + 30*time.Minute),
			lastSession: &domain.ParkingSession{
				EntryTime:       baseTime,
				ExitTime:        timePtr(baseTime.Add(25 * time.Hour)),
				TotalFeeCharged: 5020.0, // 1*5000 + 1*20
			},
			expected: 5200.0, // 1*5000 + 2*100
		},
		{
			name:      "Re-entry within 30m, still under 3h window: 1h + 15m + 45m",
			slot:      domain.SlotSP,
			entryTime: baseTime.Add(1*time.Hour + 15*time.Minute),
			exitTime:  baseTime.Add(2 * time.Hour),
			lastSession: &domain.ParkingSession{
				EntryTime:       baseTime,
				ExitTime:        timePtr(baseTime.Add(1 * time.Hour)),
				TotalFeeCharged: 40.0,
			},
			expected: 0.0, // Flat rate already paid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee, err := CalculateFee(tt.slot, tt.entryTime, tt.exitTime, tt.lastSession)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fee != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, fee)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
