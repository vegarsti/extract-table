package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/vegarsti/extract/dynamodb"
	"github.com/vegarsti/extract/html"
	"github.com/vegarsti/extract/textract"
)

func HandleRequest(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	if !req.IsBase64Encoded {
		return errorResponse(fmt.Errorf("request body must have a content-type that is either image/png, image/jpeg, or multipart/form-data")), nil
	}

	decodedBodyBytes, err := base64.StdEncoding.DecodeString(req.Body)
	if err != nil {
		return errorResponse(fmt.Errorf("unable to convert base64 to bytes: %w", err)), nil
	}

	mediaType, params, err := mime.ParseMediaType(req.Headers["content-type"])
	if err != nil {
		return errorResponse(fmt.Errorf("failed to parse media type': %w", err)), nil
	}
	if mediaType == "multipart/form-data" {
		decodedBody := string(decodedBodyBytes)
		reader := multipart.NewReader(strings.NewReader(decodedBody), params["boundary"])
		tenMBInBytes := 10000000
		form, err := reader.ReadForm(int64(tenMBInBytes))
		if err != nil {
			return errorResponse(fmt.Errorf("failed to read form': %w", err)), nil
		}
		file, ok := form.File["image"]
		if !ok {
			return errorResponse(fmt.Errorf("no file in form field 'image'")), nil
		}
		f, err := file[0].Open()
		if err != nil {
			return errorResponse(fmt.Errorf("failed to open file': %w", err)), nil
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return errorResponse(fmt.Errorf("failed to read file': %w", err)), nil
		}
		decodedBodyBytes = data
	}

	// Determine whether to return HTML or JSON by looking at the Accept HTTP header
	// The header is on the form `accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8`,
	// where the content types are listed in preferred order.
	var returnHTML bool
	acceptResponseTypesRaw := req.Headers["accept"]
	acceptResponseTypes := strings.Split(acceptResponseTypesRaw, ",")
	acceptResponseTypesLowestToHighestPriority := make([]string, len(acceptResponseTypes))
	for i, e := range acceptResponseTypes {
		mediaType, _, err := mime.ParseMediaType(e)
		if err != nil {
			return errorResponse(fmt.Errorf("unable to parse media type: %w", err)), nil
		}
		acceptResponseTypesLowestToHighestPriority[len(acceptResponseTypes)-1-i] = mediaType
	}
	for _, value := range acceptResponseTypesLowestToHighestPriority {
		if value == "text/html" {
			returnHTML = true
		}
		if value == "application/json" {
			returnHTML = false
		}
	}

	sess, err := session.NewSession()
	if err != nil {
		return errorResponse(fmt.Errorf("unable to create session: %w", err)), nil
	}

	// Check if table is stored
	checksum := md5.Sum(decodedBodyBytes)
	storedBytes, err := dynamodb.GetTable(sess, checksum[:])
	if err != nil {
		return errorResponse(fmt.Errorf("dynamodb.GetTable: %w", err)), nil
	}
	if storedBytes != nil {
		if !returnHTML {
			return JSONSuccessResponse(storedBytes), nil
		}
		var table [][]string
		if err := json.Unmarshal(storedBytes, &table); err != nil {
			return errorResponse(fmt.Errorf("failed to convert from json: %w", err)), nil
		}
		tableHTML := html.FromTable(table)
		return HTMLSuccessResponse(tableHTML), nil
	}

	output, err := textract.Extract(sess, decodedBodyBytes)
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
	if !returnHTML {
		return JSONSuccessResponse(tableBytes), nil
	}
	tableHTML := html.FromTable(table)
	return HTMLSuccessResponse(tableHTML), nil
}

func errorResponse(err error) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 400,
		Body:       fmt.Sprintf(`{"error": "%s"}`+"\n", err.Error()),
	}
}

func HTMLSuccessResponse(tableHTML string) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: 200,
		Body:       tableHTML + "\n",
	}
}

func JSONSuccessResponse(tableBytes []byte) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       string(tableBytes) + "\n",
	}
}

func main() {
	lambda.Start(HandleRequest)
}
