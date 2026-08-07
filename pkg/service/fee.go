package service

import (
	"github.com/jdquidet/oo-parking-lot/pkg/domain"
	"math"
	"time"
)

const (
	FlatRate      = 40.0
	FlatRateHours = 3
	Chunk24hRate  = 5000.0
)

// Hourly rate by parking slot size
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

// CalculateBaseFee computes total parking fee without taking prior continuous sessions into account
func CalculateBaseFee(s domain.SlotSize, entryTime, exitTime time.Time) float64 {
	duration := exitTime.Sub(entryTime)
	if duration <= 0 {
		return 0.0
	}

	totalHours := int(math.Ceil(duration.Hours()))
	chunks24h := totalHours / 24
	remainderHours := totalHours % 24

	// Preemptive calculation of 24h chunks
	totalFee := float64(chunks24h) * Chunk24hRate

	if chunks24h > 0 {
		// 24h chunks + remainder hours, flat rate not considered
		totalFee += float64(remainderHours) * HourlyRate(s)
	} else {
		// Flat rate + extra hours, no 24h chunks calculated
		if totalHours <= FlatRateHours {
			totalFee += FlatRate
		} else {
			extraHours := totalHours - FlatRateHours
			totalFee += FlatRate + (float64(extraHours) * HourlyRate(s))
		}
	}
	return totalFee
}
