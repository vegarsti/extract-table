package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/vegarsti/extract/dynamodb"
	"github.com/vegarsti/extract/textract"
)

var awsRegion string

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: extract-table [file ...]\n")
		os.Exit(1)
	}
	if err := readEnvVars(); err != nil {
		die(err)
	}
	filename := os.Args[1]
	mySession, err := session.NewSession()
	if err != nil {
		die(err)
	}
	imageBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		die(err)
	}

	// Check if table is stored
	checksum := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	storedBytes, err := dynamodb.GetTable(mySession, checksum)
	if err != nil {
		die(err)
	}
	if storedBytes != nil {
		var table [][]string
		json.Unmarshal(storedBytes, &table)
		writeTable(table)
		return
	}

	output, err := textract.Extract(mySession, imageBytes)
	if err != nil {
		die(err)
	}
	table, err := textract.ToTableFromDetectedTable(output)
	if err != nil {
		die(err)
	}
	writeTable(table)

	// store in dynamo db
	tableJSON, err := json.Marshal(table)
	if err != nil {
		die(err)
	}
	if err := dynamodb.PutTable(mySession, checksum[:], tableJSON); err != nil {
		die(err)
	}
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
