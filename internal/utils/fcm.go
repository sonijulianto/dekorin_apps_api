package utils

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type firebaseServiceAccount struct {
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

var (
	fcmMu          sync.Mutex
	fcmCachedToken string
	fcmTokenExpiry time.Time
	fcmHTTPClient  = &http.Client{Timeout: 10 * time.Second}
)

func loadFirebaseServiceAccount() (*firebaseServiceAccount, *rsa.PrivateKey, error) {
	path := strings.TrimSpace(os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"))
	if path == "" {
		path = "./firebase-service-account.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var sa firebaseServiceAccount
	if err := json.Unmarshal(raw, &sa); err != nil {
		return nil, nil, fmt.Errorf("invalid service account json: %w", err)
	}
	if sa.ProjectID == "" || sa.PrivateKey == "" || sa.ClientEmail == "" {
		return nil, nil, fmt.Errorf("incomplete firebase service account fields")
	}
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode private_key PEM")
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private_key: %w", err)
	}
	rsaKey, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("private_key is not RSA")
	}
	return &sa, rsaKey, nil
}

func getFCMAccessToken(sa *firebaseServiceAccount, rsaKey *rsa.PrivateKey) (string, error) {
	fcmMu.Lock()
	defer fcmMu.Unlock()

	if fcmCachedToken != "" && time.Now().Add(2*time.Minute).Before(fcmTokenExpiry) {
		return fcmCachedToken, nil
	}

	now := time.Now()
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	claims := jwt.MapClaims{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
		"aud":   tokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(55 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	assertion, err := tok.SignedString(rsaKey)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	resp, err := fcmHTTPClient.PostForm(tokenURI, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("oauth2 token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.AccessToken == "" {
		return "", fmt.Errorf("failed to parse oauth2 access_token")
	}

	expSec := parsed.ExpiresIn
	if expSec <= 0 {
		expSec = 3600
	}
	fcmCachedToken = parsed.AccessToken
	fcmTokenExpiry = now.Add(time.Duration(expSec) * time.Second)
	return fcmCachedToken, nil
}

// SendPushToTokens mengirim push notification FCM HTTP v1 ke daftar token perangkat.
// Mengembalikan daftar token yang tidak valid/UNREGISTERED agar dapat dibersihkan dari database.
func SendPushToTokens(tokens []string, title, body string, data map[string]string) []string {
	if len(tokens) == 0 {
		return nil
	}
	sa, rsaKey, err := loadFirebaseServiceAccount()
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("ℹ️ FCM dilewati: file service account belum tersedia (%v)", err)
		} else {
			log.Printf("⚠️ FCM gagal memuat service account: %v", err)
		}
		return nil
	}

	accessToken, err := getFCMAccessToken(sa, rsaKey)
	if err != nil {
		log.Printf("⚠️ FCM gagal mendapatkan access token: %v", err)
		return nil
	}

	endpoint := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", sa.ProjectID)
	var invalidTokens []string

	for _, deviceToken := range tokens {
		trimmed := strings.TrimSpace(deviceToken)
		if trimmed == "" {
			continue
		}
		payload := map[string]interface{}{
			"message": map[string]interface{}{
				"token": trimmed,
				"notification": map[string]string{
					"title": title,
					"body":  body,
				},
				"data": data,
				"android": map[string]interface{}{
					"priority": "high",
					"notification": map[string]string{
						"channel_id": "dekorin_collaboration",
						"sound":      "default",
					},
				},
			},
		}
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			continue
		}

		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := fcmHTTPClient.Do(req)
		if err != nil {
			log.Printf("⚠️ FCM send error: %v", err)
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound || (resp.StatusCode == http.StatusBadRequest && strings.Contains(string(respBody), "UNREGISTERED")) {
			invalidTokens = append(invalidTokens, trimmed)
		} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("⚠️ FCM response (%d): %s", resp.StatusCode, string(respBody))
		}
	}

	return invalidTokens
}
