package eid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// AuthMethod represents official E-ID Mongolia authentication channels
type AuthMethod string

const (
	AuthMethodPKISignature AuthMethod = "PKI_DIGITAL_SIGNATURE" // Тоон гарын үсэг
	AuthMethodMobileOTP    AuthMethod = "MOBILE_OTP"            // Нэг удаагийн нууц код
	AuthMethodBankSSO      AuthMethod = "BANK_SSO"              // Банкны системээр
	AuthMethodBiometric    AuthMethod = "BIOMETRIC_FACE"        // Царай танилт
)

// EIDIdentity matches official DAN / E-ID Mongolia user profile schema
type EIDIdentity struct {
	CivilID        string     `json:"civil_id"`        // Иргэний бүртгэлийн дугаар
	RegNumber      string     `json:"reg_number"`      // Регистрийн дугаар (e.g. AA90010111)
	FirstName      string     `json:"first_name"`      // Өөрийн нэр
	LastName       string     `json:"last_name"`       // Эцэг/Эхийн нэр
	FamilyName     string     `json:"family_name"`     // Ургийн овог
	Gender         string     `json:"gender"`          // Хүйс
	Email          string     `json:"email"`           // И-мэйл хаяг
	Phone          string     `json:"phone"`           // Утасны дугаар
	AuthMethod     AuthMethod `json:"auth_method"`     // Танилт хийсэн арга
	SignatureHash  string     `json:"signature_hash"`  // Тоон гарын үсгийн хеш
	VerifiedStatus bool       `json:"verified_status"` // Төрийн сангийн баталгаажуулалт
	AuthenticatedAt time.Time  `json:"authenticated_at"`
}

type Provider interface {
	GetAuthorizeURL(redirectURI, state string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (*EIDIdentity, error)
	AuthenticateWithMethod(ctx context.Context, regNumber, otpCode string, method AuthMethod) (*EIDIdentity, error)
}

type EIDService struct {
	clientID     string
	clientSecret string
	authorizeURL string
	tokenURL     string
	userInfoURL  string
	mockMode     bool
	httpClient   *http.Client
}

func NewEIDService() *EIDService {
	mock := os.Getenv("EID_MOCK_MODE") != "false"
	clientID := os.Getenv("EID_CLIENT_ID")
	if clientID == "" {
		clientID = "gerege-open-erp-client"
	}
	clientSecret := os.Getenv("EID_CLIENT_SECRET")

	authURL := os.Getenv("EID_AUTHORIZE_URL")
	if authURL == "" {
		authURL = "https://sso.gov.mn/oauth2/authorize"
	}
	tokenURL := os.Getenv("EID_TOKEN_URL")
	if tokenURL == "" {
		tokenURL = "https://sso.gov.mn/oauth2/token"
	}
	userURL := os.Getenv("EID_USERINFO_URL")
	if userURL == "" {
		userURL = "https://sso.gov.mn/oauth2/userinfo"
	}

	return &EIDService{
		clientID:     clientID,
		clientSecret: clientSecret,
		authorizeURL: authURL,
		tokenURL:     tokenURL,
		userInfoURL:  userURL,
		mockMode:     mock,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

// GetAuthorizeURL constructs official OAuth2 authorization link for eidmongolia.mn / sso.gov.mn
func (s *EIDService) GetAuthorizeURL(redirectURI, state string) string {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", s.clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("scope", "openid profile regnum civil_id phone email")
	v.Set("state", state)
	return fmt.Sprintf("%s?%s", s.authorizeURL, v.Encode())
}

// ExchangeCode exchanges OAuth2 authorization code for E-ID Identity profile
func (s *EIDService) ExchangeCode(ctx context.Context, code, redirectURI string) (*EIDIdentity, error) {
	if code == "" {
		return nil, errors.New("empty authorization code")
	}

	if s.mockMode {
		return &EIDIdentity{
			CivilID:        "CID-99887766",
			RegNumber:      "AA90010111",
			FirstName:      "Болд",
			LastName:       "Бат",
			FamilyName:     "Боржигон",
			Gender:         "MALE",
			Email:          "bat.bold@eidmongolia.mn",
			Phone:          "99112233",
			AuthMethod:     AuthMethodPKISignature,
			SignatureHash:  "sha256_mock_pki_signature_eidmongolia",
			VerifiedStatus: true,
			AuthenticatedAt: time.Now(),
		}, nil
	}

	// Live OAuth2 Token Request to SSO / E-ID Mongolia
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", s.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("E-ID Mongolia token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("E-ID token error (%d): %s", resp.StatusCode, string(body))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}

	return s.fetchUserInfo(ctx, tokenRes.AccessToken)
}

func (s *EIDService) AuthenticateWithMethod(ctx context.Context, regNumber, otpCode string, method AuthMethod) (*EIDIdentity, error) {
	if len(regNumber) < 8 {
		return nil, errors.New("invalid registration number: minimum 8 characters required")
	}

	if method == "" {
		method = AuthMethodMobileOTP
	}

	if s.mockMode {
		return &EIDIdentity{
			CivilID:        "CID-" + regNumber,
			RegNumber:      strings.ToUpper(regNumber),
			FirstName:      "Баталгаажсан",
			LastName:       "Иргэн",
			FamilyName:     "Монгол",
			Gender:         "MALE",
			Email:          strings.ToLower(regNumber) + "@eidmongolia.mn",
			Phone:          "99001122",
			AuthMethod:     method,
			SignatureHash:  "pki_signed_token_" + regNumber,
			VerifiedStatus: true,
			AuthenticatedAt: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("E-ID Mongolia OTP verification requires production SSO credentials")
}

func (s *EIDService) fetchUserInfo(ctx context.Context, accessToken string) (*EIDIdentity, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Sub       string `json:"sub"`
		RegNum    string `json:"regnum"`
		CivilID   string `json:"civil_id"`
		GivenName string `json:"given_name"`
		Family    string `json:"family_name"`
		Email     string `json:"email"`
		Phone     string `json:"phone_number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return &EIDIdentity{
		CivilID:        raw.CivilID,
		RegNumber:      raw.RegNum,
		FirstName:      raw.GivenName,
		LastName:       raw.Family,
		Email:          raw.Email,
		Phone:          raw.Phone,
		AuthMethod:     AuthMethodPKISignature,
		VerifiedStatus: true,
		AuthenticatedAt: time.Now(),
	}, nil
}
