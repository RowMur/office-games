package token

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "office-games-access"

type TokenKind struct {
	Duration time.Duration
}

var AuthenticationToken = TokenKind{Duration: time.Hour * 1440}
var ForgotPasswordToken = TokenKind{Duration: time.Hour * 3}

func GenerateToken(userId uint, tokenKind TokenKind) (string, error) {
	timeNow := time.Now().UTC()
	expiryTime := timeNow.Add(tokenKind.Duration)

	claims := &jwt.RegisteredClaims{
		Issuer: issuer,
		IssuedAt: &jwt.NumericDate{
			Time: timeNow,
		},
		ExpiresAt: &jwt.NumericDate{
			Time: expiryTime,
		},
		Subject: fmt.Sprint(userId),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}
	return token.SignedString([]byte(secret))
}

type Token struct {
	String     string
	UserId     uint
	HasExpired bool
}

func ParseToken(tokenString string) (*Token, error) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return &Token{HasExpired: true, String: tokenString}, nil
		}

		return nil, errors.New("invalid token, could not parse")
	}

	tokenIssuer, err := token.Claims.GetIssuer()
	if err != nil {
		return nil, errors.New("invalid token, could not parse")
	}
	if tokenIssuer != issuer {
		return nil, errors.New("invalid token, invalid issuer")
	}

	userId, err := token.Claims.GetSubject()
	if err != nil {
		return nil, errors.New("error parsing token")
	}

	intUserId, err := strconv.Atoi(userId)
	if err != nil {
		return nil, errors.New("error parsing token")
	}

	return &Token{UserId: uint(intUserId), HasExpired: false, String: tokenString}, nil
}
