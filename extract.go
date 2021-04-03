package extract

import (
	"sort"
)

type Word struct {
	Text    string
	LeftX   float64
	RightX  float64
	TopY    float64
	BottomY float64
}

type byXLeft []Word

func (s byXLeft) Len() int {
	return len(s)
}
func (s byXLeft) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byXLeft) Less(i, j int) bool {
	return s[i].LeftX < s[j].LeftX
}

type byRow []Word

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

	if s[i].BottomY < s[j].TopY {
		return true // i is on row before j
	}
	if s[i].TopY > s[j].BottomY {
		return false // j is on row before i
	}

	// same row, so compare x
	return s[i].LeftX < s[j].LeftX
}

type BySize [][2]float64

func (s BySize) Len() int {
	return len(s)
}
func (s BySize) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s BySize) Less(i, j int) bool {
	return s[i][1]-s[i][0] < s[j][1]-s[j][0]
}

// words need to be sorted by xLeft
func FindGroups(words []Word) [][2]float64 {
	// sort words by xleft
	sort.Sort(byXLeft(words))
	splits := make([][2]float64, 0)
	xRight := float64(0)
	for i, w := range words {
		if w.LeftX > xRight && i > 0 {
			splits = append(splits, [2]float64{xRight, w.LeftX})
		}
		xRight = w.RightX
	}
	sort.Sort(BySize(splits))
	return splits
}

func SplitRowBoxes(words []Word, xs []float64) [][]Word {
	sort.Sort(byXLeft(words))
	partitions := make([][]Word, len(xs)+1)
	for i := range partitions {
		partitions[i] = make([]Word, 0)
	}

	i := 0
	for _, word := range words {
		if i < len(xs) && word.LeftX > xs[i] {
			i++
		}
		partitions[i] = append(partitions[i], word)
	}
	return partitions
}

// words need to be sorted by row order,
// assume one row has max(yBottom) < min(yMax) other row
func PartitionIntoRows(words []Word) [][]Word {
	// sort by row
	sort.Sort(byRow(words))

	partitions := make([][]Word, 0)
	firstRow := make([]Word, 0)
	firstRow = append(firstRow, words[0])
	partitions = append(partitions, firstRow)
	i := 0
	prevX := float64(0)
	for _, w := range words[1:] {
		// if new row
		if w.LeftX < prevX {
			i++
			newRow := make([]Word, 0)
			partitions = append(partitions, newRow)
		}
		partitions[i] = append(partitions[i], w)
		prevX = w.LeftX
	}
	return partitions
}
