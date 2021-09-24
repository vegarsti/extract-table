package csv

import (
	"bytes"
	"encoding/csv"
)

func FromTable(table [][]string) string {
	s := &bytes.Buffer{}
	writer := csv.NewWriter(s)
	for _, row := range table {
		writer.Write(row)
	}
	writer.Flush()
	return s.String()
}
