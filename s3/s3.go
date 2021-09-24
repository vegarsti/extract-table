package s3

import (
	"bytes"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func Upload(sess *session.Session, identifier string, imageData []byte, csvData []byte, htmlData []byte) error {
	uploader := s3manager.NewUploader(sess)
	files := []struct {
		extension string
		data      []byte
	}{
		{extension: ".png", data: imageData},
		{extension: ".csv", data: csvData},
		{extension: ".html", data: htmlData},
	}
	for _, file := range files {
		filename := identifier + file.extension
		uploadParams := &s3manager.UploadInput{
			Bucket: aws.String("extract-table"),
			Key:    aws.String(filename),
			Body:   bytes.NewReader(file.data),
		}
		if _, err := uploader.Upload(uploadParams); err != nil {
			return fmt.Errorf("uploader.Upload: %v", err)
		}
	}
	log.Println("uploaded files")
	return nil
}
