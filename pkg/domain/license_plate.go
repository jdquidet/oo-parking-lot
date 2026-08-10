package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrInvalidLicensePlate = errors.New("invalid license plate")

var licensePlatePattern = regexp.MustCompile(`^[A-Z0-9]+(?:-[A-Z0-9]+)*$`)

// ValidateLicensePlate trims surrounding whitespace, converts the plate to uppercase,
// and validates that it contains ASCII alphanumeric groups separated by single dashes.
func ValidateLicensePlate(licensePlate string) (string, error) {
	valid := strings.ToUpper(strings.TrimSpace(licensePlate))
	if !licensePlatePattern.MatchString(valid) {
		return "", fmt.Errorf("%w: %q must contain ASCII letters or digits separated by single dashes", ErrInvalidLicensePlate, licensePlate)
	}
	return valid, nil
}
