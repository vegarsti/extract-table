package textract

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
	"github.com/vegarsti/extract"
	"github.com/vegarsti/extract/s3"
)

func Extract(bs []byte, isPDF bool) (*textract.AnalyzeDocumentOutput, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %w", err)
	}
	svc := textract.New(sess)
	tables := "TABLES"
	input := &textract.AnalyzeDocumentInput{
		Document:     &textract.Document{Bytes: bs},
		FeatureTypes: []*string{&tables},
	}
	if isPDF {
		return extractPDF(bs)
	}
	output, err := svc.AnalyzeDocument(input)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func extractPDF(bs []byte) (*textract.AnalyzeDocumentOutput, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %w", err)
	}
	identifier := fmt.Sprintf("%x", sha256.Sum256(bs))
	if err := s3.UploadPDF(identifier, bs); err != nil {
		return nil, fmt.Errorf("upload PDF: %w", err)
	}
	svc := textract.New(sess)
	bucket := "results.extract-table.com"
	name := identifier + ".pdf"
	tables := "TABLES"
	startInput := &textract.StartDocumentAnalysisInput{
		DocumentLocation: &textract.DocumentLocation{
			S3Object: &textract.S3Object{
				Bucket: &bucket,
				Name:   &name,
			},
		},
		FeatureTypes: []*string{&tables},
	}
	startOutput, err := svc.StartDocumentAnalysis(startInput)
	if err != nil {
		return nil, fmt.Errorf("start document analysis: %w", err)
	}
	getInput := &textract.GetDocumentAnalysisInput{JobId: startOutput.JobId}
	processing := true
	var getOutput *textract.GetDocumentAnalysisOutput
	for processing {
		time.Sleep(10 * time.Millisecond)
		getOutput, err = svc.GetDocumentAnalysis(getInput)
		if err != nil {
			return nil, fmt.Errorf("get document analysis: %w", err)
		}
		processing = *getOutput.JobStatus == "IN_PROGRESS"
	}
	return &textract.AnalyzeDocumentOutput{
		Blocks:           getOutput.Blocks,
		DocumentMetadata: getOutput.DocumentMetadata,
	}, nil
}

func ToTableFromDetectedTable(output *textract.AnalyzeDocumentOutput) ([][]string, error) {
	blocks := make(map[string]*textract.Block)
	var tables []*textract.Block
	for _, block := range output.Blocks {
		if *block.BlockType == "TABLE" {
			tables = append(tables, block)
		}
		blocks[*block.Id] = block
	}
	rowMap := make(map[int]map[int]string)
	if len(tables) != 1 {
		return nil, fmt.Errorf("%d tables detected, expected 1", len(tables))
	}
	b := tables[0]
	for _, r := range b.Relationships {
		if *r.Type != "CHILD" {
			continue
		}
		for _, id := range r.Ids {
			cell := blocks[*id]
			if *cell.BlockType == "CELL" {
				rowIndex := int(*cell.RowIndex)
				colIndex := int(*cell.ColumnIndex)
				if _, ok := rowMap[rowIndex]; !ok {
					rowMap[rowIndex] = make(map[int]string)
				}
				rowMap[rowIndex][colIndex] = textInCellBlock(blocks, cell)
			}
		}
	}

	var rowIndices []int
	for rowIndex := range rowMap {
		rowIndices = append(rowIndices, rowIndex)
	}
	sort.Ints(rowIndices)

	rows := make([][]string, len(rowIndices))
	for _, i := range rowIndices {
		row := rowMap[i]

		var colIndices []int
		for colIndex := range row {
			colIndices = append(colIndices, colIndex)
		}
		sort.Ints(colIndices)

		rows[i-1] = make([]string, len(colIndices))
		for _, j := range colIndices {
			cell := row[j]
			rows[i-1][j-1] = cell
		}
	}
	return rows, nil
}

func textInCellBlock(blocks map[string]*textract.Block, cell *textract.Block) string {
	var words []string
	for _, r := range cell.Relationships {
		for _, id := range r.Ids {
			if *r.Type != "CHILD" {
				continue
			}
			cell := blocks[*id]
			if *cell.BlockType != "WORD" {
				continue
			}
			words = append(words, *cell.Text)
		}
	}
	return strings.Join(words, " ")
}

func ToTableWithSplitHeuristic(output *textract.AnalyzeDocumentOutput) ([][]string, error) {
	words := make([]extract.Word, 0)
	for _, block := range output.Blocks {
		if *block.BlockType != "WORD" {
			continue
		}
		w := extract.Word{
			Text:  *block.Text,
			LeftX: 1,
			TopY:  1,
		}
		/*
			Coordinate system
				topLeft := {x: 0, y: 0}
				topRight := {x: 0, y: 1}
				bottomLeft := {x: 1, y: 0}
				bottomRight := {x: 1, y: 1}
		*/
		for _, boundingBox := range block.Geometry.Polygon {
			w.LeftX = math.Min(w.LeftX, *boundingBox.X)
			w.RightX = math.Max(w.RightX, *boundingBox.X)
			w.TopY = math.Min(w.TopY, *boundingBox.Y)
			w.BottomY = math.Max(w.BottomY, *boundingBox.Y)
		}
		words = append(words, w)
	}
	rows := extract.PartitionIntoRows(words)
	splitAt := extract.FindSplits(words)
	table := toTable(rows, splitAt, extract.SplitRowBoxesEdge)
	return table, nil
}

func toTable(rows [][]extract.Word, splitAt []float64, splitFunc func([]extract.Word, []float64) [][]extract.Word) [][]string {
	// initialize table
	table := make([][]string, len(rows))
	for i := range rows {
		table[i] = make([]string, len(splitAt)+1)
	}

	// populate table
	for i, rowBoxes := range rows {
		cellsBoxes := splitFunc(rowBoxes, splitAt)
		for j, cell := range cellsBoxes {
			wordsInCell := make([]string, len(cell))
			for k, w := range cell {
				wordsInCell[k] = w.Text
			}
			table[i][j] = strings.TrimSpace(strings.Join(wordsInCell, " "))
		}
	}
	return table
}
