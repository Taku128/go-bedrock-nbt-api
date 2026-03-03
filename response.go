package main

import (
	"strconv"

	"github.com/aws/aws-lambda-go/events"
)

// parseInt32Param extracts an int32 query-string parameter with a default fallback.
func parseInt32Param(params map[string]string, key string, def int32) int32 {
	if valStr, ok := params[key]; ok {
		if val, err := strconv.ParseInt(valStr, 10, 32); err == nil {
			return int32(val)
		}
	}
	return def
}

// parseStringParam extracts a string query-string parameter with a default fallback.
func parseStringParam(params map[string]string, key string, def string) string {
	if valStr, ok := params[key]; ok && valStr != "" {
		return valStr
	}
	return def
}

// errorResponse builds a plain-text error response with CORS headers.
func errorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       message,
		Headers: map[string]string{
			"Content-Type":                "text/plain",
			"Access-Control-Allow-Origin": "*",
		},
	}
}
