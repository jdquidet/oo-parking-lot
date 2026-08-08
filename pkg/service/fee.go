package service

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jdquidet/oo-parking-lot/pkg/domain"
)

const (
	FlatRate      = 40.0
	FlatRateHours = 3
	Chunk24hRate  = 5000.0
)

var (
	ErrInvalidTimeOrder = errors.New("exit time can't be before entry time")
	ErrUnknownSlotSize  = errors.New("unknown or unsupported parking slot size")
)

// CalculateFee calculates base fee with additional fee based on re-entry within an hour.
func CalculateFee(
	slotSize domain.SlotSize,
	entryTime, exitTime time.Time,
	lastSession *domain.ParkingSession,
) (float64, error) {
	if slotSize < domain.SlotSP || slotSize > domain.SlotLP {
		return 0.0, fmt.Errorf("%w: %v", ErrUnknownSlotSize, slotSize)
	}
	if exitTime.Before(entryTime) {
		return 0.0, fmt.Errorf("%w: entry %v is after exit %v", ErrInvalidTimeOrder, entryTime, exitTime)
	}

	// Computes as a standalone fee if last session is nonexistent or missing exit time
	if lastSession == nil || lastSession.ExitTime == nil {
		return computeBaseFee(slotSize, entryTime, exitTime), nil
	}
	// Computes as a standalone fee if not a quick re-entry
	timeGap := entryTime.Sub(*lastSession.ExitTime)
	if timeGap > 1*time.Hour || timeGap < 0 {
		return computeBaseFee(slotSize, entryTime, exitTime), nil
	}

	// Initiates calculation of continuous fee for quick re-entry
	totalDuration := exitTime.Sub(lastSession.EntryTime)
	prevDuration := lastSession.ExitTime.Sub(lastSession.EntryTime)
	totalHours := int(math.Ceil(totalDuration.Hours()))
	prevHours := int(math.Ceil(prevDuration.Hours()))
	if totalHours <= prevHours {
		return 0.0, nil
	}
	// Computes total fee of previous and active session with the active slot size
	// Subtracts phantom fee (previous session fee with active slot size) after
	totalFeeForSlot := computeFeeForHours(slotSize, totalHours)
	prevFeeForSlot := computeFeeForHours(slotSize, prevHours)
	fee := totalFeeForSlot - prevFeeForSlot
	if fee < 0 {
		return 0.0, nil
	}
	return fee, nil
}

// HourlyRate returns the hourly rate by parking slot size.
func HourlyRate(s domain.SlotSize) float64 {
	switch s {
	case domain.SlotSP:
		return 20.0
	case domain.SlotMP:
		return 60.0
	case domain.SlotLP:
		return 100.0
	default:
		return 0.0
	}
}

// computeFeeForHours is the core pricing formula for a given total hour duration.
func computeFeeForHours(slotSize domain.SlotSize, hours int) float64 {
	if hours <= 0 {
		return 0.0
	}

	// 24-hour chunk pricing
	if days := hours / 24; days > 0 {
		extraHours := hours % 24
		return (float64(days) * Chunk24hRate) + (float64(extraHours) * HourlyRate(slotSize))
	}

	// Under 24 hours pricing
	fee := FlatRate
	if extraHours := hours - FlatRateHours; extraHours > 0 {
		fee += float64(extraHours) * HourlyRate(slotSize)
	}
	return fee
}

// computeBaseFee computes total parking fee for a single session.
func computeBaseFee(slotSize domain.SlotSize, entryTime, exitTime time.Time) float64 {
	duration := exitTime.Sub(entryTime)
	if duration <= 0 {
		return 0.0
	}
	hours := int(math.Ceil(duration.Hours()))
	return computeFeeForHours(slotSize, hours)
}
