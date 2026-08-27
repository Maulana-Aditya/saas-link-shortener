package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	jwt.RegisteredClaims
}

type Issuer struct {
	AccessSecret  []byte
	RefreshSecret []byte
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func NewIssuer(accessSecret, refreshSecret string, accessTTLMinutes, refreshTTLDays int) *Issuer {
	return &Issuer{
		AccessSecret:  []byte(accessSecret),
		RefreshSecret: []byte(refreshSecret),
		AccessTTL:     time.Duration(accessTTLMinutes) * time.Minute,
		RefreshTTL:    time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

func (i *Issuer) IssueAccessToken(userID, orgID string) (string, error) {
	return i.sign(userID, orgID, i.AccessSecret, i.AccessTTL)
}

func (i *Issuer) IssueRefreshToken(userID, orgID string) (string, error) {
	return i.sign(userID, orgID, i.RefreshSecret, i.RefreshTTL)
}

func (i *Issuer) sign(userID, orgID string, secret []byte, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (i *Issuer) ParseAccessToken(tokenStr string) (*Claims, error) {
	return parse(tokenStr, i.AccessSecret)
}

func (i *Issuer) ParseRefreshToken(tokenStr string) (*Claims, error) {
	return parse(tokenStr, i.RefreshSecret)
}

func parse(tokenStr string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
