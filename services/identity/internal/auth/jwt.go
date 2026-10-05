package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	privateKey interface{}
	publicKey  interface{}
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	kid        string
	n          string
	e          string
}

func NewJWTService(privPath, pubPath, issuer, audience string, accessTTL, refreshTTL time.Duration) (*JWTService, error) {
	privPEM, err := os.ReadFile(privPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	pubPEM, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	eb := make([]byte, 0, 8)
	e := priv.PublicKey.E
	for e > 0 {
		eb = append([]byte{byte(e & 0xff)}, eb...)
		e >>= 8
	}
	if len(eb) == 0 {
		eb = []byte{0}
	}
	return &JWTService{
		privateKey: priv,
		publicKey:  pub,
		issuer:     issuer,
		audience:   audience,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		kid:        "facinect-1",
		n:          base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes()),
		e:          base64.RawURLEncoding.EncodeToString(eb),
	}, nil
}

func (j *JWTService) AccessTTL() time.Duration  { return j.accessTTL }
func (j *JWTService) RefreshTTL() time.Duration { return j.refreshTTL }

func (j *JWTService) IssueAccess(c AccessClaims) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":  j.issuer,
		"aud":  j.audience,
		"iat":  now.Unix(),
		"nbf":  now.Unix(),
		"exp":  now.Add(j.accessTTL).Unix(),
		"sub":  fmt.Sprintf("%d", c.UserID),
		"typ":  "access",
		"name": c.Name,
	}
	if c.Email != "" {
		claims["email"] = c.Email
	}
	if c.Phone != "" {
		claims["phone"] = c.Phone
	}
	if c.Role != "" {
		claims["role"] = c.Role
	}
	if len(c.PageAccess) > 0 {
		claims["page_access"] = c.PageAccess
	}
	if len(c.FacilityIDs) > 0 {
		claims["facilities"] = c.FacilityIDs
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.kid
	return token.SignedString(j.privateKey)
}

type AccessClaims struct {
	UserID      int64
	Name        string
	Email       string
	Phone       string
	Role        string
	PageAccess  []string
	FacilityIDs []int64
}

func (j *JWTService) ParseAccess(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return j.publicKey, nil
	}, jwt.WithAudience(j.audience), jwt.WithIssuer(j.issuer))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid_token")
	}
	if typ, _ := claims["typ"].(string); typ != "access" {
		return nil, fmt.Errorf("not_access_token")
	}
	return claims, nil
}

func (j *JWTService) JWKS() map[string]interface{} {
	return map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": j.kid,
			"use": "sig",
			"alg": "RS256",
			"n":   j.n,
			"e":   j.e,
		}},
	}
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}
