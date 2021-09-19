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
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       `{"error": "request body must have a content-type that is either image/png or image/jpeg"}` + "\n",
		}, nil
	}
	b, err := base64.StdEncoding.DecodeString(req.Body)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "unable to convert base64 to bytes: %s"}`+"\n", err.Error()),
		}, nil
	}
	mySession, err := session.NewSession()
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "unable to create session: %s"}`+"\n", err.Error()),
		}, nil
	}

	// Check if table is stored
	checksum := md5.Sum(b)
	storedBytes, err := dynamodb.GetTable(mySession, checksum[:])
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "dynamodb.GetTable: %s"}`+"\n", err.Error()),
		}, nil
	}
	if storedBytes != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 200,
			Body:       string(storedBytes) + "\n",
		}, nil
	}

	output, err := textract.Extract(mySession, b)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to extract: %s"}`+"\n", err.Error()),
		}, nil
	}
	table, err := textract.ToTableFromDetectedTable(output)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to convert to table: %s"}`+"\n", err.Error()),
		}, nil
	}

	tableBytes, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to convert to json: %s"}`+"\n", err.Error()),
		}, nil
	}
	if err := dynamodb.PutTable(mySession, checksum[:], tableBytes); err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "dynamodb.PutTable: %s"}`+"\n", err.Error()),
		}, nil
	}
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       string(tableBytes) + "\n",
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
