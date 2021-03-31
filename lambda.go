package main

import (
	"fmt"

	"encoding/base64"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
)

func erro(e error) string {
	return fmt.Sprintf(
		`{"error": "%s"}
`, e.Error())
}

func succ(s string) string {
	return fmt.Sprintf(
		`{"message": "%s"}
`, s)
}

type S struct {
	Content string `json:"content"`
}

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
	text, err := extract(b)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "unable to extract: %s"}`+"\n", err.Error()),
		}, nil
	}
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       fmt.Sprintf(`{"text": "%s"}`+"\n", text),
	}, nil
}

func extract(b []byte) (string, error) {
	mySession, err := session.NewSession()
	if err != nil {
		return "", err
	}
	svc := textract.New(mySession)
	input := &textract.DetectDocumentTextInput{
		Document: &textract.Document{Bytes: b},
	}
	output, err := svc.DetectDocumentText(input)
	if err != nil {
		return "", err
	}
	return output.String(), nil
}

func main() {
	lambda.Start(HandleRequest)
}
