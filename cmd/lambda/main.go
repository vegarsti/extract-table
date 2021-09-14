package main

import (
	"fmt"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/vegarsti/extract/textract"
)

func HandleRequest(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	if !req.IsBase64Encoded {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       `{"error": "request body must have content-type image/png"}` + "\n",
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
	bodyBytes, err := json.Marshal(table)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to convert to json: %s"}`+"\n", err.Error()),
		}, nil
	}
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       string(bodyBytes),
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
