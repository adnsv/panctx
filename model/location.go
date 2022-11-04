package model

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type byteOffset = int

type SourceLocation struct {
	LineNumber   int // line number
	ColumnNumber int // column number

	LineOffset     byteOffset // byte offset of line relative to buffer start
	LocationOffset byteOffset // byte offset of location relative to LineOffset
}

func CalcSourceLocation(buf string, byteoffset int) *SourceLocation {
	cur, end := 0, len(buf)
	if strings.HasPrefix(buf, "\xef\xbb\xbf") {
		cur = 3
	}
	if byteoffset > end {
		byteoffset = end
	}

	loc := SourceLocation{
		LineNumber: 1,
		LineOffset: cur,
	}

	for cur < byteoffset {
		c := buf[cur]
		cur++
		if c == '\n' {
			loc.LineNumber++
			loc.LineOffset = cur
		} else if c == '\r' {
			if cur < byteoffset && buf[cur] == '\n' {
				cur++
			}
			loc.LineNumber++
			loc.LineOffset = cur
		}
	}
	loc.LocationOffset = byteoffset - loc.LineOffset
	loc.ColumnNumber = 1 + utf8.RuneCountInString(buf[loc.LineOffset:byteoffset])

	return &loc
}

func (sl *SourceLocation) Valid() bool {
	return sl.LineNumber > 0 && sl.ColumnNumber > 0
}

func (sl *SourceLocation) String() string {
	return fmt.Sprintf("%d:%d", sl.LineNumber, sl.ColumnNumber)
}

func (sl *SourceLocation) ExtractLine(buf string) string {
	buf = buf[sl.LineOffset:]
	p := strings.IndexAny(buf, "\r\n")
	if p > 0 {
		return buf[:p]
	}
	return buf[:0]
}
