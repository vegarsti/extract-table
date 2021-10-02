package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vegarsti/extract/csv"
	"github.com/vegarsti/extract/dynamodb"
	"github.com/vegarsti/extract/html"
	"github.com/vegarsti/extract/s3"
	"github.com/vegarsti/extract/textract"
	"golang.org/x/sync/errgroup"
)

func HandleRequest(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	if !req.IsBase64Encoded {
		return errorResponse(fmt.Errorf(
			"request body must have a content-type that is either image/png, image/jpeg, multipart/form-data or application/x-www-form-urlencoded, got '%s'",
			req.Headers["content-type"],
		)), nil
	}

	decodedBodyBytes, err := base64.StdEncoding.DecodeString(req.Body)
	if err != nil {
		return errorResponse(fmt.Errorf("unable to convert base64 to bytes: %w", err)), nil
	}

	imageBytes, err := getImageBytes(decodedBodyBytes, req.Headers["content-type"])
	if err != nil {
		return errorResponse(err), nil
	}

	// get table, from cache if possible, if not from textract
	identifier := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	table, err := getTable(imageBytes, identifier)
	if err != nil {
		return errorResponse(err), nil
	}
	tableBytes, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to json: %w", err)
	}

	responseMediaType, err := determineResponseMediaType(req.Headers["accept"])
	if err != nil {
		return errorResponse(err), nil
	}
	switch responseMediaType {
	case "text/html":
		return &events.APIGatewayProxyResponse{
			Headers: map[string]string{
				"Location": "https://results.extract-table.com/" + identifier,
			},
			StatusCode: 301,
		}, nil
	case "text/csv":
		csvBody := csv.FromTable(table)
		return successResponse(csvBody, "text/csv"), nil
	default:
		jsonBody := string(tableBytes) + "\n"
		return successResponse(jsonBody, "application/json"), nil
	}
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

// getTable either cached from DynamoDB if it has been processed before, or perform OCR with Textract
func getTable(imageBytes []byte, checksum string) ([][]string, error) {
	startGet := time.Now()
	tableBytes, err := dynamodb.GetTable(checksum)
	if err != nil {
		return nil, fmt.Errorf("dynamodb.GetTable: %w", err)
	}
	log.Printf("dynamodb get: %s", time.Since(startGet).String())
	if tableBytes != nil {
		var table [][]string
		if err := json.Unmarshal(tableBytes, &table); err != nil {
			return nil, fmt.Errorf("failed to convert from json: %w", err)
		}
		return table, nil
	}
	startOCR := time.Now()
	output, err := textract.Extract(imageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract: %w", err)
	}
	log.Printf("textract: %s", time.Since(startOCR).String())
	table, err := textract.ToTableFromDetectedTable(output)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to table: %w", err)
	}
	tableBytes, err = json.MarshalIndent(table, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to convert table to json: %w", err)
	}

	csvBytes := []byte(csv.FromTable(table))
	url := "https://results.extract-table.com/" + checksum
	imageURL := url + ".png" // what about jpg?
	csvURL := url + ".csv"
	htmlBytes := html.FromTable(table, imageURL, csvURL)

	g := new(errgroup.Group)
	g.Go(func() error {
		startUpload := time.Now()
		if err := s3.UploadPNG(checksum, imageBytes); err != nil {
			return err
		}
		log.Printf("s3 png %s", time.Since(startUpload).String())
		return nil
	})
	g.Go(func() error {
		startUpload := time.Now()
		if err := s3.UploadCSV(checksum, csvBytes); err != nil {
			return err
		}
		log.Printf("s3 csv %s", time.Since(startUpload).String())
		return nil
	})
	g.Go(func() error {
		startUpload := time.Now()
		if err := s3.UploadHTML(checksum, htmlBytes); err != nil {
			return err
		}
		log.Printf("s3 html %s", time.Since(startUpload).String())
		return nil
	})
	g.Go(func() error {
		startPut := time.Now()
		if err := dynamodb.PutTable(checksum, tableBytes); err != nil {
			return fmt.Errorf("dynamodb.PutTable: %w", err)
		}
		log.Printf("dynamodb put: %s", time.Since(startPut).String())
		return nil
	})
	startErrgroup := time.Now()
	if err := g.Wait(); err != nil {
		return nil, err
	}
	log.Printf("errgroup: %s", time.Since(startErrgroup).String())
	return table, nil
}

// getImageBytes from the decoded body from the HTTP request. The contentTypeHeader is needed to determine how to get the data,
// in particular if the content type is "multipart/form-data", then we need to do some more work to get the image.
func getImageBytes(decodedBodyBytes []byte, contentTypeHeader string) ([]byte, error) {
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse media type': %w", err)
	}

	if mediaType == "application/x-www-form-urlencoded" {
		log.Println("url encoded")
		s := string(decodedBodyBytes)
		log.Println("decoded body", s)
		v, err := url.ParseQuery(s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url encoded value: %w", err)
		}
		u := v.Get("url")
		if u == "" {
			return nil, fmt.Errorf("empty value for url")
		}
		log.Printf("url %s", u)
		return nil, fmt.Errorf("don't support url encoding yet")
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

// determineResponseMediaType determines what media type to return by looking at the Accept HTTP header
// The header is on the form accept: text/html, application/xhtml+xml, application/xml;q=0.9
// where the content types are listed in preferred order.
func determineResponseMediaType(acceptResponseHeader string) (string, error) {
	acceptResponseTypes := strings.Split(acceptResponseHeader, ",")
	for _, e := range acceptResponseTypes {
		mediaType, _, err := mime.ParseMediaType(e)
		if err != nil {
			return "", fmt.Errorf("unable to parse media type '%s' in Accept header: %w", e, err)
		}
		if mediaType == "text/html" {
			return mediaType, nil
		}
		if mediaType == "application/json" {
			return mediaType, nil
		}
		if mediaType == "text/csv" {
			return mediaType, nil
		}
	}
	return "application/json", nil
}
