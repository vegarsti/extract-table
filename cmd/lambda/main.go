package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vegarsti/extract"
	"github.com/vegarsti/extract/box"
	"github.com/vegarsti/extract/csv"
	"github.com/vegarsti/extract/dynamodb"
	"github.com/vegarsti/extract/html"
	"github.com/vegarsti/extract/image"
	"github.com/vegarsti/extract/s3"
	"github.com/vegarsti/extract/textract"
	"golang.org/x/sync/errgroup"
)

func HandleRequest(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	// ensure headers are lower-case (according to the spec, they are case insensitive)
	reqHeaders := make(map[string]string)
	for header, value := range req.Headers {
		reqHeaders[strings.TrimSpace(strings.ToLower(header))] = value
		log.Printf("%s: %s", header, value)
	}

	if !req.IsBase64Encoded {
		return errorResponse(fmt.Errorf(
			"request body must have a content-type that is either image/png, image/jpeg, application/pdf, multipart/form-data or application/x-www-form-urlencoded, got '%s'",
			reqHeaders["content-type"],
		)), nil
	}

	decodedBodyBytes, err := base64.StdEncoding.DecodeString(req.Body)
	if err != nil {
		return errorResponse(fmt.Errorf("unable to convert base64 to bytes: %w", err)), nil
	}

	// TODO: Re-enable requiring API key
	// apiKey, err := getAPIKey(decodedBodyBytes, reqHeaders["content-type"], reqHeaders["api-key"])
	// if err != nil {
	// 	return errorResponse(fmt.Errorf("error getting api key: %w", err)), nil
	// }
	// // check if apiKey is valid
	// log.Printf("api-key was: '%s'", apiKey)
	// if apiKey == "" {
	// 	err := fmt.Errorf("no api-key was provided")
	// 	return errorResponse(err), nil
	// }
	// valid, err := dynamodb.VerifyAPIKey(apiKey)
	// if err != nil {
	// 	err := fmt.Errorf("verify api key: %w", err)
	// 	return errorResponse(err), nil
	// }
	// if !valid {
	// 	err := fmt.Errorf("API key '%s' is invalid, please email vegard.stikbakke@gmail.com to get a free api key", apiKey)
	// 	return errorResponse(err), nil
	// }

	file, err := getFile(decodedBodyBytes, reqHeaders["content-type"])
	if err != nil {
		return errorResponse(err), nil
	}

	// get table, from cache if possible, if not from textract
	table, err := getTable(file)
	if err != nil {
		return errorResponse(err), nil
	}
	tableBytes, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to json: %w", err)
	}

	responseMediaType, err := determineResponseMediaType(reqHeaders["accept"])
	if err != nil {
		return errorResponse(err), nil
	}
	url := "https://results.extract-table.com/" + file.Checksum
	log.Println(url)
	log.Printf("responsemediatype is %s", responseMediaType)
	switch responseMediaType {
	case "text/html":
		return &events.APIGatewayProxyResponse{
			Headers: map[string]string{
				"Location": url,
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
func getTable(file *extract.File) ([][]string, error) {
	startGet := time.Now()
	tableBytes, err := dynamodb.GetTable(file.Checksum)
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
	// Old: Textract's Analyze Document
	// output, err := textract.AnalyzeDocument(file)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to extract: %w", err)
	// }
	startOCR := time.Now()
	// Don't use Textract's Analyze Document, use OCR and custom algorithm instead
	output, err := textract.DetectDocumentText(file)
	log.Printf("textract: %s", time.Since(startOCR).String())
	if err != nil {
		return nil, fmt.Errorf("textract text detection failed: %w", err)
	}
	startAlgorithm := time.Now()
	boxes, err := textract.ToLinesFromOCR(output)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to boxes: %w", err)
	}
	rows, table := box.ToTable(boxes)

	// Add boxes
	if file.ContentType == extract.PNG {
		newEncodedImage, err := image.AddBoxes(file.Bytes, boxes)
		if err != nil {
			log.Printf("add boxes to image 1 failed: %v", err)
		} else {
			rowsFlattened := make([]box.Box, 0)
			for _, row := range rows {
				rowsFlattened = append(rowsFlattened, row...)
			}
			newEncodedImage2, err := image.AddBoxes(file.Bytes, rowsFlattened)
			if err != nil {
				log.Printf("add boxes to image 2 failed: %v", err)
			} else {
				file.BytesWithBoxes = []byte(newEncodedImage)
				file.BytesWithRowBoxes = []byte(newEncodedImage2)
			}
		}
	}

	log.Printf("ocr-to-table: %s", time.Since(startAlgorithm).String())
	if err != nil {
		return nil, fmt.Errorf("failed to convert to table: %w", err)
	}
	tableBytes, err = json.MarshalIndent(table, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to convert table to json: %w", err)
	}

	csvBytes := []byte(csv.FromTable(table))
	url := "https://results.extract-table.com/" + file.Checksum
	imageURL := url + ".png" // what about jpg?
	csvURL := url + ".csv"
	pdfURL := url + ".pdf"
	htmlBytes := html.FromTable(table, file.ContentType, imageURL, csvURL, pdfURL)

	g := new(errgroup.Group)
	g.Go(func() error {
		startUpload := time.Now()
		if err := s3.UploadPNG(file.Checksum, file.Bytes); err != nil {
			return err
		}
		log.Printf("s3 png %s", time.Since(startUpload).String())
		return nil
	})
	g.Go(func() error {
		startUpload := time.Now()
		if err := s3.UploadCSV(file.Checksum, csvBytes); err != nil {
			return err
		}
		log.Printf("s3 csv %s", time.Since(startUpload).String())
		return nil
	})
	g.Go(func() error {
		startUpload := time.Now()
		if err := s3.UploadHTML(file.Checksum, htmlBytes); err != nil {
			return err
		}
		log.Printf("s3 html %s", time.Since(startUpload).String())
		return nil
	})
	g.Go(func() error {
		startPut := time.Now()
		if err := dynamodb.PutTable(file.Checksum, tableBytes); err != nil {
			return fmt.Errorf("dynamodb.PutTable: %w", err)
		}
		log.Printf("dynamodb put: %s", time.Since(startPut).String())
		return nil
	})
	// Upload the image with boxes to S3
	log.Printf("len of file.BytesWithBoxes: %d", len(file.BytesWithBoxes))
	if file.ContentType == extract.PNG && len(file.BytesWithBoxes) > 0 {
		g.Go(func() error {
			startUpload := time.Now()
			if err := s3.UploadPNG(file.Checksum+"_boxes", file.BytesWithBoxes); err != nil {
				return err
			}
			log.Printf("s3 png %s", time.Since(startUpload).String())
			return nil
		})
	}
	log.Printf("len of file.BytesWithRowBoxes: %d", len(file.BytesWithRowBoxes))
	if file.ContentType == extract.PNG && len(file.BytesWithRowBoxes) > 0 {
		g.Go(func() error {
			startUpload := time.Now()
			if err := s3.UploadPNG(file.Checksum+"_rows", file.BytesWithRowBoxes); err != nil {
				return err
			}
			log.Printf("s3 png %s", time.Since(startUpload).String())
			return nil
		})
	}

	startErrgroup := time.Now()
	if err := g.Wait(); err != nil {
		return nil, err
	}
	log.Printf("errgroup: %s", time.Since(startErrgroup).String())
	return table, nil
}

func getAPIKey(decodedBodyBytes []byte, contentTypeHeader string, apiKeyHeader string) (string, error) {
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return "", fmt.Errorf("failed to parse media type: %w", err)
	}

	if mediaType == "application/x-www-form-urlencoded" {
		s := string(decodedBodyBytes)
		v, err := url.ParseQuery(s)
		if err != nil {
			return "", fmt.Errorf("failed to parse url encoded value: %w", err)
		}
		apiKey := v.Get("api-key")
		return apiKey, nil
	}
	if mediaType == "multipart/form-data" {
		decodedBody := string(decodedBodyBytes)
		reader := multipart.NewReader(strings.NewReader(decodedBody), params["boundary"])

		tenMBInBytes := 10000000
		form, err := reader.ReadForm(int64(tenMBInBytes))
		if err != nil {
			return "", fmt.Errorf("failed to read form': %w", err)
		}

		for key, values := range form.Value {
			if key == "api-key" {
				if len(values) != 1 {
					log.Printf("more than one value for api key")
				}
				return values[0], nil
			}
		}
		return "", nil
	}
	return apiKeyHeader, nil
}

// getFile from the decoded body from the HTTP request. The contentTypeHeader is needed to determine how to get the data,
// in particular if the content type is "multipart/form-data", then we need to do some more work to get the file.
func getFile(decodedBodyBytes []byte, contentTypeHeader string) (*extract.File, error) {
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse media type: %w", err)
	}

	if mediaType == "application/x-www-form-urlencoded" {
		s := string(decodedBodyBytes)
		v, err := url.ParseQuery(s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url encoded value: %w", err)
		}
		u := v.Get("url")
		if u == "" {
			return nil, fmt.Errorf("empty value for url")
		}
		resp, err := http.Get(u)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch url '%s': %w", u, err)
		}
		bs, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response from url fetch: %w", err)
		}
		if len(bs) == 0 {
			return nil, fmt.Errorf("empty response from url")
		}
		if strings.Contains(u, ".pdf") {
			return extract.NewPDF(bs), nil
		}
		if strings.Contains(u, ".jpg") || strings.Contains(u, ".jpeg") {
			return extract.NewJPG(bs), nil
		}
		return extract.NewPNG(bs), nil
	}
	if mediaType == "image/png" {
		return extract.NewPNG(decodedBodyBytes), nil
	}
	if mediaType == "image/jpeg" {
		return extract.NewJPG(decodedBodyBytes), nil
	}
	if mediaType == "application/pdf" {
		return extract.NewPDF(decodedBodyBytes), nil
	}

	if mediaType != "multipart/form-data" {
		return nil, fmt.Errorf("invalid media type: %s", mediaType)
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

	log.Printf("multipart form request body values")
	for key, values := range form.Value {
		for _, value := range values {
			log.Printf("%s: %s", key, value)
		}
	}

	log.Printf("multipart form file")
	log.Printf("number of files: %d", len(file))
	log.Printf("%s: %s", "filename", file[0].Filename)
	for key, values := range form.Value {
		for _, value := range values {
			log.Printf("%s: %s", key, value)
		}
	}
	log.Printf("multipart form file header")
	for key, values := range file[0].Header {
		for _, value := range values {
			log.Printf("%s: %s", key, value)
		}
	}

	contentType := file[0].Header.Get("content-type")
	mediaType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse media type in form: %w", err)
	}

	f, err := file[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file': %w", err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read file': %w", err)
	}

	if mediaType == "image/png" {
		return extract.NewPNG(data), nil
	}
	if mediaType == "image/jpeg" {
		return extract.NewJPG(data), nil
	}
	if mediaType == "application/pdf" {
		return extract.NewPDF(data), nil
	}

	return nil, fmt.Errorf("invalid media type: %s", mediaType)
}

// determineResponseMediaType determines what media type to return by looking at the Accept HTTP header
// The header is on the form accept: text/html, application/xhtml+xml, application/xml;q=0.9
// where the content types are listed in preferred order.
func determineResponseMediaType(acceptResponseHeader string) (string, error) {
	log.Printf("Accept: '%s'", acceptResponseHeader)
	if acceptResponseHeader == "" {
		return "application/json", nil
	}
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
