// Package main is the AWS Lambda entry-point for the NBT Converter API.
// It accepts Bedrock (.mcworld, .mcstructure) and Java (.litematic, .schem, .nbt)
// files and returns a Java Edition structure NBT file.
//
// File layout:
//
//	main.go            – Lambda handler, routing, entry point
//	convert_java.go    – Java format conversion (litematic / schem / nbt)
//	convert_bedrock.go – Bedrock format conversion (mcworld / mcstructure)
//	response.go        – HTTP response helpers & query-string parsers
package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// handler is the Lambda request handler.
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received %s request for %s", req.HTTPMethod, req.Path)

	// ---- Handle OPTIONS (CORS Preflight) ------------------------------------
	if strings.EqualFold(req.HTTPMethod, "OPTIONS") {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "POST, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,X-Requested-With,Accept",
				"Access-Control-Max-Age":       "600",
			},
		}, nil
	}

	// ---- Setup Timeout ------------------------------------------------------
	// Limit total execution time to 25s (Lambda default is usually 30s)
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	// ---- Decode body --------------------------------------------------------
	var rawBody []byte
	var err error

	if req.IsBase64Encoded {
		rawBody, err = base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return errorResponse(http.StatusBadRequest, "Failed to decode base64 body"), nil
		}
	} else {
		rawBody = []byte(req.Body)
	}

	if len(rawBody) == 0 {
		return errorResponse(http.StatusBadRequest, "Empty request body for "+req.HTTPMethod+" request"), nil
	}

	// ---- Bind Parameters (oapi-codegen) -------------------------------------
	params, err := BindParams(req)
	if err != nil {
		return errorResponse(http.StatusBadRequest, "Invalid query parameters: "+err.Error()), nil
	}

	// ---- Determine format ---------------------------------------------------
	var ext string
	if params.Filename != nil {
		ext = strings.ToLower(filepath.Ext(*params.Filename))
	}

	// If extension is missing, or we want to be sure, use the sniffer.
	if ext == "" {
		detectedExt, err := DetectFormat(rawBody)
		if err == nil && detectedExt != "" {
			log.Printf("Auto-detected file format: %s", detectedExt)
			ext = detectedExt
		} else {
			// Fallback to legacy "type" param or default to .mcstructure
			if req.QueryStringParameters["type"] == "mcworld" {
				ext = ".mcworld"
			} else {
				ext = ".mcstructure"
			}
		}
	}

	// ---- Route to converter -------------------------------------------------
	var nbtBytes []byte

	switch ext {
	// ---- Bedrock: .mcworld --------------------------------------------------
	case ".mcworld":
		tmpFile := "/tmp/input.mcworld"
		if err = writeTmp(tmpFile, rawBody); err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to process uploaded file"), nil
		}
		defer os.Remove(tmpFile)

		nbtBytes, err = convertBedrockMcworld(ctx, tmpFile, params)
		if err != nil {
			return errorResponse(http.StatusUnprocessableEntity,
				"Failed to parse or convert Bedrock world: "+err.Error()), nil
		}

	// ---- Java: .litematic / .schem / .nbt -----------------------------------
	case ".litematic", ".schem", ".nbt":
		tmpFile := "/tmp/input" + ext
		if err = writeTmp(tmpFile, rawBody); err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to process uploaded file"), nil
		}
		defer os.Remove(tmpFile)

		nbtBytes, err = convertJavaNbt(ctx, tmpFile)
		if err != nil {
			return errorResponse(http.StatusUnprocessableEntity, err.Error()), nil
		}

	// ---- Default: try Bedrock .mcstructure, fallback to Java -----------------
	default:
		tmpFile := "/tmp/input.mcstructure"
		if err = writeTmp(tmpFile, rawBody); err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to process uploaded file"), nil
		}
		defer os.Remove(tmpFile)

		nbtBytes, err = convertBedrockMcstructure(ctx, tmpFile)
		if err != nil {
			log.Printf("Bedrock conversion error: %v — trying Java fallback", err)

			var errJava error
			nbtBytes, errJava = convertJavaNbt(ctx, tmpFile)
			if errJava != nil {
				return errorResponse(http.StatusUnprocessableEntity,
					"Bedrock: "+err.Error()+"\nJava fallback: "+errJava.Error()), nil
			}
		}
	}

	// ---- Return binary response ---------------------------------------------
	outputName := "converted.nbt"
	if params.Output != nil && *params.Output != "" {
		outputName = *params.Output
	}

	return events.APIGatewayProxyResponse{
		StatusCode:      http.StatusOK,
		IsBase64Encoded: true,
		Body:            base64.StdEncoding.EncodeToString(nbtBytes),
		Headers: map[string]string{
			"Content-Type":                "application/octet-stream",
			"Content-Disposition":         `attachment; filename="` + outputName + `"`,
			"Access-Control-Allow-Origin": "*",
			"Access-Control-Allow-Methods": "POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,X-Requested-With,Accept",
		},
	}, nil
}

// writeTmp writes data to a temporary file, logging on failure.
func writeTmp(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Failed to write tmp file %s: %v", path, err)
		return err
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
