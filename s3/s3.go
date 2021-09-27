package s3

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func UploadPNG(sess *session.Session, identifier string, data []byte) error {
	uploader := s3manager.NewUploader(sess)
	contentDisposition := fmt.Sprintf(`attachment; filename="%s.png"`, identifier)
	contentType := "image/png"
	uploadParams := &s3manager.UploadInput{
		Bucket:             aws.String("results.extract-table.com"),
		Key:                aws.String(identifier + ".png"),
		Body:               bytes.NewReader(data),
		ContentDisposition: &contentDisposition,
		ContentType:        &contentType,
	}
	if _, err := uploader.Upload(uploadParams); err != nil {
		return fmt.Errorf("uploadPNG: %v", err)
	}
	return nil
}

func UploadCSV(sess *session.Session, identifier string, data []byte) error {
	uploader := s3manager.NewUploader(sess)
	contentType := "text/csv"
	contentDisposition := fmt.Sprintf(`attachment; filename="%s.csv"`, identifier)
	uploadParams := &s3manager.UploadInput{
		Bucket:             aws.String("results.extract-table.com"),
		Key:                aws.String(identifier + ".csv"),
		Body:               bytes.NewReader(data),
		ContentDisposition: &contentDisposition,
		ContentType:        &contentType,
	}
	if _, err := uploader.Upload(uploadParams); err != nil {
		return fmt.Errorf("uploadCSV: %v", err)
	}
	return nil
}

func UploadHTML(sess *session.Session, identifier string, data []byte) error {
	uploader := s3manager.NewUploader(sess)
	contentType := "text/html"
	contentDisposition := "inline"
	uploadParams := &s3manager.UploadInput{
		Bucket:             aws.String("results.extract-table.com"),
		Key:                aws.String(identifier),
		Body:               bytes.NewReader(data),
		ContentDisposition: &contentDisposition,
		ContentType:        &contentType,
	}
	if _, err := uploader.Upload(uploadParams); err != nil {
		return fmt.Errorf("uploadHTML: %v", err)
	}
	return nil
}
