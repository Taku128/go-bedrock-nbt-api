package main

import (
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/ntaku256/go-bedrock-nbt-api/api"
	"github.com/oapi-codegen/runtime"
)

// BindParams populates the ConvertFileParams struct from APIGatewayProxyRequest.
func BindParams(req events.APIGatewayProxyRequest) (*api.ConvertFileParams, error) {
	values := make(url.Values)
	for k, v := range req.QueryStringParameters {
		values.Add(k, v)
	}

	var params api.ConvertFileParams
	err := runtime.BindForm(&params, values, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind query parameters: %w", err)
	}

	return &params, nil
}

// errorResponse builds a plain-text error response with CORS headers.
func errorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       message,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,X-Requested-With,Accept",
			"Content-Type":                "text/plain",
		},
	}
}
