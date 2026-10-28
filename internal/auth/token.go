package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultIssuer   = "opspilot"
	DefaultAudience = "opspilot-api"
)

// Claims represents standard and custom JWT claims.
type Claims struct {
	JTI       string `json:"jti"`
	Subject   string `json:"sub"`
	TenantID  string `json:"tenant_id"`
	Role      Role   `json:"role"`
	Email     string `json:"email"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// TokenService handles generation and validation of cryptographic JWT tokens.
type TokenService interface {
	GenerateToken(user User) (string, *Claims, error)
	ValidateToken(tokenStr string) (*Claims, error)
}

type jwtTokenService struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

// NewTokenService creates a new JWT token service with specified secret, issuer, audience, and TTL.
func NewTokenService(secret string, issuer, audience string, ttl time.Duration) (TokenService, error) {
	if len(secret) < 16 {
		return nil, errors.New("jwt secret must be at least 16 bytes")
	}
	if issuer == "" {
		issuer = DefaultIssuer
	}
	if audience == "" {
		audience = DefaultAudience
	}
	if ttl == 0 {
		ttl = 1 * time.Hour
	}

	return &jwtTokenService{
		secret:   []byte(secret),
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// GenerateToken generates a signed HS256 JWT access token for a user.
func (s *jwtTokenService) GenerateToken(user User) (string, *Claims, error) {
	now := time.Now().UTC()
	exp := now.Add(s.ttl)

	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random jti: %w", err)
	}
	jti := hex.EncodeToString(jtiBytes)

	claims := &Claims{
		JTI:       jti,
		Subject:   user.ID,
		TenantID:  user.TenantID,
		Role:      user.Role,
		Email:     user.Email,
		Issuer:    s.issuer,
		Audience:  s.audience,
		IssuedAt:  now.Unix(),
		ExpiresAt: exp.Unix(),
	}

	headerJSON, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", nil, err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", nil, err
	}

	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	unsignedToken := headerEncoded + "." + claimsEncoded
	signature := s.sign(unsignedToken)
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	tokenStr := unsignedToken + "." + signatureEncoded
	return tokenStr, claims, nil
}

// ValidateToken verifies token signature, structure, issuer, audience, and expiration.
func (s *jwtTokenService) ValidateToken(tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrUnauthorized
	}

	headerPart := parts[0]
	claimsPart := parts[1]
	signaturePart := parts[2]

	expectedSignature := s.sign(headerPart + "." + claimsPart)
	actualSignature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if !hmac.Equal(expectedSignature, actualSignature) {
		return nil, ErrUnauthorized
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(headerPart)
	if err != nil {
		return nil, ErrUnauthorized
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrUnauthorized
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return nil, ErrUnauthorized
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(claimsPart)
	if err != nil {
		return nil, ErrUnauthorized
	}
	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, ErrUnauthorized
	}

	now := time.Now().UTC().Unix()
	if claims.ExpiresAt <= now {
		return nil, ErrTokenExpired
	}

	if claims.Issuer != s.issuer {
		return nil, ErrUnauthorized
	}

	if claims.Audience != s.audience {
		return nil, ErrUnauthorized
	}

	return &claims, nil
}

func (s *jwtTokenService) sign(data string) []byte {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(data))
	return h.Sum(nil)
}
