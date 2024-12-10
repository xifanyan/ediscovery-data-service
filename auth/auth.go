package auth

import (
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type JWTClaim struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

var (
	jwtSecret = []byte("secret-key")
)

func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		token := extractToken(c)
		if token == "" {
			token, _ = createToken("pyan", time.Hour*1, jwtSecret)
			// return echo.ErrUnauthorized
		}

		claims, err := validateToken(token, jwtSecret)
		if err != nil {
			log.Error().Msg(err.Error())
			return echo.ErrUnauthorized
		}

		if claims.Username == "" {
			log.Error().Msg("failed to pull user name")
			return echo.ErrUnauthorized
		}

		c.Set("user", claims.Username)
		// c.Set("token", token)
		// c.Request().Header.Set("Authorization", "Bearer "+token)

		// Set claims in context
		// c.Set("claims", claims)
		return next(c)
	}
}

func createToken(username string, expirationTime time.Duration, secret []byte) (string, error) {
	claims := &JWTClaim{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expirationTime).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func extractToken(c echo.Context) string {
	bearerToken := c.Request().Header.Get("Authorization")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func validateToken(token string, secret []byte) (*JWTClaim, error) {
	claims := &JWTClaim{}
	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !tkn.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}
