package latex2mathml

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/neruyzo/etree"
)

var COLUMNS_ALINGNMENT_MAP = map[string]string{"r": "right", "l": "left", "c": "center"}
var SPACE_LIST = []string{`\ `, "~", NOBREAKSPACE, SPACE}
var SUB_LIST = []string{DETERMINANT, GCD, INTOP, INJLIM, LIMINF, LIMSUP, PR, PROJLIM}

// BIG_OPERATORS are the large n-ary operators whose sub/super scripts auto-stack
// as under/over in DISPLAY style (matching KaTeX/LaTeX \displaylimits). Their
// unpatched behaviour was side-by-side <msubsup>/<msub>; the tag-selection in
// convertCommand promotes them to <munderover>/<munder>/<mover> when
// isDisplayMode is true. Integrals (\int, \oint, \iint, …) are DELIBERATELY
// EXCLUDED — by TeX convention they keep side limits unless an explicit \limits
// is given (that path is the LIMITS-modifier branch, untouched here). The
// \lim-family is carried separately via the LIMIT set (commands.go).
var BIG_OPERATORS = []string{
	`\sum`, `\prod`, `\coprod`,
	`\bigcup`, `\bigcap`, `\bigsqcup`, `\biguplus`,
	`\bigvee`, `\bigwedge`,
	`\bigoplus`, `\bigotimes`, `\bigodot`,
}

var OPERATORS = []string{
	"+",
	"-",
	"*",
	"/",
	"(",
	")",
	"=",
	",",
	"?",
	"[",
	"]",
	"|",
	`\|`,
	"!",
	`\{`,
	`\}`,
	`>`,
	`<`,
	`.`,
	`\bigotimes`,
	`\centerdot`,
	`\dots`,
	`\dotsc`,
	`\dotso`,
	`\gt`,
	`\ldotp`,
	`\lt`,
	`\lvert`,
	`\lVert`,
	`\lvertneqq`,
	`\ngeqq`,
	`\omicron`,
	`\rvert`,
	`\rVert`,
	`\S`,
	`\smallfrown`,
	`\smallint`,
	`\smallsmile`,
	`\surd`,
	`\varsubsetneqq`,
	`\varsupsetneqq`,
}

type Mode int64

const (
	TextMode Mode = 0
	MathMode Mode = 1
)

type Moded struct {
	Value string
	Mode  Mode
}

var MATH_MODE_PATTERN = regexp.MustCompile(`\\\$|\$|\\?[^\\$]+`)
var NUMBER_PATTERN = regexp.MustCompile(`\d+(.\d+)?`)

func Convert(latex string, xmlns string, display string, indent int) string {
	InitializeCommands()
	ParseSymbol()
	doc := etree.NewDocument()
	math := doc.CreateElement("math")
	math.CreateAttr("xmlns", xmlns)
	math.CreateAttr("display", display)
	row := math.CreateElement("mrow")
	nodes, _ := Walk(latex)
	convertGroup(nodes, row, map[string]string{})
	if indent != 0 {
		doc.Indent(indent)
	}
	doc.WriteSettings.NoEscape = true
	mathml, _ := doc.WriteToString()
	return mathml
}

func convertMatrix(nodes []Node, parent *etree.Element, command string, alignment string) {
	var row *etree.Element
	var cell *etree.Element

	var colIndex = 0
	var maxColSize = 0
	var colAlignment *string

	var rowIndex = 0
	var rowLines = []string{}

	var indexes = []bool{}

	for _, node := range nodes {

		if row == nil {
			row = parent.CreateElement("mtr")
		}

		if cell == nil {
			colAlignment, colIndex = getColumnAlignment(&alignment, colAlignment, colIndex)
			cell = makeMatrixCell(row, colAlignment)
		}

		if node.Token == BRACES {
			convertGroup([]Node{node}, cell, map[string]string{})
		} else if node.Token == "&" {
			setCellAlignment(cell, indexes)
			indexes = []bool{}
			colAlignment, colIndex = getColumnAlignment(&alignment, colAlignment, colIndex)
			cell = makeMatrixCell(row, colAlignment)
			if (command == SPLIT || command == ALIGN || command == ALIGNED) && colIndex%2 == 0 {
				cell.CreateElement("mi")
			}
		} else if node.Token == DOUBLEBACKSLASH || node.Token == CARRIAGE_RETURN {
			setCellAlignment(cell, indexes)
			indexes = []bool{}
			rowIndex = rowIndex + 1
			if colIndex > maxColSize {
				maxColSize = colIndex
			}
			colIndex = 0
			colAlignment, colIndex = getColumnAlignment(&alignment, colAlignment, colIndex)
			row = parent.CreateElement("mtr")
			cell = makeMatrixCell(row, colAlignment)
		} else if node.Token == HLINE {
			rowLines = append(rowLines, "solid")
		} else if node.Token == HDASHLINE {
			rowLines = append(rowLines, "dashed")
		} else if node.Token == HFIL {
			indexes = append(indexes, true)
		} else {
			if rowIndex > len(rowLines) {
				rowLines = append(rowLines, "none")
			}
			indexes = append(indexes, false)
			convertGroup([]Node{node}, cell, map[string]string{})
		}
	}

	if colIndex > maxColSize {
		maxColSize = colIndex
	}

	if slices.Contains(rowLines, "solid") {
		parent.CreateAttr("rowlines", strings.Join(rowLines, " "))
	}

	if row != nil && cell != nil && len(cell.ChildElements()) == 0 && cell.Text() == "" {
		children := parent.ChildElements()
		parent.RemoveChildAt(len(children) - 1)
	}

	if maxColSize > 0 && (command == ALIGN || command == ALIGNED) {
		multiplier := maxColSize / 2
		spacing := "0em 2em"
		for i := 0; i < multiplier-1; i++ {
			spacing = spacing + " 0em 2em"
		}
		parent.CreateAttr("columnspacing", spacing)
	}
}

func setCellAlignment(cell *etree.Element, indexes []bool) {
	if slices.Contains(indexes, true) {
		if indexes[0] && !indexes[len(indexes)-1] {
			cell.CreateAttr("columnalign", "right")
		} else if !indexes[0] && indexes[len(indexes)-1] {
			cell.CreateAttr("columnalign", "left")
		}
	}
}

func getColumnAlignment(alignment *string, columnAlignment *string, columnIndex int) (*string, int) {
	if alignment != nil && *alignment != "" {
		var align = *alignment
		var alignmentIndex = align[columnIndex%len(align)]
		columnAlign, exist := COLUMNS_ALINGNMENT_MAP[string(alignmentIndex)]

		if exist {
			columnAlignment = &columnAlign
		}

		columnIndex = columnIndex + 1
	}

	return columnAlignment, columnIndex
}

func makeMatrixCell(row *etree.Element, columnAlignment *string) *etree.Element {
	var element = etree.NewElement("mtd")

	if columnAlignment != nil {
		element.CreateAttr("columnalign", *columnAlignment)
	}
	row.AddChild(element)

	return element
}

func convertGroup(nodes []Node, parent *etree.Element, font map[string]string) {
	for _, node := range nodes {
		token := node.Token

		if _, exist := MSTYLE_SIZES[token]; exist {
			node := Node{Token: token, Children: nodes}
			convertCommand(node, parent, font)
		} else if _, exist := STYLES[token]; exist {
			node := Node{Token: token, Children: nodes}
			convertCommand(node, parent, font)
		} else if _, exist := CONVERSION_MAP[token]; exist || token == MOD || token == PMOD {
			convertCommand(node, parent, font)
		} else if _, exist := LOCAL_FONTS[token]; exist && node.Children != nil {
			convertGroup(node.Children, parent, LOCAL_FONTS[token])
		} else if strings.HasPrefix(token, MATH) && node.Children != nil {
			convertGroup(node.Children, parent, font)
		} else if _, exist := GLOBAL_FONTS[token]; exist {
			font, _ = GLOBAL_FONTS[token]
		} else if node.Children == nil {
			convertSymbol(node, parent, font)
		} else {
			row := parent.CreateElement("mrow")
			addAttributes(row, node.Attributes)
			convertGroup(node.Children, row, font)
		}
	}

}

func getAlignmentAndColumnLine(alignment *string) (*string, *string) {
	if alignment == nil {
		return nil, nil
	}

	if !strings.Contains(*alignment, "|") {
		return alignment, nil
	}

	var ajusted = ""
	var columnLines = []string{}

	for _, char := range *alignment {
		if char == '|' {
			columnLines = append(columnLines, "solid")
		} else {
			ajusted = ajusted + string(char)
		}

		if len(ajusted)-len(columnLines) == 2 {
			columnLines = append(columnLines, "none")
		}
	}

	var column = strings.Join(columnLines, " ")

	return &ajusted, &column
}

func SeparateByMode(text string) []Moded {
	var value = ""
	var isMathMode = false

	var moded = []Moded{}

	for _, match := range MATH_MODE_PATTERN.FindAllString(text, 0) {
		if match == "$" {
			if isMathMode {
				moded = append(moded, Moded{Value: value, Mode: MathMode})
			} else {
				moded = append(moded, Moded{Value: value, Mode: TextMode})
			}
			value = ""
			isMathMode = !isMathMode
		} else {
			value = value + match
		}
	}

	if len(value) > 0 {
		if isMathMode {
			moded = append(moded, Moded{Value: value, Mode: MathMode})
		} else {
			moded = append(moded, Moded{Value: value, Mode: TextMode})
		}
	}

	return moded
}

func convertCommand(node Node, parent *etree.Element, font map[string]string) {
	command := node.Token
	modifier := node.Modifier

	if command == SUBSTACK || command == SMALLMATRIX {
		parent = parent.CreateElement("mstyle")
		parent.CreateAttr("scriptlevel", "1")
	} else if command == CASES {
		parent = parent.CreateElement("mrow")
		lbrace := parent.CreateElement("mo")
		lbrace.CreateAttr("stretchy", "true")
		lbrace.CreateAttr("fence", "true")
		lbrace.CreateAttr("form", "prefix")
		code, _ := ConvertSymbol(LBRACE)
		lbrace.SetText("&#x" + code + ";")
	} else if command == DBINOM || command == DFRAC {
		parent = parent.CreateElement("mstyle")
		parent.CreateAttr("displaystyle", "true")
		parent.CreateAttr("scriptlevel", "0")
	} else if command == HPHANTOM {
		parent = parent.CreateElement("mpadded")
		parent.CreateAttr("height", "0")
		parent.CreateAttr("depth", "0")
	} else if command == VPHANTOM {
		parent = parent.CreateElement("mpadded")
		parent.CreateAttr("width", "0")
	} else if command == TBINOM || command == HBOX || command == MBOX || command == TFRAC {
		parent = parent.CreateElement("mstyle")
		parent.CreateAttr("displaystyle", "false")
		parent.CreateAttr("scriptlevel", "0")
	} else if command == MOD || command == PMOD {
		parent = parent.CreateElement("mspace")
		parent.CreateAttr("width", "1em")
	}

	style, _ := CONVERSION_MAP[command]

	if len(node.Attributes) > 0 && node.Token != SKEW {
		for key, value := range node.Attributes {
			style.Modifiers[key] = value
		}
	}

	if command == LEFT {
		parent = parent.CreateElement("mrow")
	}

	appendPrefixElement(node, parent)

	alignment, columnLines := getAlignmentAndColumnLine(&node.Alignment)

	if columnLines != nil {
		style.Modifiers["columnlines"] = *columnLines
	}

	var tag = style.Tag

	if command == SUBSUP && len(node.Children) > 0 && node.Children[0].Token == GCD {
		tag = "munderover"
	} else if command == SUPERSCRIPT && (modifier == LIMITS || modifier == OVERBRACE) {
		tag = "mover"
	} else if command == SUBSCRIPT && (modifier == LIMITS || modifier == UNDERBRACE) {
		tag = "munder"
	} else if command == SUBSUP && (modifier == LIMITS || modifier == OVERBRACE || modifier == UNDERBRACE) {
		tag = "munderover"
	} else if (command == XLEFTARROW || command == XRIGHTARROW) && len(node.Children) == 2 {
		tag = "munderover"
	} else if len(node.Children) > 0 && isDisplayMode(parent) &&
		(slices.Contains(BIG_OPERATORS, node.Children[0].Token) || slices.Contains(LIMIT, node.Children[0].Token)) {
		// Big n-ary operators (\sum \prod \bigcup …) and the \lim-family
		// auto-stack their scripts as under/over in DISPLAY style (no explicit
		// \limits needed) — KaTeX/LaTeX \displaylimits. Open Q1 resolved to this
		// tag-switch (NOT movablelimits): the operator renders as <mi>/<mo> and
		// MathML-Core only stacks via munder*/mover*. Strictly gated on
		// SUBSUP/SUBSCRIPT/SUPERSCRIPT + displaystyle so inline `$\sum_i^n$` and
		// limitless `\sum` are unaffected.
		switch command {
		case SUBSUP:
			tag = "munderover"
		case SUBSCRIPT:
			tag = "munder"
		case SUPERSCRIPT:
			tag = "mover"
		}
	}

	element := parent.CreateElement(tag)
	addAttributes(element, style.Modifiers)

	if slices.Contains(LIMIT, command) {
		element.SetText(command[1:])
	} else if command == MOD || command == PMOD {
		element.SetText("mod")
		space := parent.CreateElement("mspace")
		space.CreateAttr("width", "0.333em")
	} else if command == BMOD {
		element.SetText("mod")
	} else if command == XLEFTARROW || command == XRIGHTARROW {
		style := element.CreateElement("mstyle")
		style.CreateAttr("scriptlevel", "0")
		arrow := style.CreateElement("mo")
		if command == XLEFTARROW {
			arrow.SetText("&#x2190;")
		} else {
			arrow.SetText("&#x2192;")
		}
	} else if node.Text != "" {
		if command == MIDDLE {
			code, _ := ConvertSymbol(node.Text)
			element.SetText("&#x" + code + ";")
		} else if command == HBOX {
			mtext := element
			for _, mode := range SeparateByMode(node.Text) {
				if mode.Mode == TextMode {
					if mtext == nil {
						mtext = parent.CreateElement(tag)
						addAttributes(mtext, style.Modifiers)
						setFont(mtext, "mtext", font)
						mtext = nil
					} else {
						row := parent.CreateElement("mrow")
						nodes, _ := Walk(mode.Value)
						convertGroup(nodes, row, map[string]string{})
					}
				}
			}
		} else {
			if command == FBOX {
				element = element.CreateElement("mtext")
			}
			element.SetText(strings.ReplaceAll(node.Text, " ", "&#x000A0;"))
			setFont(element, "mtext", font)
		}

	} else if node.Delimiter != "" && command != FRAC && command != GENFRAC {

		if node.Delimiter != "." {
			code, err := ConvertSymbol(node.Delimiter)
			if err == nil {
				element.SetText("&#x" + code + ";")
			}
		}
	}

	if node.Children != nil {
		localParent := element

		if command == LEFT || command == MOD || command == PMOD {
			localParent = parent
		}

		align := *alignment

		if slices.Contains(MATRICES, command) {

			if command == CASES {
				align = "l"
			} else if command == SPLIT || command == ALIGN || command == ALIGNED {
				align = "rl"
			}
			convertMatrix(node.Children, localParent, command, align)

		} else if command == CFRAC {

			for _, child := range node.Children {
				p := localParent.CreateElement("mstyle")
				p.CreateAttr("displaystyle", "false")
				p.CreateAttr("scriptlevel", "0")
				convertGroup([]Node{child}, p, font)
			}

		} else if command == SIDESET {

			convertGroup(node.Children[0:1], localParent, font)
			fill := localParent.CreateElement("mstyle")
			fill.CreateAttr("scriptlevel", "0")
			space := fill.CreateElement("mspace")
			space.CreateAttr("width", "-0.167em")
			convertGroup(node.Children[1:2], localParent, font)

		} else if command == SKEW {

			child := node.Children[0]
			newNode := Node{
				Token: child.Token,
				Children: []Node{
					{
						Token: BRACES,
						Children: append(
							child.Children,
							Node{Token: MKERN, Attributes: node.Attributes},
						),
					},
				},
			}
			convertGroup([]Node{newNode}, localParent, font)

		} else if command == XLEFTARROW || command == XRIGHTARROW {
			for _, child := range node.Children {
				padded := localParent.CreateElement("mpadded")
				// padded.CreateAttr("width", "0.833em")
				// padded.CreateAttr("lspace", "0.556em")
				// padded.CreateAttr("voffset", "-0.2em")
				// padded.CreateAttr("height", "-0.2em")
				convertGroup([]Node{child}, padded, font)
				space := padded.CreateElement("mspace")
				space.CreateAttr("depth", "0.25em")
			}

		} else {

			convertGroup(node.Children, localParent, font)
		}
	}

	addDiacritic(command, element)
	appendPostfixElement(node, parent)
}

func addDiacritic(command string, parent *etree.Element) {
	style, exist := DIACRITICS[command]
	if exist {
		element := etree.NewElement("mo")
		element.SetText(style.Tag)
		for key, value := range style.Modifiers {
			element.CreateAttr(key, value)
		}
		parent.AddChild(element)
	}
}

func addAttributes(element *etree.Element, attributes map[string]string) {
	for key, value := range attributes {
		element.CreateAttr(key, value)
	}
}

func convertAndAppendCommand(command string, parent *etree.Element, attributes map[string]string) {
	code, err := ConvertSymbol(command)
	element := parent.CreateElement("mo")
	addAttributes(element, attributes)
	if err == nil {
		element.SetText("&#x" + code + ";")
	} else {
		element.SetText(command)
	}
}

func appendPrefixElement(node Node, parent *etree.Element) {
	var size = "2.047em"

	if parent.SelectAttrValue("displaystyle", "none") == "false" || node.Token == TBINOM {
		size = "1.2em"
	}

	if node.Token == `\pmatrix` {
		// pmatrix opens with a CONTENT-SIZED stretchy '(' — matching \binom.
		// The missing minsize/maxsize was the fence asymmetry (research Open Q2):
		// binom's branch passed sizing, pmatrix's passed an empty map. PMOD keeps
		// its own unsized paren (out of scope for the matrix/binom fence fix).
		convertAndAppendCommand(`\lparen`, parent, map[string]string{"minsize": size, "maxsize": size})
	} else if node.Token == PMOD {
		convertAndAppendCommand(`\lparen`, parent, map[string]string{})
	} else if node.Token == BINOM || node.Token == DBINOM || node.Token == TBINOM {
		convertAndAppendCommand(`\lparen`, parent, map[string]string{"minsize": size, "maxsize": size})
	} else if node.Token == `\bmatrix` {
		convertAndAppendCommand(`\lbrack`, parent, map[string]string{})
	} else if node.Token == `\Bmatrix` {
		convertAndAppendCommand(`\lbrace`, parent, map[string]string{})
	} else if node.Token == `\vmatrix` {
		convertAndAppendCommand(`\vert`, parent, map[string]string{})
	} else if node.Token == `\Vmatrix` {
		convertAndAppendCommand(`\Vert`, parent, map[string]string{})
	} else if (node.Token == FRAC || node.Token == GENFRAC) && node.Delimiter != "" && node.Delimiter[0] != '.' {
		convertAndAppendCommand(string(node.Delimiter[0]), parent, map[string]string{"minsize": size, "maxsize": size})
	}
}

func appendPostfixElement(node Node, parent *etree.Element) {
	var size = "2.047em"

	if parent.SelectAttrValue("displaystyle", "none") == "false" || node.Token == TBINOM {
		size = "1.2em"
	}

	if node.Token == `\pmatrix` {
		// CLOSE with the MATCHING ')' — the unpatched postfix reused '\lparen',
		// emitting the opening '(' as the closing fence too (<mo>(…<mo>( ).
		// Content-sized to match the opening paren.
		convertAndAppendCommand(`\rparen`, parent, map[string]string{"minsize": size, "maxsize": size})
	} else if node.Token == PMOD {
		convertAndAppendCommand(`\lparen`, parent, map[string]string{})
	} else if node.Token == BINOM || node.Token == DBINOM || node.Token == TBINOM {
		// CLOSE with ')' (was '\lparen' — the same reused-open-paren bug).
		convertAndAppendCommand(`\rparen`, parent, map[string]string{"minsize": size, "maxsize": size})
	} else if node.Token == `\bmatrix` {
		convertAndAppendCommand(`\rbrack`, parent, map[string]string{})
	} else if node.Token == `\Bmatrix` {
		convertAndAppendCommand(`\rbrace`, parent, map[string]string{})
	} else if node.Token == `\vmatrix` {
		convertAndAppendCommand(`\vert`, parent, map[string]string{})
	} else if node.Token == `\Vmatrix` {
		convertAndAppendCommand(`\Vert`, parent, map[string]string{})
	} else if (node.Token == FRAC || node.Token == GENFRAC) && node.Delimiter != "" && node.Delimiter[0] != '.' {
		convertAndAppendCommand(string(node.Delimiter[1]), parent, map[string]string{"minsize": size, "maxsize": size})
	} else if width, ok := node.Attributes["width"]; node.Token == SKEW && ok {
		element := etree.NewElement("mspace")
		element.CreateAttr("width", "-"+width)
		parent.AddChild(element)
	}
}

func convertSymbol(node Node, parent *etree.Element, font map[string]string) {
	token := node.Token
	attributes := node.Attributes
	code, errCode := ConvertSymbol(token)

	if NUMBER_PATTERN.MatchString(token) {

		element := parent.CreateElement("mn")
		addAttributes(element, attributes)
		element.SetText(token)
		setFont(element, element.Tag, font)

	} else if slices.Contains(OPERATORS, token) {

		element := parent.CreateElement("mo")
		addAttributes(element, attributes)
		element.SetText("&#x" + code + ";")

		if token == `\|` {
			element.CreateAttr("fence", "false")
		} else if token == `\smallint` {
			element.CreateAttr("largeop", "false")
		}

		if slices.Contains([]string{"(", ")", "[", "]", "|", `\|`, `\{`, `\}`, `\surd`}, token) {
			element.CreateAttr("stretchy", "false")
			setFont(element, "fence", font)
		} else {
			setFont(element, element.Tag, font)
		}

	} else if value, err := strconv.ParseInt(code, 16, 64); code == "." || err != nil && (value >= 0x2200 && value < 0x22ff+1 || value >= 0x2190 && value < 0x21ff+1) {

		element := parent.CreateElement("mo")
		addAttributes(element, attributes)
		element.SetText("&#x" + code + ";")
		setFont(element, element.Tag, font)

	} else if slices.Contains(SPACE_LIST, token) {

		element := parent.CreateElement("mtext")
		addAttributes(element, attributes)
		element.SetText("&#x000A0;")
		setFont(element, "mtext", font)

	} else if token == NOT {

		padded := parent.CreateElement("mapped")
		padded.CreateAttr("width", "0")
		element := padded.CreateElement("mtext")
		element.SetText("&#x029F8;")

	} else if slices.Contains(SUB_LIST, token) {

		element := parent.CreateElement("mo")
		element.CreateAttr("movablelimits", "true")
		addAttributes(element, attributes)

		if token == INJLIM {
			element.SetText("inj&#x02006;lim")
		} else if token == INTOP {
			element.SetText("&#x0222B;")
		} else if token == LIMINF {
			element.SetText("lim&#x02006;inf")
		} else if token == LIMSUP {
			element.SetText("lim&#x02006;sup")
		} else if token == PROJLIM {
			element.SetText("proj&#x02006;lim")
		} else {
			element.SetText(token[1:])
		}

		setFont(element, element.Tag, font)
	} else if token == IDOTSINT {

		element := parent.CreateElement("mrow")
		addAttributes(element, attributes)

		for _, s := range []string{"&#x0222B;", "&#x022EF;", "&#x0222B;"} {
			child := element.CreateElement("mo")
			child.SetText(s)
		}

	} else if token == LATEX || token == TEX {

		localParent := parent.CreateElement("mrow")
		addAttributes(localParent, attributes)

		if token == LATEX {
			mi_l := localParent.CreateElement("mi")
			mi_l.SetText("L")
			space := localParent.CreateElement("mspace")
			space.CreateAttr("width", "-0.325em")
			padded := localParent.CreateElement("mpadded")
			padded.CreateAttr("height", "0.21ex")
			padded.CreateAttr("depth", "-0.21ex")
			padded.CreateAttr("voffset", "0.21ex")
			style := padded.CreateElement("mstyle")
			style.CreateAttr("displaystyle", "false")
			style.CreateAttr("scriptlevel", "1")
			row := style.CreateElement("mrow")
			mi_a := row.CreateElement("mi")
			mi_a.SetText("A")
			space = localParent.CreateElement("mspace")
			space.CreateAttr("width", "-0.17em")
			setFont(mi_l, mi_l.Tag, font)
			setFont(mi_a, mi_a.Tag, font)
		}

		mi_t := localParent.CreateElement("mi")
		mi_t.SetText("T")
		space := localParent.CreateElement("mspace")
		space.CreateAttr("width", "-0.14")
		padded := localParent.CreateElement("mpadded")
		padded.CreateAttr("height", "-0.5ex")
		padded.CreateAttr("depth", "0.5ex")
		padded.CreateAttr("voffset", "-0.5ex")
		row := padded.CreateElement("mrow")
		mi_e := row.CreateElement("mi")
		mi_e.SetText("E")
		space = localParent.CreateElement("mspace")
		space.CreateAttr("width", "-0.115em")
		mi_x := localParent.CreateElement("mi")
		mi_x.SetText("X")

		setFont(mi_t, mi_t.Tag, font)
		setFont(mi_e, mi_e.Tag, font)
		setFont(mi_x, mi_x.Tag, font)

	} else if strings.HasPrefix(token, OPERATORNAME) {

		element := parent.CreateElement("mo")
		addAttributes(element, attributes)
		element.SetText(token[14 : len(token)-1])

	} else if strings.HasPrefix(token, BACKSLASH) {

		element := parent.CreateElement("mi")
		addAttributes(element, attributes)

		if errCode == nil {
			element.SetText("&#x" + code + ";")
		} else if slices.Contains(FUNCTIONS, token) {
			element.SetText(token[1:])
		} else {
			element.SetText(token)
		}

		setFont(element, element.Tag, font)

	} else {
		element := parent.CreateElement("mi")
		addAttributes(element, attributes)
		element.SetText(token)
		setFont(element, element.Tag, font)
	}
}

// variantBase maps a Mathematical Alphanumeric font variant to the Unicode base
// codepoints of its uppercase-'A' and lowercase-'a' glyphs. MathML Core IGNORES
// the legacy `mathvariant` attribute for any non-"normal" value, so a styled
// letter must be emitted as its actual Mathematical-Alphanumeric CODEPOINT
// instead. Only the three variants the Objective-8 spike corpus exercises are
// mapped here — extend with fraktur/sans-serif/monospace (each has its own base
// offsets and, for some, named holes) when those are promoted to spike cases.
var variantBase = map[string][2]rune{
	"double-struck": {0x1D538, 0x1D552},
	"bold":          {0x1D400, 0x1D41A},
	"script":        {0x1D49C, 0x1D4B6},
}

// variantHoles are letters that live OUTSIDE the contiguous Mathematical
// Alphanumeric block (they sit in Letterlike Symbols, e.g. ℝ U+211D, ℒ U+2112)
// and must be looked up individually rather than by base offset.
var variantHoles = map[string]map[rune]rune{
	"double-struck": {
		'C': 0x2102, 'H': 0x210D, 'N': 0x2115, 'P': 0x2119,
		'Q': 0x211A, 'R': 0x211D, 'Z': 0x2124,
	},
	"script": {
		'B': 0x212C, 'E': 0x2130, 'F': 0x2131, 'H': 0x210B,
		'I': 0x2110, 'L': 0x2112, 'M': 0x2133,
		'e': 0x212F, 'g': 0x210A, 'o': 0x2134,
	},
}

// variantCodepoint returns the numeric-char-ref (&#xHHHH;) form of a single ASCII
// letter under a Mathematical Alphanumeric font variant, and whether it applied.
// Named holes are honoured; non-letters, multi-rune text, and unmapped variants
// return ("", false) so the caller keeps its default (attribute) behaviour.
func variantCodepoint(variant, text string) (string, bool) {
	base, ok := variantBase[variant]
	if !ok {
		return "", false
	}
	r := []rune(text)
	if len(r) != 1 {
		return "", false
	}
	c := r[0]
	if holes, ok := variantHoles[variant]; ok {
		if cp, ok := holes[c]; ok {
			return codepointRef(cp), true
		}
	}
	switch {
	case c >= 'A' && c <= 'Z':
		return codepointRef(base[0] + (c - 'A')), true
	case c >= 'a' && c <= 'z':
		return codepointRef(base[1] + (c - 'a')), true
	}
	return "", false
}

// codepointRef formats a rune as an uppercase hex numeric character reference,
// matching the &#xHHHH; convention the converter uses everywhere.
func codepointRef(cp rune) string {
	return "&#x" + strings.ToUpper(strconv.FormatInt(int64(cp), 16)) + ";"
}

func setFont(element *etree.Element, key string, font map[string]string) {
	// mathvariant→Unicode codepoint: MathML Core ignores non-"normal" mathvariant,
	// so a styled single letter (\mathbb{R}→ℝ, \mathbf{v}→𝐯, \mathcal{L}→ℒ) is
	// emitted as its Mathematical-Alphanumeric codepoint instead of an attribute
	// the browser drops. Only identifier letters are remapped; the attribute path
	// is preserved for every other variant/element (sans-serif, monospace, …).
	if element.Tag == "mi" {
		if cp, ok := variantCodepoint(font["default"], element.Text()); ok {
			element.SetText(cp)
			return
		}
	}
	if value, exist := font[key]; exist {
		element.CreateAttr("mathvariant", value)
	}
}

// isDisplayMode reports whether el will render in display (block) style. It walks
// the ancestor chain: the nearest element carrying an explicit displaystyle
// attribute wins (so a big operator nested inside \tfrac/\text —
// displaystyle="false" — does NOT auto-stack), otherwise the root
// <math display="block"> decides. Used to gate big-operator limit stacking.
func isDisplayMode(el *etree.Element) bool {
	for e := el; e != nil; e = e.Parent() {
		if v := e.SelectAttrValue("displaystyle", ""); v != "" {
			return v == "true"
		}
		if e.Tag == "math" {
			return e.SelectAttrValue("display", "") == "block"
		}
	}
	return false
}
