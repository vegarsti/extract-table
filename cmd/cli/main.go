package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

	// TODO: Detect from file extension? use `file` built-in/utility
	contentType := extract.PNG

	// Send image to OCR
	// Get/store raw output in box.Box format
	// Store image
	// Get/store ToTable output in box.Box format
	// Store image
	// Print final output in readable format

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
	boxes, err := textract.ToBoxesFromOCR(output)
	if err != nil {
		die(fmt.Errorf("failed to convert to boxes: %w", err))
	}
	bs, err := json.MarshalIndent(boxes, "", "  ")
	if err != nil {
		panic(err)
	}
	filenameBoxesRaw := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_boxes_raw.json"
	if err := os.WriteFile(filenameBoxesRaw, bs, 0644); err != nil {
		panic(err)
	}

	rows, table := box.ToTable(boxes)

	// Add boxes
	if contentType == extract.PNG {
		imageWithBoxes, err := image.AddBoxes(file.Bytes, boxes)
		if err != nil {
			log.Printf("add boxes to image 1 failed: %v", err)
		} else {
			filenameBoxes := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_boxes.png"
			if err := os.WriteFile(filenameBoxes, imageWithBoxes, 0644); err != nil {
				die(err)
			}
			rowsFlattened := make([]box.Box, 0)
			for _, row := range rows {
				rowsFlattened = append(rowsFlattened, row...)
			}
			imageWithBoxes, err = image.AddBoxes(file.Bytes, rowsFlattened)
			if err != nil {
				log.Printf("add boxes to image 2 failed: %v", err)
				file.BytesWithBoxes = []byte(imageWithBoxes)
				file.BytesWithRowBoxes = []byte(imageWithBoxes)
			}
			filenameRows := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_rows.png"
			if err := os.WriteFile(filenameRows, imageWithBoxes, 0644); err != nil {
				die(err)
			}
		}
	}
	boxesJSON, err := json.MarshalIndent(boxes, "", "  ")
	if err != nil {
		die(err)
	}
	filenameBoxesJSON := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_boxes.json"
	if err := os.WriteFile(filenameBoxesJSON, boxesJSON, 0644); err != nil {
		die(err)
	}

	fmt.Printf("%+v\n", table)
	// filenameTable := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_table.txt"
	// f, err := os.Create(filenameTable)
	// if err != nil {
	// 	die(err)
	// }
	// writeTable(bufio.NewWriter(f), table)
	// writeTable(os.Stdout, table)

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
func writeTable(wr io.Writer, table [][]string) {
	fmt.Println(table)
	w := tabwriter.NewWriter(wr, 4, 4, 2, ' ', tabwriter.Debug)
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
