package html

import (
	"bytes"
	"text/template"
)

type Cell struct {
	Text string
}

type Row struct {
	Cells []Cell
}

type Table struct {
	Rows     []Row
	ImageURL string
	CSVURL   string
}

var tmplString = `
<!DOCTYPE html>
<html>
	<head>
		<style>
			table, th, td {
				border: 1px solid black;
				border-collapse: collapse;
				padding: 5px;
			}
		</style>
	</head>
	<body>
		<a href="{{.CSVURL}}">Download CSV.</a>
		<table>{{range .Rows}}
			<tr>{{range .Cells}}
				<td>{{.Text}}</td>{{end}}
			</tr>{{end}}
		</table>
		<br />
		<img src="{{.ImageURL}}">
	</body>
</html>
`

var tmpl = template.Must(template.New("table").Parse(tmplString))

func FromTable(stringTable [][]string, imageURL string, csvURL string) []byte {
	table := Table{
		ImageURL: imageURL,
		CSVURL:   csvURL,
	}
	for _, row := range stringTable {
		var r Row
		for _, cell := range row {
			r.Cells = append(r.Cells, Cell{cell})
		}
		table.Rows = append(table.Rows, r)
	}
	buf := bytes.NewBufferString("")
	tmpl.Execute(buf, table)
	return buf.Bytes()
}
