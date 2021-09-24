package s3

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func Upload(sess *session.Session, identifier string, imageData []byte, csvData []byte) (string, string, error) {
	uploader := s3manager.NewUploader(sess)
	files := []struct {
		extension string
		data      []byte
		url       *string
	}{
		{extension: ".png", data: imageData},
		{extension: ".csv", data: csvData},
	}
	for _, file := range files {
		filename := identifier + file.extension
		uploadParams := &s3manager.UploadInput{
			Bucket: aws.String("extract-table"),
			Key:    aws.String(filename),
			Body:   bytes.NewReader(file.data),
		}
		if _, err := uploader.Upload(uploadParams); err != nil {
			return "", "", fmt.Errorf("uploader.Upload: %v", err)
		}
		url := fmt.Sprintf("https://extract-table.s3.eu-west-1.amazonaws.com/%s%s", identifier, file.extension)
		file.url = &url
	}
	return *files[0].url, *files[1].url, nil
}
