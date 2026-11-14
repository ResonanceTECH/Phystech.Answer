package parse

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// extractXLSX dumps each worksheet as text: "Лист: <name>" followed by rows
// of tab-separated cell values. Formulas are read as cached values only —
// macros and formulas are never executed.
func extractXLSX(data []byte) (string, error) {
	zr, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("xlsx: not a zip: %w", err)
	}

	shared, err := xlsxSharedStrings(zr)
	if err != nil {
		return "", err
	}
	sheets, err := xlsxSheetNames(zr)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "xl/worksheets/sheet") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		idx := sheetIndex(f.Name)
		name := f.Name
		if idx >= 0 && idx < len(sheets) {
			name = sheets[idx]
		}
		b.WriteString("Лист: ")
		b.WriteString(name)
		b.WriteByte('\n')
		if err := xlsxDumpSheet(rc, shared, &b); err != nil {
			rc.Close()
			return "", fmt.Errorf("xlsx: %s: %w", f.Name, err)
		}
		rc.Close()
		b.WriteByte('\n')
	}
	text := strings.TrimSpace(cleanText(b.String()))
	if text == "" {
		return "", ErrNoText
	}
	return text, nil
}

func sheetIndex(filename string) int {
	base := filename[strings.LastIndexByte(filename, '/')+1:]
	num := strings.TrimSuffix(strings.TrimPrefix(base, "sheet"), ".xml")
	i, err := strconv.Atoi(num)
	if err != nil {
		return -1
	}
	return i - 1
}

type xlSharedStrings struct {
	Items []struct {
		Text string `xml:"t"`
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

func xlsxSharedStrings(zr *zip.Reader) ([]string, error) {
	for _, f := range zr.File {
		if f.Name != "xl/sharedStrings.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		var ss xlSharedStrings
		if err := xml.NewDecoder(rc).Decode(&ss); err != nil {
			return nil, fmt.Errorf("xlsx: sharedStrings: %w", err)
		}
		out := make([]string, len(ss.Items))
		for i, it := range ss.Items {
			if it.Text != "" {
				out[i] = it.Text
				continue
			}
			var sb strings.Builder
			for _, r := range it.Runs {
				sb.WriteString(r.Text)
			}
			out[i] = sb.String()
		}
		return out, nil
	}
	return nil, nil
}

type xlWorkbook struct {
	Sheets []struct {
		Name string `xml:"name,attr"`
	} `xml:"sheets>sheet"`
}

func xlsxSheetNames(zr *zip.Reader) ([]string, error) {
	for _, f := range zr.File {
		if f.Name != "xl/workbook.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		var wb xlWorkbook
		if err := xml.NewDecoder(rc).Decode(&wb); err != nil {
			return nil, fmt.Errorf("xlsx: workbook: %w", err)
		}
		names := make([]string, len(wb.Sheets))
		for i, s := range wb.Sheets {
			names[i] = s.Name
		}
		return names, nil
	}
	return nil, nil
}

// xlsxDumpSheet streams sheet XML, emitting "cellref=value" per row.
func xlsxDumpSheet(r io.Reader, shared []string, b *strings.Builder) error {
	dec := xml.NewDecoder(r)
	var cellRef, cellType, cellVal string
	inCell, inV := false, false
	rowOpen := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				rowOpen = true
			case "c":
				inCell = true
				cellVal = ""
				cellRef, cellType = "", ""
				for _, a := range t.Attr {
					if a.Name.Local == "r" {
						cellRef = a.Value
					}
					if a.Name.Local == "t" {
						cellType = a.Value
					}
				}
			case "v":
				inV = true
			case "is":
				// inline string; text captured via CharData with inV semantics
				inV = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "c":
				if inCell {
					if cellType == "s" {
						if n, err := strconv.Atoi(cellVal); err == nil && n >= 0 && n < len(shared) {
							cellVal = shared[n]
						}
					}
					if cellVal != "" {
						fmt.Fprintf(b, "%s=%s\t", cellRef, cellVal)
					}
					inCell = false
				}
			case "v", "is":
				inV = false
			case "row":
				if rowOpen {
					b.WriteByte('\n')
					rowOpen = false
				}
			}
		case xml.CharData:
			if inV || (inCell && cellType == "inlineStr") {
				cellVal += string(t)
			}
		}
	}
}
