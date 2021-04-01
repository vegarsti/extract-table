package main

import (
	"fmt"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
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
	output, err := extract(b)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to extract: %s"}`+"\n", err.Error()),
		}, nil
	}
	table, err := toTable(output)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to convert to table: %s"}`+"\n", err.Error()),
		}, nil
	}
	bs, err := json.Marshal(table)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "failed to convert to json: %s"}`+"\n", err.Error()),
		}, nil
	}
	rows := string(bs)
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       fmt.Sprintf(`{"rows": "%s"}`+"\n", rows),
	}, nil
}

func extract(bs []byte) (*textract.DetectDocumentTextOutput, error) {
	mySession, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	svc := textract.New(mySession)
	input := &textract.DetectDocumentTextInput{
		Document: &textract.Document{Bytes: bs},
	}
	output, err := svc.DetectDocumentText(input)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func main() {
	lambda.Start(HandleRequest)
}

/*

	// write table to stdout
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, row := range table {
		for j, cell := range row {
			fmt.Fprintf(w, cell)
			if j < len(row)-1 {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintf(w, "\n")
	}
	w.Flush()
*/
