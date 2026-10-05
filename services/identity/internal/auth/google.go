package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
	googleUserURL  = "https://www.googleapis.com/oauth2/v3/userinfo"
)

type GoogleOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTPClient   *http.Client
}

type GoogleUserInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
}

func NewGoogleOAuth(clientID, clientSecret, redirectURI string) *GoogleOAuth {
	return &GoogleOAuth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		HTTPClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *GoogleOAuth) Configured() bool {
	return g != nil && g.ClientID != "" && g.ClientSecret != ""
}

func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func CodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (g *GoogleOAuth) BuildAuthorizationURL(state, challenge string) string {
	q := url.Values{}
	q.Set("client_id", g.ClientID)
	q.Set("redirect_uri", g.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("access_type", "online")
	q.Set("prompt", "select_account")
	return googleAuthURL + "?" + q.Encode()
}

func (g *GoogleOAuth) ExchangeCode(code, verifier string) (string, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", g.ClientID)
	form.Set("client_secret", g.ClientSecret)
	form.Set("redirect_uri", g.RedirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", verifier)

	req, err := http.NewRequest(http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("oauth_exchange_failed:decode")
	}
	access, _ := decoded["access_token"].(string)
	if access == "" {
		errCode, _ := decoded["error"].(string)
		if errCode == "" {
			errCode = "unknown"
		}
		return "", fmt.Errorf("oauth_exchange_failed:%s", errCode)
	}
	return access, nil
}

func (g *GoogleOAuth) FetchUserInfo(accessToken string) (GoogleUserInfo, error) {
	var info GoogleUserInfo
	req, err := http.NewRequest(http.MethodGet, googleUserURL, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := g.HTTPClient.Do(req)
	if err != nil {
		return info, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &info); err != nil {
		return info, fmt.Errorf("oauth_userinfo_failed")
	}
	if info.Email == "" {
		return info, fmt.Errorf("oauth_userinfo_failed")
	}
	return info, nil
}
