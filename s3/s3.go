package s3

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func Upload(sess *session.Session, identifier string, imageData []byte, csvData []byte, htmlData []byte) error {
	uploader := s3manager.NewUploader(sess)
	files := []struct {
		extension          string
		data               []byte
		contentDisposition string
		contentType        string
	}{
		{".png", imageData, fmt.Sprintf(`attachment; filename="%s.png"`, identifier), "image/png"},
		{".csv", csvData, fmt.Sprintf(`attachment; filename="%s.csv"`, identifier), "text/csv"},
		{".html", htmlData, "inline", "text/html"},
	}
	for _, file := range files {
		filename := identifier + file.extension
		uploadParams := &s3manager.UploadInput{
			Bucket:             aws.String("results.extract-table.com"),
			Key:                aws.String(filename),
			Body:               bytes.NewReader(file.data),
			ContentDisposition: &file.contentDisposition,
			ContentType:        &file.contentType,
		}
		if _, err := uploader.Upload(uploadParams); err != nil {
			return fmt.Errorf("uploader.Upload: %v", err)
		}
	}
	return nil
}
