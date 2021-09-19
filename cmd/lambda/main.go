package main

import (
	"crypto/md5"
	"fmt"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/vegarsti/extract/dynamodb"
	"github.com/vegarsti/extract/textract"
)

func HandleRequest(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	if !req.IsBase64Encoded {
		return errorResponse(fmt.Errorf("request body must have a content-type that is either image/png or image/jpeg")), nil
	}
	imageBytes, err := base64.StdEncoding.DecodeString(req.Body)
	if err != nil {
		return errorResponse(fmt.Errorf("unable to convert base64 to bytes: %w", err)), nil
	}
	sess, err := session.NewSession()
	if err != nil {
		return errorResponse(fmt.Errorf("unable to create session: %w", err)), nil
	}

	// Check if table is stored
	checksum := md5.Sum(imageBytes)
	storedBytes, err := dynamodb.GetTable(sess, checksum[:])
	if err != nil {
		return errorResponse(fmt.Errorf("dynamodb.GetTable: %w", err)), nil
	}
	if storedBytes != nil {
		return successResponse(storedBytes), nil
	}

	output, err := textract.Extract(sess, imageBytes)
	if err != nil {
		return errorResponse(fmt.Errorf("failed to extract: %w", err)), nil
	}
	table, err := textract.ToTableFromDetectedTable(output)
	if err != nil {
		return errorResponse(fmt.Errorf("failed to convert to table: %w", err)), nil
	}

	tableBytes, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return errorResponse(fmt.Errorf("failed to convert to json: %w", err)), nil
	}
	if err := dynamodb.PutTable(sess, checksum[:], tableBytes); err != nil {
		return errorResponse(fmt.Errorf("dynamodb.PutTable: %w", err)), nil
	}
	return successResponse(tableBytes), nil
}

func errorResponse(err error) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 400,
		Body:       fmt.Sprintf(`{"error": "%s"}`+"\n", err.Error()),
	}
}

func successResponse(tableBytes []byte) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       string(tableBytes) + "\n",
	}
}

func main() {
	lambda.Start(HandleRequest)
}
