package main

import (
	"log"
	"math"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/textract"
)

var awsRegion string

type word struct {
	text    string
	xLeft   float64
	xRight  float64
	yTop    float64
	yBottom float64
}

type byXLeft []word

func (s byXLeft) Len() int {
	return len(s)
}
func (s byXLeft) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byXLeft) Less(i, j int) bool {
	return s[i].xLeft < s[j].xLeft
}

type byRow []word

func (s byRow) Len() int {
	return len(s)
}
func (s byRow) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byRow) Less(i, j int) bool {
	// sort boxes by row, then by x
	// find row by checking if the bottom y is above the top y.
	// within a row, use xLeft

	if s[i].yBottom < s[j].yTop {
		return true // i is on row before j
	}
	if s[i].yTop > s[j].yBottom {
		return false // j is on row before i
	}

	// same row, so compare x
	return s[i].xLeft < s[j].xLeft
}

type bySize [][2]float64

func (s bySize) Len() int {
	return len(s)
}
func (s bySize) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s bySize) Less(i, j int) bool {
	return s[i][1]-s[i][0] < s[j][1]-s[j][0]
}

// words need to be sorted by xLeft
func findGroups(words []word) [][2]float64 {
	// sort words by xleft
	sort.Sort(byXLeft(words))
	splits := make([][2]float64, 0)
	xRight := float64(0)
	for i, w := range words {
		if w.xLeft > xRight && i > 0 {
			splits = append(splits, [2]float64{xRight, w.xLeft})
		}
		xRight = w.xRight
	}
	sort.Sort(bySize(splits))
	return splits
}

func splitRowBoxes(words []word, xs []float64) [][]word {
	sort.Sort(byXLeft(words))
	partitions := make([][]word, len(xs)+1)
	for i := range partitions {
		partitions[i] = make([]word, 0)
	}

	i := 0
	for _, word := range words {
		if i < len(xs) && word.xLeft > xs[i] {
			i++
		}
		partitions[i] = append(partitions[i], word)
	}
	return partitions
}

// words need to be sorted by row order,
// assume one row has max(yBottom) < min(yMax) other row
func partitionIntoRows(words []word) [][]word {
	// sort by row
	sort.Sort(byRow(words))

	partitions := make([][]word, 0)
	firstRow := make([]word, 0)
	firstRow = append(firstRow, words[0])
	partitions = append(partitions, firstRow)
	i := 0
	prevX := float64(0)
	for _, w := range words[1:] {
		// if new row
		if w.xLeft < prevX {
			i++
			newRow := make([]word, 0)
			partitions = append(partitions, newRow)
		}
		partitions[i] = append(partitions[i], w)
		prevX = w.xLeft
	}
	return partitions
}

func toTable(output *textract.DetectDocumentTextOutput) ([][]string, error) {
	words := make([]word, 0)
	for _, block := range output.Blocks {
		if *block.BlockType != "WORD" {
			continue
		}
		var w word
		w.text = *block.Text
		/*
			Coordinate system
				topLeft := {x: 0, y: 0}
				topRight := {x: 0, y: 1}
				bottomLeft := {x: 1, y: 0}
				bottomRight := {x: 1, y: 1}
		*/
		w.xLeft = 1
		w.yTop = 1
		for _, boundingBox := range block.Geometry.Polygon {
			w.xLeft = math.Min(w.xLeft, *boundingBox.X)
			w.xRight = math.Max(w.xRight, *boundingBox.X)
			w.yTop = math.Min(w.yTop, *boundingBox.Y)
			w.yBottom = math.Max(w.yBottom, *boundingBox.Y)
		}
		words = append(words, w)
	}

	// find partitions
	intervals := findGroups(words)
	sort.Sort(bySize(intervals))
	log.Println(intervals)
	splitAt := make([]float64, len(intervals))
	for i, interval := range intervals {
		splitAt[i] = interval[0] + ((interval[1] - interval[0]) / 2)
	}
	// how many columns?
	nColumns := 3
	nSplits := nColumns - 1
	splitAt = splitAt[:nSplits]
	sort.Float64s(splitAt)

	rows := partitionIntoRows(words)

	// initialize table
	table := make([][]string, len(rows))
	for i := range rows {
		table[i] = make([]string, len(splitAt)+1)
	}

	// populate table
	for i, rowBoxes := range rows {
		cellsBoxes := splitRowBoxes(rowBoxes, splitAt)
		for j, cell := range cellsBoxes {
			wordsInCell := make([]string, len(cell))
			for k, w := range cell {
				wordsInCell[k] = w.text
			}
			table[i][j] = strings.TrimSpace(strings.Join(wordsInCell, " "))
		}
	}
	return table, nil
}
