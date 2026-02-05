package auth

import "github.com/aws/aws-lambda-go/events"

func CheckAuth(v Validator, request events.APIGatewayV2HTTPRequest) (string, error) {
	authHeader := request.Headers["Authorization"]
	if authHeader == "" {
		authHeader = request.Headers["authorization"]
	}
	tokenString, err := ExtractBearerToken(authHeader)
	if err != nil {
		return "", err
	}
	claims, err := v.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims["sub"].(string), nil
}
