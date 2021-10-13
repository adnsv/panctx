package pandoc

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// see https://hackage.haskell.org/package/pandoc-types-1.22/docs/Text-Pandoc-Definition.html

type Document struct {
	PandocApiVersion json.RawMessage        `json:"pandoc-api-version"`
	Meta             map[string]interface{} `json:"meta"`
	Blocks           []interface{}          `json:"blocks"`
}

type TC struct {
	T string      `json:"t"`
	C interface{} `json:"c"`
}

func NewDocument(buf []byte) (*Document, error) {
	doc := &Document{}
	err := json.Unmarshal(buf, doc)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (d *Document) ParseMeta() map[string]string {
	ret := map[string]string{}
	for k, r := range d.Meta {
		tc, err := loadTC(r)
		if err != nil || tc.T != "MetaInlines" {
			continue
		}
		ss, err := loadInlineSlice(tc.C)
		if err != nil {
			continue
		}
		buf := bytes.Buffer{}
		for _, l := range ss {
			switch l := l.(type) {
			case *Space:
				buf.WriteString(" ")
			case *SoftBreak:
				buf.WriteString("\n")
			case *LineBreak:
				buf.WriteString("\n")
			case *Str:
				buf.WriteString(l.Text)
			case *RawInline:
				buf.WriteString(l.Text)
			}
		}

		ret[k] = buf.String()
	}
	return ret
}

func loadInt(raw interface{}) (i int, e error) {
	f, ok := raw.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid int")
	}
	return int(f), nil
}

func loadString(raw interface{}) (s string, e error) {
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("invalid string")
	}
	return
}

func loadTC(raw interface{}) (tc *TC, e error) {
	ii, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid (t,c)")
	}
	tc = &TC{}
	tc.T, e = loadString(ii["t"])
	if e != nil {
		return nil, fmt.Errorf("(t,c) > %s", e)
	}
	tc.C = ii["c"]
	return
}

func loadTString(raw interface{}) (s string, e error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid string")
	}
	s, e = loadString(m["t"])
	return
}

func loadStringSlice(raw interface{}) (ss []string, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid string[] content")
	}
	for idx, i := range ii {
		s, e := loadString(i)
		if e != nil {
			return nil, fmt.Errorf("string[%d] > %s", idx, e)
		}
		ss = append(ss, s)
	}
	return
}

func loadKeyVal(raw interface{}) (kv *KeyVal, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid (k,v)")
	}
	kv = &KeyVal{}
	kv.Key, e = loadString(ii[0])
	if e != nil {
		return nil, fmt.Errorf("(k,v), in key > %s", e)
	}
	if e == nil {
		kv.Val, e = loadString(ii[1])
	}
	if e != nil {
		return nil, fmt.Errorf("(k,v), in value > %s", e)
	}
	return
}

func loadKeyValSlice(raw interface{}) (kvs []*KeyVal, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid KeyVal[]")
	}
	for idx, i := range ii {
		kv, e := loadKeyVal(i)
		if e != nil {
			return nil, fmt.Errorf("KeyVal[%d] > %s", idx, e)
		}
		kvs = append(kvs, kv)
	}
	return
}

func loadAttr(raw interface{}) (a Attr, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 3 {
		return Attr{}, fmt.Errorf("invalid Attr")
	}
	a.Identifier, e = loadString(ii[0])
	if e != nil {
		return Attr{}, fmt.Errorf("Attr.Identifier > %s", e)
	}
	a.Classes, e = loadStringSlice(ii[1])
	if e != nil {
		return Attr{}, fmt.Errorf("Attr.Classes > %s", e)
	}
	a.KeyVals, e = loadKeyValSlice(ii[2])
	if e != nil {
		return Attr{}, fmt.Errorf("Attr.KeyVals > %s", e)
	}
	return
}

func loadInlineSliceSlice(raw interface{}) (lll []InlineList, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Inline[][]")
	}
	for idx, i := range ii {
		ll, e := loadInlineSlice(i)
		if e != nil {
			return nil, fmt.Errorf("KeyVal[%d] > %s", idx, e)
		}
		lll = append(lll, ll)
	}
	return
}

func loadHeader(raw interface{}) (h *Header, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 3 {
		return nil, fmt.Errorf("invalid Header")
	}
	h = &Header{}

	h.Level, e = loadInt(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Header.Level > %s", e)
	}

	h.Attr, e = loadAttr(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Header.Attr > %s", e)
	}

	h.Inlines, e = loadInlineSlice(ii[2])
	if e != nil {
		return nil, fmt.Errorf("Header.Inlines > %s", e)
	}
	return
}

func loadPlain(raw interface{}) (p *Plain, e error) {
	p = &Plain{}
	p.Inlines, e = loadInlineSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("Plain > %s", e)
	}
	return
}

func loadPara(raw interface{}) (p *Para, e error) {
	p = &Para{}
	p.Inlines, e = loadInlineSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("Para > %s", e)
	}
	return
}

func loadLineBlock(raw interface{}) (ll *LineBlock, e error) {
	ll = &LineBlock{}
	ll.Lines, e = loadInlineSliceSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("LineBlock > %s", e)
	}
	return
}

func loadCodeBlock(raw interface{}) (cb *CodeBlock, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid CodeBlock[]")
	}
	cb = &CodeBlock{}
	cb.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("CodeBlock.Attr > %s", e)
	}
	cb.Text, e = loadString(ii[1])
	if e != nil {
		return nil, fmt.Errorf("LineBlock.Text > %s", e)
	}
	return
}

func loadRawBlock(raw interface{}) (rb *RawBlock, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid RawBlock[]")
	}
	rb = &RawBlock{}
	rb.Format, e = loadString(ii[0])
	if e != nil {
		return nil, fmt.Errorf("RawBlock.Format > %s", e)
	}
	rb.Text, e = loadString(ii[1])
	if e != nil {
		return nil, fmt.Errorf("RawBlock.Text > %s", e)
	}
	return
}

func loadBlockQuote(raw interface{}) (bq *BlockQuote, e error) {
	bq = &BlockQuote{}
	bq.Blocks, e = loadBlockSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("BlockQuote > %s", e)
	}
	return
}

func loadBlockSlice(raw interface{}) (bb BlockList, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid block[]")
	}
	for idx, i := range ii {
		tc, e := loadTC(i)
		if e != nil {
			return nil, fmt.Errorf("block[%d] > %s", idx, e)
		}
		b, e := loadBlock(tc)
		if e != nil {
			return nil, fmt.Errorf("block[%d] > %s", idx, e)
		}
		if b != nil {
			bb = append(bb, b)
		}
	}
	return
}

func loadBlockSliceSlice(raw interface{}) (bbb []BlockList, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid block[][] content")
	}
	for idx, i := range ii {
		bb, e := loadBlockSlice(i)
		if e != nil {
			return nil, fmt.Errorf("block[%d][] > %s", idx, e)
		}
		bbb = append(bbb, bb)
	}
	return
}

func loadBulletList(raw interface{}) (bl *BulletList, e error) {
	bl = &BulletList{}
	bl.Items, e = loadBlockSliceSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("BulletList > %s", e)
	}
	return
}

func loadDefinitionItem(raw interface{}) (d *DefinitionItem, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid DefinitionItem")
	}
	d = &DefinitionItem{}
	d.Term, e = loadInlineSlice(ii[0])
	if e != nil {
		return nil, fmt.Errorf("term > %s", e)
	}
	d.Definitions, e = loadBlockSliceSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("definitions > %s", e)
	}
	return
}

func loadDefinitionList(raw interface{}) (dl *DefinitionList, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid DefinitionList")
	}
	dl = &DefinitionList{}
	for idx, i := range ii {
		d, e := loadDefinitionItem(i)
		if e != nil {
			return nil, fmt.Errorf("DefinitionList[%d] > %s", idx, e)
		}
		dl.Items = append(dl.Items, d)
	}
	return
}

func loadOrderedList(raw interface{}) (ol *OrderedList, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid OrderedList")
	}

	ol = &OrderedList{}
	// list attributes
	attrs, ok := ii[0].([]interface{})
	if !ok || len(attrs) != 3 {
		return nil, fmt.Errorf("invalid OrderedList.ListAttributes")
	}
	ol.StartNumber, e = loadInt(attrs[0])
	if e != nil {
		return nil, fmt.Errorf("invalid OrderedList.StartNumber")
	}
	ol.NumberStyle, e = loadTString(attrs[1])
	if e != nil {
		return nil, fmt.Errorf("invalid OrderedList.NumberStyle")
	}
	ol.NumberDelim, e = loadTString(attrs[2])
	if e != nil {
		return nil, fmt.Errorf("invalid OrderedList.NumberDelim")
	}

	// items
	ol.Items, e = loadBlockSliceSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("OrderedList.Items > %s", e)
	}
	return
}

func loadDiv(raw interface{}) (d *Div, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid DefinitionList")
	}
	d = &Div{}
	d.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Div.Attr > %s", e)
	}
	d.Blocks, e = loadBlockSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Div.Blocks > %s", e)
	}
	return
}

func loadCell(raw interface{}) (c *Cell, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 5 {
		return nil, fmt.Errorf("invalid Cell")
	}
	c = &Cell{}
	c.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("invalid Cell.Attr %w", e)
	}
	c.Alignment, e = loadTString(ii[1])
	if e != nil {
		return nil, fmt.Errorf("invalid Cell.Align %w", e)
	}
	c.RowSpan, e = loadInt(ii[2])
	if e != nil {
		return nil, fmt.Errorf("invalid Cell.RowSpan %w", e)
	}
	c.ColSpan, e = loadInt(ii[3])
	if e != nil {
		return nil, fmt.Errorf("invalid Cell.ColSpan %w", e)
	}
	c.Blocks, e = loadBlockSlice(ii[4])
	if e != nil {
		return nil, fmt.Errorf("invalid Cell.Blocks %w", e)
	}
	return
}

func loadCellSlice(raw interface{}) (cc []*Cell, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Cell[]")
	}
	for idx, i := range ii {
		// each cell is an array of 5 fields
		c, e := loadCell(i)
		if e != nil {
			return nil, fmt.Errorf("invalid Cell[%d] > %s", idx, e)
		}
		cc = append(cc, c)
	}
	return
}

func loadRowSlice(raw interface{}) (rr []*Row, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Row[]")
	}
	for idx, i := range ii {
		ss, ok := i.([]interface{})
		if !ok || len(ss) != 2 {
			return nil, fmt.Errorf("invalid Row[%d]", idx)
		}
		r := &Row{}
		rr = append(rr, r)
		r.Attr, e = loadAttr(ss[0])
		if e != nil {
			return nil, fmt.Errorf("invalid Row[%d].Attr > %s", idx, e)
		}
		r.Cells, e = loadCellSlice(ss[1])
		if e != nil {
			return nil, fmt.Errorf("Row[%d] > %s", idx, e)
		}
	}
	return
}

func loadColSpecs(raw interface{}) (cc []*ColSpec, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid ColSpec[] %w", e)
	}
	for idx, i := range ii {
		t, ok := i.([]interface{})
		if !ok || len(t) != 2 {
			return nil, fmt.Errorf("Table invalid ColSpec[%d] > %s", idx, e)
		}
		m, ok := t[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Table invalid ColSpec[%d] > %s", idx, e)
		}
		c := &ColSpec{}
		cc = append(cc, c)
		c.Alignment, e = loadString(m["t"])
		if e != nil {
			return nil, fmt.Errorf("Table invalid ColSpec[%d].Align > %s", idx, e)
		}
		// ignore width until we figure out if and how we deal with it
	}
	return
}

func loadTableHead(raw interface{}) (t TableHeadOrFoot, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return TableHeadOrFoot{}, fmt.Errorf(".TableHead > invalid content")
	}
	t.Attr, e = loadAttr(ii[0])
	if e != nil {
		return TableHeadOrFoot{}, fmt.Errorf(".TableHead.Attr > %s", e)
	}
	t.Rows, e = loadRowSlice(ii[1])
	if e != nil {
		return TableHeadOrFoot{}, fmt.Errorf(".TableHead > %s", e)
	}
	return
}

func loadTableFoot(raw interface{}) (t TableHeadOrFoot, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return TableHeadOrFoot{}, fmt.Errorf(".TableFoot > invalid content")
	}
	t.Attr, e = loadAttr(ii[0])
	if e != nil {
		return TableHeadOrFoot{}, fmt.Errorf(".TableFoot.Attr > %s", e)
	}
	t.Rows, e = loadRowSlice(ii[1])
	if e != nil {
		return TableHeadOrFoot{}, fmt.Errorf(".TableHead > %s", e)
	}
	return
}

func loadTableBody(raw interface{}) (b *TableBody, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 4 {
		return nil, fmt.Errorf(".TableBody > invalid content")
	}
	b = &TableBody{}
	b.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf(".TableBody > %s", e)
	}
	b.RowHeadColumns, e = loadInt(ii[1])
	if e != nil {
		return nil, fmt.Errorf(".TableBody.RowHeadColumns > %s", e)
	}
	b.Rows1, e = loadRowSlice(ii[2])
	if e != nil {
		return nil, fmt.Errorf(".TableBody > %s", e)
	}
	b.Rows2, e = loadRowSlice(ii[3])
	if e != nil {
		return nil, fmt.Errorf(".TableBody > %s", e)
	}
	return
}

func loadTable(raw interface{}) (t *Table, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 6 {
		return nil, fmt.Errorf("invalid Table")
	}
	t = &Table{}
	t.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf(".Attr > %s", e)
	}

	// Caption
	cc, ok := ii[1].([]interface{})
	if !ok || len(cc) != 2 {
		return nil, fmt.Errorf(".Caption > %s", e)
	}
	if cc[0] != nil {
		t.ShortCaption, e = loadInlineSlice(cc[0])
		if e != nil {
			return nil, fmt.Errorf(".ShortCaption > %s", e)
		}
	}
	t.Caption, e = loadBlockSlice(cc[1])
	if e != nil {
		return nil, fmt.Errorf(".Caption > %s", e)
	}

	t.ColSpecs, e = loadColSpecs(ii[2])
	if e != nil {
		return nil, e
	}

	t.Head, e = loadTableHead(ii[3])
	if e != nil {
		return nil, e
	}

	bb, ok := ii[4].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Table.Body[]")
	}
	for idx, bi := range bb {
		b, e := loadTableBody(bi)
		if e != nil {
			return nil, fmt.Errorf("Table.Body[%d] > %s", idx, e)
		}
		t.Bodies = append(t.Bodies, b)
	}

	t.Foot, e = loadTableFoot(ii[5])
	if e != nil {
		return nil, fmt.Errorf("Table > %s", e)
	}

	return
}

func loadBlock(tc *TC) (b Block, e error) {
	switch tc.T {
	case "Plain":
		b, e = loadPlain(tc.C)
	case "Para":
		b, e = loadPara(tc.C)
	case "LineBlock":
		b, e = loadLineBlock(tc.C)
	case "CodeBlock":
		b, e = loadCodeBlock(tc.C)
	case "RawBlock":
		b, e = loadRawBlock(tc.C)
	case "BlockQuote":
		b, e = loadBlockQuote(tc.C)
	case "OrderedList":
		b, e = loadOrderedList(tc.C)
	case "BulletList":
		b, e = loadBulletList(tc.C)
	case "DefinitionList":
		b, e = loadDefinitionList(tc.C)
	case "Header":
		b, e = loadHeader(tc.C)
	case "HorizontalRule":
		b, e = &HorizontalRule{}, nil
	case "Table":
		b, e = loadTable(tc.C)
	case "Div":
		b, e = loadDiv(tc.C)
	case "Null":
		// ignore
	default:
		return nil, fmt.Errorf("unsupported element type %s", tc.T)
	}
	if e != nil {
		e = fmt.Errorf("%s > %s", tc.T, e)
	}
	return
}

func loadFormatted(raw interface{}, f InlineFmt) (s *Formatted, e error) {
	s = &Formatted{Fmt: f}
	s.Content, e = loadInlineSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("%s > %s", f, e)
	}
	return
}

func loadQuoted(raw interface{}) (q *Quoted, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid Quoted")
	}
	q = &Quoted{}
	q.QuoteType, e = loadTString(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Quoted.QuoteType > %s", e)
	}
	q.Content, e = loadInlineSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Quoted > %s", e)
	}
	return
}

func loadCite(raw interface{}) (c *Cite, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid Cite")
	}
	c = &Cite{}

	cc, ok := ii[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Cite.Citation")
	}

	for k, v := range cc {
		switch k {
		case "citationId":
			c.Id, e = loadString(v)
		case "citationPrefix":
			c.Prefix, e = loadInlineSlice(cc["citationPrefix"])
		case "citationSuffix":
			c.Suffix, e = loadInlineSlice(cc["citationSuffix"])
		case "citationMode":
			c.Mode, e = loadString(cc["citationMode"])
		case "citationNoteNum":
			c.NoteNum, e = loadInt(cc["citationNoteNum"])
		case "citationHash":
			c.Hash, e = loadInt(cc["citationHash"])
		}
		if e != nil {
			return nil, fmt.Errorf("invalid Cite.%s > %s", k, e)
		}
	}

	c.Content, e = loadInlineSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Cite > %s", e)
	}
	return
}

func loadCode(raw interface{}) (c *Code, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid Code")
	}
	c = &Code{}
	c.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Code.Attr > %s", e)
	}
	c.Text, e = loadString(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Code.Content > %s", e)
	}
	return
}

func loadMath(raw interface{}) (m *Math, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid Math")
	}
	m = &Math{}
	m.Type, e = loadTString(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Math.Type > %s", e)
	}
	m.Text, e = loadString(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Math.Content > %s", e)
	}
	return
}

func loadRawInline(raw interface{}) (r *RawInline, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return nil, fmt.Errorf("invalid RawInline")
	}
	r = &RawInline{}
	r.Format, e = loadString(ii[0])
	if e != nil {
		return nil, fmt.Errorf("RawInline.Format > %s", e)
	}
	r.Text, e = loadString(ii[1])
	if e != nil {
		return nil, fmt.Errorf("RawInline.Content > %s", e)
	}
	return
}

func loadTarget(raw interface{}) (t Target, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 2 {
		return Target{}, fmt.Errorf("invalid RawInline")
	}
	t.URL, e = loadString(ii[0])
	if e != nil {
		return Target{}, fmt.Errorf("RawInline.Format > %s", e)
	}
	t.Title, e = loadString(ii[1])
	if e != nil {
		return Target{}, fmt.Errorf("RawInline.Content > %s", e)
	}
	return
}

func loadLink(raw interface{}) (l *Link, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 3 {
		return nil, fmt.Errorf("invalid Link")
	}
	l = &Link{}
	l.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Link.Attr > %s", e)
	}
	l.Content, e = loadInlineSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Link.Content > %s", e)
	}
	l.Target, e = loadTarget(ii[2])
	if e != nil {
		return nil, fmt.Errorf("Link.Target > %s", e)
	}
	return
}

func loadImage(raw interface{}) (i *Image, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 3 {
		return nil, fmt.Errorf("invalid Image")
	}
	i = &Image{}
	i.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Image.Attr > %s", e)
	}
	i.Content, e = loadInlineSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Image.Content > %s", e)
	}
	i.Target, e = loadTarget(ii[2])
	if e != nil {
		return nil, fmt.Errorf("Image.Target > %s", e)
	}
	return
}

func loadNote(raw interface{}) (n *Note, e error) {
	n = &Note{}
	n.Blocks, e = loadBlockSlice(raw)
	if e != nil {
		return nil, fmt.Errorf("Note > %s", e)
	}
	return
}

func loadSpan(raw interface{}) (s *Span, e error) {
	ii, ok := raw.([]interface{})
	if !ok || len(ii) != 3 {
		return nil, fmt.Errorf("invalid Image")
	}
	s = &Span{}
	s.Attr, e = loadAttr(ii[0])
	if e != nil {
		return nil, fmt.Errorf("Span.Attr > %s", e)
	}
	s.Content, e = loadInlineSlice(ii[1])
	if e != nil {
		return nil, fmt.Errorf("Span.Content > %s", e)
	}
	return
}

func loadInline(tc *TC) (l Inline, e error) {
	switch tc.T {
	case "Str":
		t := &Str{}
		t.Text, e = loadString(tc.C)
		l = t
	case "Emph":
		l, e = loadFormatted(tc.C, Emph)
	case "Underline":
		l, e = loadFormatted(tc.C, Underline)
	case "Strong":
		l, e = loadFormatted(tc.C, Strong)
	case "Strikeout":
		l, e = loadFormatted(tc.C, Strikeout)
	case "Superscript":
		l, e = loadFormatted(tc.C, Superscript)
	case "Subscript":
		l, e = loadFormatted(tc.C, Subscript)
	case "SmallCaps":
		l, e = loadFormatted(tc.C, SmallCaps)
	case "Quoted":
		l, e = loadQuoted(tc.C)
	case "Cite":
		l, e = loadCite(tc.C)
	case "Code":
		l, e = loadCode(tc.C)
	case "Space":
		l, e = &Space{}, nil
	case "SoftBreak":
		l, e = &SoftBreak{}, nil
	case "LineBreak":
		l, e = &LineBreak{}, nil
	case "Math":
		l, e = loadMath(tc.C)
	case "RawInline":
		l, e = loadRawInline(tc.C)
	case "Link":
		l, e = loadLink(tc.C)
	case "Image":
		l, e = loadImage(tc.C)
	case "Note":
		l, e = loadNote(tc.C)
	case "Span":
		l, e = loadSpan(tc.C)
	default:
		return nil, fmt.Errorf("unsupported inline type %s", tc.T)
	}
	return
}

func loadInlineSlice(raw interface{}) (ll InlineList, e error) {
	ii, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Inline[] content")
	}

	for _, i := range ii {
		tc, e := loadTC(i)
		if e != nil {
			return nil, e
		}
		l, e := loadInline(tc)
		if e != nil {
			return nil, e
		}
		if l != nil {
			ll = append(ll, l)
		}
	}
	return
}

func (d *Document) Flow() (bb BlockList, e error) {
	bb, e = loadBlockSlice(d.Blocks)
	return
}
