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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// handler is the Lambda request handler.
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		return errorResponse(http.StatusBadRequest, "Empty request body"), nil
	}

	// ---- Determine format ---------------------------------------------------
	// "filename" query param is preferred; fall back to legacy "type" param.
	inputFilename := parseStringParam(req.QueryStringParameters, "filename", "")
	ext := strings.ToLower(filepath.Ext(inputFilename))
	if ext == "" {
		if req.QueryStringParameters["type"] == "mcworld" {
			ext = ".mcworld"
		} else {
			ext = ".mcstructure"
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

		nbtBytes, err = convertBedrockMcworld(tmpFile, req.QueryStringParameters)
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

		nbtBytes, err = convertJavaNbt(tmpFile)
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

		nbtBytes, err = convertBedrockMcstructure(tmpFile)
		if err != nil {
			log.Printf("Bedrock conversion error: %v — trying Java fallback", err)

			var errJava error
			nbtBytes, errJava = convertJavaNbt(tmpFile)
			if errJava != nil {
				return errorResponse(http.StatusUnprocessableEntity,
					"Bedrock: "+err.Error()+"\nJava fallback: "+errJava.Error()), nil
			}
		}
	}

	// ---- Return binary response ---------------------------------------------
	outputName := parseStringParam(req.QueryStringParameters, "output", "converted.nbt")
	return events.APIGatewayProxyResponse{
		StatusCode:      http.StatusOK,
		IsBase64Encoded: true,
		Body:            base64.StdEncoding.EncodeToString(nbtBytes),
		Headers: map[string]string{
			"Content-Type":                "application/octet-stream",
			"Content-Disposition":         `attachment; filename="` + outputName + `"`,
			"Access-Control-Allow-Origin": "*",
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
