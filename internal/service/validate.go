package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
)

const maxKeyLength = 255

func validateKey(key string) error {
	if len(key) > maxKeyLength {
		return model.Invalidf("Idempotency-Key must be at most %d characters", maxKeyLength)
	}
	return nil
}

func (s *Service) validateBooking(req *model.BookingRequest) error {
	req.GuestName = strings.TrimSpace(req.GuestName)
	if req.GuestName == "" {
		return model.Invalidf("guest_name must not be blank")
	}
	return s.validateStay(req.CheckIn, req.CheckOut)
}

// validateStay checks the dates of a stay: both set, check-out after check-in,
// not in the past, and not longer than maxNights.
func (s *Service) validateStay(checkIn, checkOut model.Date) error {
	if checkIn.IsZero() || checkOut.IsZero() {
		return model.Invalidf("check_in and check_out are required")
	}
	if !checkOut.Time().After(checkIn.Time()) {
		return model.Invalidf("check_out must be after check_in")
	}
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if checkIn.Time().Before(today) {
		return model.Invalidf("check_in must not be in the past")
	}
	if n := checkIn.Nights(checkOut); n > maxNights {
		return model.Invalidf("stay of %d nights exceeds the maximum of %d", n, maxNights)
	}
	return nil
}

// hashRequest fingerprints the request body, it reflects exactly what the client sent.
func hashRequest(req model.BookingRequest) (string, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("hash request: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
