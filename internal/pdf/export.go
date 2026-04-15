package pdf

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"astrolabe/internal/astrology"
)

const (
	pageWidth      = 595
	pageHeight     = 842
	leftMargin     = 48
	topStart       = 792
	lineHeight     = 18
	maxLinesPerPage = 36
)

type pdfObject struct {
	id   int
	body []byte
}

func BuildReport(resp astrology.NatalChartResponse) []byte {
	lines := buildLines(resp)
	pages := chunkLines(lines, maxLinesPerPage)
	if len(pages) == 0 {
		pages = [][]string{{"Astrolabe Report"}}
	}

	objects := make([]pdfObject, 0, 5+len(pages)*2)
	pageIDs := make([]int, 0, len(pages))

	fontID := 3
	descFontID := 4
	fontDescriptorID := 5
	nextID := 6

	for _, pageLines := range pages {
		contentID := nextID
		pageID := nextID + 1
		nextID += 2

		content := buildPageContent(pageLines)
		objects = append(objects, pdfObject{
			id: contentID,
			body: []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)),
		})
		objects = append(objects, pdfObject{
			id: pageID,
			body: []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", pageWidth, pageHeight, fontID, contentID)),
		})
		pageIDs = append(pageIDs, pageID)
	}

	objects = append(objects,
		pdfObject{id: 1, body: []byte("<< /Type /Catalog /Pages 2 0 R >>")},
		pdfObject{id: 2, body: []byte(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pageIDs), joinObjectRefs(pageIDs)))},
		pdfObject{id: fontID, body: []byte(fmt.Sprintf("<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [%d 0 R] >>", descFontID))},
		pdfObject{id: descFontID, body: []byte(fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 4 >> /FontDescriptor %d 0 R /DW 1000 >>", fontDescriptorID))},
		pdfObject{id: fontDescriptorID, body: []byte("<< /Type /FontDescriptor /FontName /STSong-Light /Flags 4 /FontBBox [-25 -254 1000 880] /ItalicAngle 0 /Ascent 880 /Descent -120 /CapHeight 880 /StemV 80 >>")},
	)

	sortObjects(objects)
	return buildPDF(objects)
}

func buildLines(resp astrology.NatalChartResponse) []string {
	input := resp.Meta.Input
	lines := []string{
		"Astrolabe Report",
		"",
		fmt.Sprintf("Report ID: %s", fallback(resp.ReportID, "-")),
		fmt.Sprintf("Generated At: %s", resp.Meta.GeneratedAt.Format(time.RFC3339)),
		fmt.Sprintf("Birth Date: %s", fallback(input.BirthDate, "-")),
		fmt.Sprintf("Birth Time: %s", fallback(input.BirthTime, "-")),
		fmt.Sprintf("Location: %s / %s", fallback(input.BirthCity, "-"), fallback(input.BirthCountry, "-")),
		fmt.Sprintf("Timezone: %s", fallback(input.Timezone, "-")),
		fmt.Sprintf("Confidence: %.2f", resp.Meta.Confidence),
		"",
		"Summary",
	}

	lines = append(lines, wrapText(resp.Reading.Summary, 28)...)
	lines = append(lines,
		"",
		"Love",
	)
	lines = append(lines, wrapText(resp.Reading.Love, 28)...)
	lines = append(lines,
		"",
		"Career",
	)
	lines = append(lines, wrapText(resp.Reading.Career, 28)...)
	lines = append(lines,
		"",
		"Money",
	)
	lines = append(lines, wrapText(resp.Reading.Money, 28)...)
	lines = append(lines,
		"",
		"Family",
	)
	lines = append(lines, wrapText(resp.Reading.Family, 28)...)
	lines = append(lines,
		"",
		"Growth",
	)
	lines = append(lines, wrapText(resp.Reading.Growth, 28)...)
	lines = append(lines,
		"",
		"Action",
	)
	lines = append(lines, wrapText(resp.Reading.Action, 28)...)

	if resp.Reading.Disclaimer != "" {
		lines = append(lines, "", "Disclaimer")
		lines = append(lines, wrapText(resp.Reading.Disclaimer, 28)...)
	}

	return lines
}

func buildPageContent(lines []string) string {
	var b strings.Builder
	b.WriteString("BT\n")
	b.WriteString("/F1 12 Tf\n")
	b.WriteString(fmt.Sprintf("1 0 0 1 %d %d Tm\n", leftMargin, topStart))

	for i, line := range lines {
		if i > 0 {
			b.WriteString(fmt.Sprintf("0 -%d Td\n", lineHeight))
		}
		b.WriteString(encodePDFText(line))
		b.WriteString(" Tj\n")
	}

	b.WriteString("ET")
	return b.String()
}

func encodePDFText(text string) string {
	encoded := utf16.Encode([]rune(strings.TrimSpace(text)))
	if len(encoded) == 0 {
		return "<FEFF>"
	}

	buf := bytes.NewBufferString("feff")
	tmp := make([]byte, 2)
	for _, r := range encoded {
		tmp[0] = byte(r >> 8)
		tmp[1] = byte(r)
		buf.WriteString(hex.EncodeToString(tmp))
	}

	return "<" + strings.ToUpper(buf.String()) + ">"
}

func wrapText(text string, width int) []string {
	clean := strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if clean == "" {
		return []string{"-"}
	}

	parts := strings.Split(clean, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		runes := []rune(strings.TrimSpace(part))
		if len(runes) == 0 {
			lines = append(lines, "")
			continue
		}
		for len(runes) > width {
			lines = append(lines, string(runes[:width]))
			runes = runes[width:]
		}
		lines = append(lines, string(runes))
	}
	return lines
}

func chunkLines(lines []string, size int) [][]string {
	if size <= 0 {
		return [][]string{lines}
	}

	out := make([][]string, 0, (len(lines)+size-1)/size)
	for start := 0; start < len(lines); start += size {
		end := start + size
		if end > len(lines) {
			end = len(lines)
		}
		out = append(out, lines[start:end])
	}
	return out
}

func joinObjectRefs(ids []int) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d 0 R", id))
	}
	return strings.Join(parts, " ")
}

func sortObjects(objects []pdfObject) {
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].id < objects[j].id
	})
}

func buildPDF(objects []pdfObject) []byte {
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	offsets := make([]int, len(objects)+1)
	for _, object := range objects {
		offsets[object.id] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", object.id)
		out.Write(object.body)
		out.WriteString("\nendobj\n")
	}

	xrefStart := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(offsets))
	out.WriteString("0000000000 65535 f \n")
	for id := 1; id < len(offsets); id++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[id])
	}

	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(offsets), xrefStart)
	return out.Bytes()
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}
