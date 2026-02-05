package utils

import (
	"github.com/aws/aws-lambda-go/events"
)

func SuccessResponse(code int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: code, Body: body, Headers: map[string]string{"Content-Type": "application/json"}}
}

func ErrorResponse(code int, msg string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: code, Body: `{"error":"` + msg + `"}`}
}
