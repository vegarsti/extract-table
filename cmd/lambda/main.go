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
		return errorResponse("request body must have a content-type that is either image/png or image/jpeg"), nil
	}
	imageBytes, err := base64.StdEncoding.DecodeString(req.Body)
	if err != nil {
		return errorResponse(fmt.Sprintf("unable to convert base64 to bytes: %s", err.Error())), nil
	}
	mySession, err := session.NewSession()
	if err != nil {
		return errorResponse(fmt.Sprintf("unable to create session: %s", err.Error())), nil
	}

	// Check if table is stored
	checksum := md5.Sum(imageBytes)
	storedBytes, err := dynamodb.GetTable(mySession, checksum[:])
	if err != nil {
		return errorResponse(fmt.Sprintf("dynamodb.GetTable: %s", err.Error())), nil
	}
	if storedBytes != nil {
		return successResponse(storedBytes), nil
	}

	output, err := textract.Extract(mySession, imageBytes)
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to extract: %s", err.Error())), nil
	}
	table, err := textract.ToTableFromDetectedTable(output)
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to convert to table: %s", err.Error())), nil
	}

	tableBytes, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to convert to json: %s", err.Error())), nil
	}
	if err := dynamodb.PutTable(mySession, checksum[:], tableBytes); err != nil {
		return errorResponse(fmt.Sprintf("dynamodb.PutTable: %s", err.Error())), nil
	}
	return successResponse(tableBytes), nil
}

func errorResponse(message string) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 400,
		Body:       fmt.Sprintf(`{"error": "%s"}`+"\n", message),
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
