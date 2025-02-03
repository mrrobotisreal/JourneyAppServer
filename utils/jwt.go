package utils

import (
	"JourneyAppServer/types"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

const (
	jwtSecretKey = "my-temp-secret-key-here"

	apiKeyLength    = 32
	maxRequestRate  = 100
	keyRotationDays = 90
)

func GetJWTSecret() string {
	return jwtSecretKey
}

type RateLimiter struct {
	limiters sync.Map
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

func (rl *RateLimiter) GetLimiter(apiKey string) *rate.Limiter {
	limiter, exists := rl.limiters.Load(apiKey)
	if !exists {
		newLimiter := rate.NewLimiter(rate.Limit(maxRequestRate), maxRequestRate)
		rl.limiters.Store(apiKey, newLimiter)
		return newLimiter
	}
	return limiter.(*rate.Limiter)
}

func GenerateAndStoreJWT(username, sessionOption string) (string, error) {
	var expirationTime time.Time

	switch sessionOption {
	case "always":
		expirationTime = time.Now().Add(366 * 244 * time.Hour)
	case "daily":
		expirationTime = time.Now().Add(24 * time.Hour)
	case "weekly":
		expirationTime = time.Now().Add(7 * 24 * time.Hour)
	case "monthly":
		expirationTime = time.Now().Add(30 * 24 * time.Hour)
	case "never":
		expirationTime = time.Now().Add(1 * time.Minute)
	default:
		return "", errors.New("invalid session option")
	}

	claims := jwt.MapClaims{
		"username": username,
		"exp":      expirationTime.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", fmt.Errorf("error signing token: %v", err)
	}

	return tokenString, nil
}

func GenerateSecureAPIKey() (*types.APIKey, error) {
	randomBytes := make([]byte, apiKeyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("error generating random bytes: %v", err)
	}

	key := fmt.Sprintf("sk_%s", hex.EncodeToString(randomBytes))

	apiKey := &types.APIKey{
		Key:       key,
		Created:   time.Now(),
		LastUsed:  time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(keyRotationDays) * 24 * time.Hour),
	}

	return apiKey, nil
}

func ValidateAPIKey(apiKey *types.APIKey) error {
	if apiKey == nil {
		return errors.New("api key is nil")
	}

	if time.Now().After(apiKey.ExpiresAt) {
		return errors.New("api key has expired")
	}

	apiKey.LastUsed = time.Now()

	return nil
}

func IsKeyRotationNeeded(apiKey *types.APIKey) bool {
	return time.Now().After(apiKey.Created.Add(time.Duration(keyRotationDays) * 24 * time.Hour))
}

func IsValidSessionOption(option string) bool {
	validOptions := map[string]bool{
		"always":  true,
		"weekly":  true,
		"monthly": true,
		"never":   true,
	}
	return validOptions[option]
}
