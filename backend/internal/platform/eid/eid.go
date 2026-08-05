package eid

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type EIDIdentity struct {
	CivilID        string `json:"civil_id"`
	RegNumber      string `json:"reg_number"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	VerifiedStatus bool   `json:"verified_status"`
}

type Provider interface {
	VerifyToken(ctx context.Context, ssoToken string) (*EIDIdentity, error)
	AuthenticateRegNumber(ctx context.Context, regNumber, otpCode string) (*EIDIdentity, error)
}

type EIDService struct {
	mockMode bool
	apiURL   string
}

func NewEIDService() *EIDService {
	mock := os.Getenv("EID_MOCK_MODE") != "false"
	apiURL := os.Getenv("EID_API_URL")
	if apiURL == "" {
		apiURL = "https://eid.gerege.mn/api/v1"
	}
	return &EIDService{
		mockMode: mock,
		apiURL:   apiURL,
	}
}

func (s *EIDService) VerifyToken(ctx context.Context, ssoToken string) (*EIDIdentity, error) {
	if ssoToken == "" {
		return nil, errors.New("empty E-ID SSO token")
	}

	if s.mockMode {
		// Mock E-ID Verification Response for development & testing
		regNo := "AA90010111"
		if strings.HasPrefix(ssoToken, "eid_") {
			parts := strings.Split(ssoToken, "_")
			if len(parts) > 1 {
				regNo = parts[1]
			}
		}
		return &EIDIdentity{
			CivilID:        "CID-99887766",
			RegNumber:      regNo,
			FirstName:      "Bat",
			LastName:       "Bold",
			Email:          "bat.bold@example.mn",
			Phone:          "99112233",
			VerifiedStatus: true,
		}, nil
	}

	return nil, fmt.Errorf("E-ID SSO service live integration requires production credentials")
}

func (s *EIDService) AuthenticateRegNumber(ctx context.Context, regNumber, otpCode string) (*EIDIdentity, error) {
	if len(regNumber) < 8 {
		return nil, errors.New("invalid registration number format")
	}

	if s.mockMode {
		return &EIDIdentity{
			CivilID:        "CID-" + regNumber,
			RegNumber:      strings.ToUpper(regNumber),
			FirstName:      "Verified",
			LastName:       "Citizen",
			Email:          strings.ToLower(regNumber) + "@eid.mn",
			Phone:          "99001122",
			VerifiedStatus: true,
		}, nil
	}

	return nil, fmt.Errorf("E-ID OTP authentication live service unreachable")
}
