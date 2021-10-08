package html

import (
	"bytes"
	"log"
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
	PDFURL   string
}

var imageHTMLTemplateString = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
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

var pdfHTMLTemplateString = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
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
		<a href="{{.PDFURL}}">Original PDF.</a>
	</body>
</html>
`

var imageHTMLTemplate = template.Must(template.New("table").Parse(imageHTMLTemplateString))
var pdfHTMLTemplate = template.Must(template.New("table").Parse(pdfHTMLTemplateString))

func FromTable(stringTable [][]string, mediaType string, imageURL string, csvURL string, pdfURL string) []byte {
	log.Printf("creating html for media type %s", mediaType)
	var table Table
	table.CSVURL = csvURL
	buf := bytes.NewBufferString("")
	for _, row := range stringTable {
		var r Row
		for _, cell := range row {
			r.Cells = append(r.Cells, Cell{cell})
		}
		table.Rows = append(table.Rows, r)
	}
	if mediaType == "pdf" {
		log.Println("pdf")
		table.PDFURL = pdfURL
		pdfHTMLTemplate.Execute(buf, table)
	} else {
		table.ImageURL = imageURL
		imageHTMLTemplate.Execute(buf, table)
	}
	return buf.Bytes()
}
