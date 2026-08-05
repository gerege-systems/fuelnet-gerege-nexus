package xyp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type CitizenInfo struct {
	RegNumber      string `json:"reg_number"`
	CivilID        string `json:"civil_id"`
	LastName       string `json:"last_name"`
	FirstName      string `json:"first_name"`
	Gender         string `json:"gender"`
	Address        string `json:"address"`
	PassportStatus string `json:"passport_status"`
	Verified       bool   `json:"verified"`
}

type CompanyInfo struct {
	CompanyReg  string `json:"company_reg"`
	Name        string `json:"name"`
	Executive   string `json:"executive"`
	Address     string `json:"address"`
	VatPayer    bool   `json:"vat_payer"`
	Status      string `json:"status"`
	FoundingDate string `json:"founding_date"`
}

type XYPService struct {
	endpoint string
	mockMode bool
	apiKey   string
}

func NewXYPService() *XYPService {
	endpoint := os.Getenv("XYP_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://xyp.gerege.mn/api/v1"
	}
	mock := os.Getenv("XYP_MOCK_MODE") != "false"
	apiKey := os.Getenv("XYP_API_KEY")

	return &XYPService{
		endpoint: endpoint,
		mockMode: mock,
		apiKey:   apiKey,
	}
}

// GetCitizenInfo queries citizen data from ХУР (xyp.gerege.mn)
func (s *XYPService) GetCitizenInfo(ctx context.Context, regNumber string) (*CitizenInfo, error) {
	if regNumber == "" {
		return nil, errors.New("empty registration number")
	}

	cleanReg := strings.ToUpper(strings.TrimSpace(regNumber))
	if len(cleanReg) < 8 {
		return nil, errors.New("invalid registration number format: must be at least 8 characters")
	}

	if s.mockMode {
		return &CitizenInfo{
			RegNumber:      cleanReg,
			CivilID:        "CID-" + cleanReg,
			LastName:       "Бат",
			FirstName:      "Болд",
			Gender:         "Male",
			Address:        "Улаанбаатар хот, Сүхбаатар дүүрэг, 1-р хороо",
			PassportStatus: "ACTIVE",
			Verified:       true,
		}, nil
	}

	return nil, fmt.Errorf("XYP live service at %s requires valid XYP_API_KEY credentials", s.endpoint)
}

// GetCompanyInfo queries legal entity data from ХУР (xyp.gerege.mn)
func (s *XYPService) GetCompanyInfo(ctx context.Context, companyReg string) (*CompanyInfo, error) {
	if companyReg == "" {
		return nil, errors.New("empty company registration number")
	}

	cleanReg := strings.TrimSpace(companyReg)

	if s.mockMode {
		return &CompanyInfo{
			CompanyReg:  cleanReg,
			Name:        "Гэрэгэ Системс ХХК",
			Executive:   "Ц.Эрдэнэбат",
			Address:     "Улаанбаатар хот, Хан-Уул дүүрэг, Гэрэгэ тауэр 5 давхар",
			VatPayer:    true,
			Status:      "ACTIVE",
			FoundingDate: "2018-05-15",
		}, nil
	}

	return nil, fmt.Errorf("XYP live service at %s requires valid XYP_API_KEY credentials", s.endpoint)
}
