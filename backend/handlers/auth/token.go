package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// [AI_DEPENDENCIES_START]
// DEPENDENCY_MAP:
// {
//   "external": ["github.com/golang-jwt/jwt/v5"],
//   "internal": [],
//   "usage": ["SignupHandler", "LoginHandler", "GetUserIDFromToken"]
// }
// [AI_DEPENDENCIES_END]

// [AI_SECURITY_START]
// SECURITY_CONSTRAINTS:
// {
//   "token_type": "JWT",
//   "algorithm": "HS256",
//   "expiration": "24h",
//   "claims": ["user_id", "exp"],
//   "secret_key": "environment_variable_required"
// }
// [AI_SECURITY_END]

// GenerateToken creates a JWT token for user authentication
// Used by: SignupHandler, LoginHandler
// Dependencies: jwt package
func GenerateToken(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	secretKey := os.Getenv("JWT_SECRET_KEY")
	fmt.Printf("Debug: JWT_SECRET_KEY length: %d\n", len(secretKey))
	if secretKey == "" {
		return "", fmt.Errorf("JWT_SECRET_KEY environment variable not set")
	}

	return token.SignedString([]byte(secretKey))
}

// GetUserIDFromToken extracts user ID from JWT token
// Used by: All authenticated endpoints
// Dependencies: jwt package
func GetUserIDFromToken(r *http.Request) (int, error) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		return 0, fmt.Errorf("no token provided")
	}

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		return 0, fmt.Errorf("JWT_SECRET_KEY environment variable not set")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token claims")
	}

	return int(claims["user_id"].(float64)), nil
}
