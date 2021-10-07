package s3

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func UploadPNG(identifier string, data []byte) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create session: %w", err)
	}
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

func UploadPDF(identifier string, data []byte) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create session: %w", err)
	}
	uploader := s3manager.NewUploader(sess)
	contentDisposition := fmt.Sprintf(`attachment; filename="%s.pdf"`, identifier)
	contentType := "application/pdf"
	uploadParams := &s3manager.UploadInput{
		Bucket:             aws.String("results.extract-table.com"),
		Key:                aws.String(identifier + ".pdf"),
		Body:               bytes.NewReader(data),
		ContentDisposition: &contentDisposition,
		ContentType:        &contentType,
	}
	if _, err := uploader.Upload(uploadParams); err != nil {
		return fmt.Errorf("uploadPNG: %v", err)
	}
	return nil
}

func UploadCSV(identifier string, data []byte) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create session: %w", err)
	}
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

func UploadHTML(identifier string, data []byte) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create session: %w", err)
	}
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
