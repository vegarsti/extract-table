package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vegarsti/extract/csv"
	"github.com/vegarsti/extract/dynamodb"
	"github.com/vegarsti/extract/html"
	"github.com/vegarsti/extract/s3"
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

	imageBytes, err := getImageBytes(decodedBodyBytes, req.Headers["content-type"])
	if err != nil {
		return errorResponse(err), nil
	}

	// get table, from cache if possible
	checksum := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	table, err := getTable(imageBytes, checksum)
	if err != nil {
		return errorResponse(err), nil
	}
	tableBytes, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to json: %w", err)
	}

	// Format media type responses
	jsonBody := string(tableBytes) + "\n"
	csvBody := csv.FromTable(table)

	// Determine what media type to return by looking at the Accept HTTP header
	// The header is on the form accept: text/html, application/xhtml+xml, application/xml;q=0.9
	// where the content types are listed in preferred order.
	acceptResponseTypes := strings.Split(req.Headers["accept"], ",")
	for _, e := range acceptResponseTypes {
		mediaType, _, err := mime.ParseMediaType(e)
		if err != nil {
			return errorResponse(fmt.Errorf("unable to parse media type '%s' in Accept header: %w", e, err)), nil
		}
		if mediaType == "text/html" {
			return &events.APIGatewayProxyResponse{
				Headers: map[string]string{
					"Location": "https://results.extract-table.com/" + checksum,
				},
				StatusCode: 301,
			}, nil
		}
		if mediaType == "application/json" {
			return successResponse(jsonBody, mediaType), nil
		}
		if mediaType == "text/csv" {
			return successResponse(csvBody, mediaType), nil
		}
	}
	return successResponse(jsonBody, "application/json"), nil
}

func errorResponse(err error) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 400,
		Body:       fmt.Sprintf(`{"error": "%s"}`+"\n", err.Error()),
	}
}

func successResponse(body string, mediaType string) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": mediaType},
		StatusCode: 200,
		Body:       body,
	}
}

func main() {
	lambda.Start(HandleRequest)
}

func getTable(imageBytes []byte, checksum string) ([][]string, error) {
	tableBytes, err := dynamodb.GetTable(checksum)
	if err != nil {
		return nil, fmt.Errorf("dynamodb.GetTable: %w", err)
	}
	var table [][]string
	if tableBytes == nil {
		output, err := textract.Extract(imageBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to extract: %w", err)
		}
		table, err := textract.ToTableFromDetectedTable(output)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to table: %w", err)
		}

		csvBytes := []byte(csv.FromTable(table))
		url := "https://results.extract-table.com/" + checksum
		imageURL := url + ".png" // what about jpg?
		csvURL := url + ".csv"
		htmlBytes := html.FromTable(table, imageURL, csvURL)

		if err := s3.UploadPNG(checksum, imageBytes); err != nil {
			return nil, err
		}
		if err := s3.UploadCSV(checksum, csvBytes); err != nil {
			return nil, err
		}
		if err := s3.UploadHTML(checksum, htmlBytes); err != nil {
			return nil, err
		}
		if err := dynamodb.PutTable(checksum, tableBytes); err != nil {
			return nil, fmt.Errorf("dynamodb.PutTable: %w", err)
		}
	}
	if err := json.Unmarshal(tableBytes, &table); err != nil {
		return nil, fmt.Errorf("failed to convert from json: %w", err)
	}
	return table, nil
}

func getImageBytes(decodedBodyBytes []byte, contentTypeHeader string) ([]byte, error) {
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse media type': %w", err)
	}

	if mediaType != "multipart/form-data" {
		return decodedBodyBytes, nil
	}
	decodedBody := string(decodedBodyBytes)
	reader := multipart.NewReader(strings.NewReader(decodedBody), params["boundary"])
	tenMBInBytes := 10000000
	form, err := reader.ReadForm(int64(tenMBInBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read form': %w", err)
	}
	file, ok := form.File["image"]
	if !ok {
		return nil, fmt.Errorf("no file in form field 'image'")
	}
	f, err := file[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file': %w", err)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read file': %w", err)
	}
	return data, nil
}
