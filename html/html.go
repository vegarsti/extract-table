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
	Rows []Row
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
		<table>{{range .Rows}}
			<tr>{{range .Cells}}
				<td>{{.Text}}</td>{{end}}
			</tr>{{end}}
		</table>
	</body>
</html>
`

var tmpl = template.Must(template.New("table").Parse(tmplString))

func FromTable(stringTable [][]string) string {
	var table Table
	for _, row := range stringTable {
		var r Row
		for _, cell := range row {
			r.Cells = append(r.Cells, Cell{cell})
		}
		table.Rows = append(table.Rows, r)
	}
	buf := bytes.NewBufferString("")
	tmpl.Execute(buf, table)
	return buf.String()
}
