package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vegarsti/extract"
	"github.com/vegarsti/extract/box"
	"github.com/vegarsti/extract/image"
	"github.com/vegarsti/extract/textract"
)

var awsRegion string

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: extract-table [file ...] isPDF \n")
		os.Exit(1)
	}
	if err := readEnvVars(); err != nil {
		die(err)
	}
	filename := os.Args[1]
	imageBytes, err := os.ReadFile(filename)
	if err != nil {
		die(err)
	}

	contentType := extract.PNG
	if len(os.Args) == 3 {
		boolean := strings.ToLower(os.Args[2])
		isPDF, ok := map[string]bool{"f": false, "false": false, "t": true, "true": true}[boolean]
		if !ok {
			die(fmt.Errorf("'%s' is invalid for boolean flag isPDF", boolean))
		}
		if isPDF {
			contentType = extract.PDF
		}
	}

	// Check if table is stored
	checksum := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	fmt.Println(checksum)

	// Get from cache
	// storedBytes, err := dynamodb.GetTable(checksum)
	// if err != nil {
	// 	die(err)
	// }
	// if storedBytes != nil {
	// 	var table [][]string
	// 	json.Unmarshal(storedBytes, &table)
	// 	writeTable(table)
	// 	return
	// }

	file := &extract.File{
		Bytes:       imageBytes,
		ContentType: contentType,
	}

	output, err := textract.DetectDocumentText(file)
	if err != nil {
		die(fmt.Errorf("textract text detection failed: %w", err))
	}
	boxes, err := textract.ToLinesFromOCR(output)
	if err != nil {
		die(fmt.Errorf("failed to convert to boxes: %w", err))
	}
	rows, table := box.ToTable(boxes)

	// Add boxes
	if contentType == extract.PNG {
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
				file.BytesWithBoxes = []byte(newEncodedImage)
				file.BytesWithRowBoxes = []byte(newEncodedImage2)
			}
			fmt.Println("hello")
		}
	}

	writeTable(table)

	// store in dynamo db
	// tableJSON, err := json.Marshal(table)
	// if err != nil {
	// 	die(err)
	// }
	// if err := dynamodb.PutTable(checksum[:], tableJSON, []byte{}); err != nil {
	// 	die(err)
	// }
}

func readEnvVars() error {
	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		return fmt.Errorf("set environment variable AWS_REGION")
	}
	return nil
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "extract-table: %v\n", err)
	os.Exit(1)
}

// writeTeable to stdout
func writeTable(table [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 4, 4, 2, ' ', tabwriter.Debug)
	for _, row := range table {
		for j, cell := range row {
			fmt.Fprint(w, cell)
			if j < len(row)-1 {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintf(w, "\n")
	}
	w.Flush()
}
