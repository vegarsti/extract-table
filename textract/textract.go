package textract

import (
	"math"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
	"github.com/vegarsti/extract"
)

func Extract(mySession *session.Session, bs []byte) (*textract.DetectDocumentTextOutput, error) {
	svc := textract.New(mySession)
	input := &textract.DetectDocumentTextInput{
		Document: &textract.Document{Bytes: bs},
	}
	output, err := svc.DetectDocumentText(input)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func ToTable(output *textract.DetectDocumentTextOutput) ([][]string, error) {
	words := make([]extract.Word, 0)
	for _, block := range output.Blocks {
		if *block.BlockType != "WORD" {
			continue
		}
		var w extract.Word
		w.Text = *block.Text
		/*
			Coordinate system
				topLeft := {x: 0, y: 0}
				topRight := {x: 0, y: 1}
				bottomLeft := {x: 1, y: 0}
				bottomRight := {x: 1, y: 1}
		*/
		w.LeftX = 1
		w.TopY = 1
		for _, boundingBox := range block.Geometry.Polygon {
			w.LeftX = math.Min(w.LeftX, *boundingBox.X)
			w.RightX = math.Max(w.RightX, *boundingBox.X)
			w.TopY = math.Min(w.TopY, *boundingBox.Y)
			w.BottomY = math.Max(w.BottomY, *boundingBox.Y)
		}
		words = append(words, w)
	}

	// find partitions
	intervals := extract.FindGroups(words)
	sort.Sort(extract.BySize(intervals))
	splitAt := make([]float64, len(intervals))
	for i, interval := range intervals {
		splitAt[i] = interval[0] + ((interval[1] - interval[0]) / 2)
	}
	// how many columns?
	nColumns := 3
	nSplits := nColumns - 1
	splitAt = splitAt[:nSplits]
	sort.Float64s(splitAt)

	rows := extract.PartitionIntoRows(words)

	// initialize table
	table := make([][]string, len(rows))
	for i := range rows {
		table[i] = make([]string, len(splitAt)+1)
	}

	// populate table
	for i, rowBoxes := range rows {
		cellsBoxes := extract.SplitRowBoxes(rowBoxes, splitAt)
		for j, cell := range cellsBoxes {
			wordsInCell := make([]string, len(cell))
			for k, w := range cell {
				wordsInCell[k] = w.Text
			}
			table[i][j] = strings.TrimSpace(strings.Join(wordsInCell, " "))
		}
	}
	return table, nil
}
