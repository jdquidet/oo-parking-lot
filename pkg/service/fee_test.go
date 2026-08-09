package service

import (
	"errors"
	"testing"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

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
			expected: 5280.0, // (2*5000 + 3*100) - 5020 = 10300 - 5020
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

func TestCalculateFee_ChainedSessions(t *testing.T) {
	// Setup LP chain
	sessionL1 := &domain.ParkingSession{
		EntryTime:       baseTime,
		ExitTime:        timePtr(baseTime.Add(2 * time.Hour)),
		TotalFeeCharged: 40.0,
	}
	sessionL2 := &domain.ParkingSession{
		EntryTime:       baseTime.Add(2*time.Hour + 15*time.Minute),
		ExitTime:        timePtr(baseTime.Add(2*time.Hour + 30*time.Minute)),
		TotalFeeCharged: 0.0,
		PreviousSession: sessionL1,
	}
	// Setup MP chain
	sessionM1 := &domain.ParkingSession{
		EntryTime:       baseTime,
		ExitTime:        timePtr(baseTime.Add(22 * time.Hour)),
		TotalFeeCharged: 1180.0,
	}
	sessionM2 := &domain.ParkingSession{
		EntryTime:       baseTime.Add(22*time.Hour + 15*time.Minute),
		ExitTime:        timePtr(baseTime.Add(22*time.Hour + 30*time.Minute)),
		TotalFeeCharged: 60.0,
		PreviousSession: sessionM1,
	}
	// Setup multi-slot chain: SP(2h) -> MP(2h + 15m gap) -> LP(2h + 15m gap)
	sessionSP := &domain.ParkingSession{
		EntryTime:       baseTime,
		ExitTime:        timePtr(baseTime.Add(2 * time.Hour)),
		TotalFeeCharged: 40.0,
	}
	sessionMP := &domain.ParkingSession{
		EntryTime:       baseTime.Add(2*time.Hour + 15*time.Minute),
		ExitTime:        timePtr(baseTime.Add(4*time.Hour + 15*time.Minute)),
		TotalFeeCharged: 120.0,
		PreviousSession: sessionSP,
	}
	// Setup 24h boundary crossing: SP(23h) -> LP(2h)
	sessionChunk := &domain.ParkingSession{
		EntryTime:       baseTime,
		ExitTime:        timePtr(baseTime.Add(23 * time.Hour)),
		TotalFeeCharged: 440.0, // 40 + 20*20
	}

	tests := []struct {
		name        string
		slot        domain.SlotSize
		entryTime   time.Time
		exitTime    time.Time
		lastSession *domain.ParkingSession
		expected    float64
	}{
		{
			name:        "3-session LP chain: Prevent hourly rate bypass",
			slot:        domain.SlotLP,
			entryTime:   baseTime.Add(2*time.Hour + 45*time.Minute),
			exitTime:    baseTime.Add(4*time.Hour + 45*time.Minute),
			lastSession: sessionL2,
			expected:    200.0, // 5h LP (240) - 3h LP (40)
		},
		{
			name:        "3-session MP chain: Prevent 24h threshold bypass",
			slot:        domain.SlotMP,
			entryTime:   baseTime.Add(22*time.Hour + 45*time.Minute),
			exitTime:    baseTime.Add(44*time.Hour + 45*time.Minute),
			lastSession: sessionM2,
			expected:    5020.0, // 45h MP (6260) - 23h MP (1240)
		},
		{
			name:        "Multi-slot chain: SP(2h) -> MP(2h+15m gap) -> LP(2h+15m gap) = 460",
			slot:        domain.SlotLP,
			entryTime:   baseTime.Add(4*time.Hour + 30*time.Minute),
			exitTime:    baseTime.Add(6*time.Hour + 30*time.Minute),
			lastSession: sessionMP,
			expected:    300.0, // ceil(2h15m)=3h at LP rate 100
		},
		{
			name:        "Multi-slot chain: 24h boundary crossing SP(23h) -> LP(2h) = 4660",
			slot:        domain.SlotLP,
			entryTime:   baseTime.Add(23*time.Hour + 15*time.Minute),
			exitTime:    baseTime.Add(25 * time.Hour),
			lastSession: sessionChunk,
			expected:    4660.0, // (5000 + 1*100) - 440
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

func TestSumPreviousFees_ChainBoundary(t *testing.T) {
	// Day 1: Session 0 (2h on SP) -> Paid 40.0. Exits Day 1, 02:00.
	// Gap of 22 hours (>1h gap)
	session0Old := &domain.ParkingSession{
		EntryTime:       baseTime.Add(-24 * time.Hour),
		ExitTime:        timePtr(baseTime.Add(-22 * time.Hour)),
		TotalFeeCharged: 40.0,
	}
	// Day 2: Session 1 (23h on SP) -> Paid 440.0. Enters Day 2, 00:00. Exits Day 2, 23:00.
	// Has pointer to session0Old, but separated by >1h gap.
	session1Active := &domain.ParkingSession{
		EntryTime:       baseTime,
		ExitTime:        timePtr(baseTime.Add(23 * time.Hour)),
		TotalFeeCharged: 440.0,
		PreviousSession: session0Old,
	}

	tests := []struct {
		name        string
		slot        domain.SlotSize
		entryTime   time.Time
		exitTime    time.Time
		lastSession *domain.ParkingSession
		expected    float64
	}{
		{
			name:        "Excludes fees prior to >1h gap when crossing 24h chunk boundary",
			slot:        domain.SlotLP,
			entryTime:   baseTime.Add(23*time.Hour + 15*time.Minute),
			exitTime:    baseTime.Add(25*time.Hour + 15*time.Minute),
			lastSession: session1Active,
			expected:    4760.0, // 26h LP (5200) - 23h SP (440) = 4760
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
