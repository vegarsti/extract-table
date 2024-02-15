package box

import (
	"sort"
	"strings"
)

// Box is a data structure representing a box in an image,
// with x and y float coordinates, and the text inside the box.
type Box struct {
	XLeft   float64
	XRight  float64
	YBottom float64
	YTop    float64
	Content string
}

// Inside other box o if it is completely inside,
// i.e. all coordinates for the outer box are more extreme or overlap
func (b Box) Inside(o Box) bool {
	return o.XLeft <= b.XLeft && o.XRight >= b.XRight && o.YTop <= b.YTop && o.YBottom >= b.YBottom
}

// Box's x coordinates overlap with the region of left and right
func (b Box) XOverlap(left float64, right float64) bool {
	// box is to the left
	if b.XRight < left {
		return false
	}
	// box is to the right
	if b.XLeft > right {
		return false
	}
	return true
}

// Box's y coordinates overlap with the region of top and bottom
func (b Box) YOverlap(top float64, bottom float64) bool {
	// box is above
	if b.YBottom < top {
		return false
	}
	// box is below
	if b.YTop > bottom {
		return false
	}
	return true
}

// Find all non-overlapping regions in x direction of coordinates
// where there is at least one box.
func XRegions(boxes []Box) [][]float64 {
	regions := make([][]float64, 0)
	for _, b := range boxes {
		found := false
		for _, region := range regions {
			left := region[0]
			right := region[1]
			if b.XOverlap(left, right) {
				region[0] = min(left, b.XLeft)
				region[1] = max(right, b.XRight)
				found = true
				break
			}
		}
		if !found {
			regions = append(regions, []float64{b.XLeft, b.XRight})
		}
	}
	return mergeRegions(regions)
}

// Find all non-overlapping regions in y direction of coordinates
// where there is at least one box.
func YRegions(boxes []Box) [][]float64 {
	regions := make([][]float64, 0)
	// iterate over all boxes
	for _, b := range boxes {
		// overlap = has overlap with existing region
		overlap := false
		for _, region := range regions {
			top := region[0]
			bottom := region[1]
			// box overlaps with current region; expand the region
			if b.YOverlap(top, bottom) {
				region[0] = min(top, b.YTop)
				region[1] = max(bottom, b.YBottom)
				overlap = true
				break
			}
		}
		// does not have overlap with existing region
		if !overlap {
			regions = append(regions, []float64{b.YTop, b.YBottom})
		}
	}
	// we're likely to have some duplicate regions; merge so we don't have overlap
	return mergeRegions(regions)
}

// Remove duplicates
func mergeRegions(regions [][]float64) [][]float64 {
	newRegions := make([][]float64, 0)
	for _, r := range regions {
		overlap := false
		for i, n := range newRegions {
			// this region (n) is inside other (r); skip it
			if r[0] <= n[0] && n[1] <= r[1] {
				overlap = true
				break
			}
			// this region (n) is completely outside other (r)
			if n[0] <= r[0] && r[1] <= n[1] {
				overlap = true
				break
			}
			// this region is to the left, but not on the right
			if n[0] <= r[0] && n[1] <= r[1] && r[0] <= n[1] {
				overlap = true
				newRegions[i][0] = n[0]
				newRegions[i][1] = r[1]
				break
			}
			// this region is to the right, but not to the left
			if r[0] <= n[0] && r[1] <= n[1] && n[0] <= r[1] {
				overlap = true
				newRegions[i][0] = n[0]
				newRegions[i][1] = n[1]
				break
			}
		}
		if !overlap {
			newRegions = append(newRegions, r)
		}
	}
	return newRegions
}

func min(f1, f2 float64) float64 {
	if f1 < f2 {
		return f1
	}
	return f2
}

func max(f1, f2 float64) float64 {
	if f1 < f2 {
		return f2
	}
	return f1
}

// Given non-overlapping regions in x direction and y direction, create
func CartesianProduct(xRegions [][]float64, yRegions [][]float64) [][]Box {
	rows := make([][]Box, len(yRegions))
	for i, yRegion := range yRegions {
		rows[i] = make([]Box, len(xRegions))
		for j, xRegion := range xRegions {
			rows[i][j] = Box{
				XLeft:   xRegion[0],
				XRight:  xRegion[1],
				YBottom: yRegion[1],
				YTop:    yRegion[0],
			}
		}
	}
	return rows
}

type Cell []Box

func (c Cell) Len() int {
	return len(c)
}
func (c Cell) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Cell) Less(i, j int) bool {
	// sort boxes by row, then by x
	// find row by checking if the bottom y is above the top y.
	// within a row, use xLeft

	// different row
	if c[i].YBottom < c[j].YTop {
		return true // i should be first
	}
	if c[i].YTop > c[j].YBottom {
		return false // i should be first
	}
	// same row, so compare x
	return c[i].XLeft < c[j].XLeft
}

func Assign(rows [][]Box, boxes []Box) {
	// Fill cell with corresponding boxes
	// Find word boxes to put in this cell
	// Assign table cell (x, y) to each box:
	// find all
	for i := range rows {
		for j := range rows[i] {
			c := Cell(boxes)
			sort.Sort(c)
			boxes = []Box(c)
			for _, b := range boxes {
				if b.Inside(rows[i][j]) {
					rows[i][j].Content = strings.Trim(rows[i][j].Content+" "+b.Content, " ")
				}
			}
		}
	}
}

func ToTable(boxes []Box) ([][]Box, [][]string) {
	// TODO: Explain this better
	// Find all regions in x direction with a box,
	// and same in y direction
	xRegions := XRegions(boxes)
	yRegions := YRegions(boxes)

	// Create all cells by taking the cartesian product
	// of x regions and y regions: for each x region, all y regions.
	rows := CartesianProduct(xRegions, yRegions)
	// Assign table cell (x, y) to each box
	// (mutates rows)
	Assign(rows, boxes)

	// Create table ([][]string) from [][]box.Box
	lines := make([][]string, len(rows))
	for i := range rows {
		lines[i] = make([]string, len(rows[i]))
		for j := range rows[i] {
			lines[i][j] = rows[i][j].Content
		}
	}
	return rows, lines
}
