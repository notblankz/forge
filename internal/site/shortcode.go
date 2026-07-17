package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

// TODO: add doc comments
type Shortcodes struct {
	set *template.Template
}

type shortcodeHelpers struct {
	assetsDir string
}

func loadShortcodes(themeDir, siteLayoutsDir, contentDir string) (*Shortcodes, error) {
	helpers := shortcodeHelpers{assetsDir: filepath.Join(contentDir, "assets")}
	set := template.New("shortcodes").Funcs(helpers.funcMap())

	for _, dir := range []string{
		filepath.Join(themeDir, "layouts", "shortcodes"),
		filepath.Join(siteLayoutsDir, "shortcodes"),
	} {
		matches, err := filepath.Glob(filepath.Join(dir, "*.html"))
		if err != nil {
			return nil, err
		}
		if len(matches) > 0 {
			if _, err := set.ParseFiles(matches...); err != nil {
				return nil, err
			}
		}
	}

	return &Shortcodes{set: set}, nil
}

func (h shortcodeHelpers) funcMap() template.FuncMap {
	return template.FuncMap{
		"readDir": h.readDir,
	}
}

// expansion is the result of a pre-goldmark shortcode pass: markdown with each
// shortcode tag replaced by an opaque placeholder token, plus the rendered HTML
// each token stands for which will be replaced using expansion.Restore
type expansion struct {
	markdown     string
	replacements map[string]string
}

// Credit to Claude Opus 4.8 (high)
const (
	fencedTriple = "```[\\s\\S]*?```" // ```code block```
	fencedTilde  = "~~~[\\s\\S]*?~~~" // ~~~code block~~~
	inlineCode   = "`[^`\n]*`"        // `inline code`

	shortcode = `\{\{<\s*(?P<closing>/?)\s*(?P<name>[\w-]+)\s*(?P<params>[\s\S]*?)\s*>\}\}`
)

var tagPattern = regexp.MustCompile(
	"(?P<code>" + fencedTriple + "|" + fencedTilde + "|" + inlineCode + ")|" + shortcode,
)

// End of credit to Claude 4.8 (high)

var (
	codeGroup    = tagPattern.SubexpIndex("code")
	nameGroup    = tagPattern.SubexpIndex("name")
	paramsGroup  = tagPattern.SubexpIndex("params")
	closingGroup = tagPattern.SubexpIndex("closing")
)

// group returns the substring captured by group idx in match m, or "" if that
// group did not participate
func groupSubstring(md string, m []int, idx int) string {
	start := m[2*idx]
	if start < 0 {
		return ""
	}
	return md[start:m[2*idx+1]]
}

func (s *Shortcodes) Expand(md string) (expansion, error) {
	e := expansion{replacements: map[string]string{}}
	var out strings.Builder
	tokenNum := 0
	cursor := 0

	// This matches is a [][]int and each row contains index of the matched string indices
	// Each row includes the different groups in that specific match of the string (can be worded better)
	matches := tagPattern.FindAllStringSubmatchIndex(md, -1)
	for i := 0; i < len(matches); i++ {
		m := matches[i]
		out.WriteString(md[cursor:m[0]]) // writing the string from cursor till start of the match string

		// if the match is a code region we copy it as-is
		// Note: we multiply codeGroup by 2 since the start index of any match is (2 * group index)
		if m[2*codeGroup] >= 0 {
			out.WriteString(md[m[0]:m[1]])
			cursor = m[1]
			continue
		}

		name := groupSubstring(md, m, nameGroup)
		params := groupSubstring(md, m, paramsGroup)
		closing := groupSubstring(md, m, closingGroup)

		// a closing tag that reaches here has no opener (a paired one is consumed below)
		if closing == "/" {
			return expansion{}, fmt.Errorf("shortcode: unexpected closing tag {{< /%s >}}", name)
		}

		var body string
		consumedTo := m[1] // marker to signify where the cursor has consumed till

		// The below if stmt checks if there exists a valid match after the current match
		if next := i + 1; next < len(matches) {
			n := matches[next]
			// If the next match (n) has a closing tag (using closing group) and has the same name
			// as the current loop match (m) that means the text between needs to be put into body
			if groupSubstring(md, n, closingGroup) == "/" && groupSubstring(md, n, nameGroup) == name {
				body = md[m[1]:n[0]] // The body will be from the ending of the current loop match till the start of the next match
				consumedTo = n[1]    // Update consumed marker to after the closing tag match
				i = next             // Update the iterating variable so that the next iteration does not process it again
			}
		}

		html, err := s.render(name, params, body)
		if err != nil {
			return expansion{}, err
		}

		// handle the match being an actual shortcode which needs to be replaced with a TOKEN
		// TOKEN format: forgeshortcode00000end, forgeshortcode00001end and so on
		token := fmt.Sprintf("forgeshortcode%05dend", tokenNum)
		tokenNum++

		// Add the rendered HTML to the map with a the token as the key for later replacement
		e.replacements[token] = html

		out.WriteString(token) // Write the token in the expanded markdown

		cursor = consumedTo // update cursor to point at the next character after the previous match string
	}

	// Write the remaining data from the cursor till the end of the md string
	// This is to make sure that no markdown is left behind after all the matches have been processed
	out.WriteString(md[cursor:])

	e.markdown = out.String()
	return e, nil
}

// render executes the named shortcode template with its parsed params (plus a
// paired tag's body as .Body) and returns the resulting HTML
func (s *Shortcodes) render(name, rawParams, body string) (string, error) {
	tmpl := s.set.Lookup(name + ".html")
	if tmpl == nil {
		return "", fmt.Errorf("shortcode: unknown shortcode %q", name)
	}

	data, err := parseParams(rawParams)
	if err != nil {
		return "", fmt.Errorf("shortcode %q: %w", name, err)
	}

	// This Body field stores the body of the shortcode IF the shortcode
	// needs to store a body between it's tags
	// For eg: {{< xyz >}} This is the body of the shortcode {{< /xyz >}}
	data["Body"] = body

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("shortcode %q: %w", name, err)
	}

	return buf.String(), nil
}

func (e *expansion) Restore(html string) string {
	for token, snippet := range e.replacements {
		// Replace all occurences of <p>forgeshortcode00000end</p> with the snippet from the map
		// NOTE: Since we pass the tokenised markdown into Goldmark the TOKENs get converted into <p></p>
		html = strings.ReplaceAll(html, "<p>"+token+"</p>", snippet)
		// This is to just replace the leftover inline tokens if any
		html = strings.ReplaceAll(html, token, snippet)
	}

	return html
}

func (h shortcodeHelpers) readDir(sub string) ([]string, error) {
	var webPaths []string
	err := filepath.WalkDir(filepath.Join(h.assetsDir, sub), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(h.assetsDir, path)
		if err != nil {
			return err
		}
		webPaths = append(webPaths, "/assets/"+filepath.ToSlash(rel))
		return nil
	})
	return webPaths, err
}

// shortcode key=value parsing
func parseParams(raw string) (map[string]any, error) {

	// needs to match cols=3 dir=gallery etc etc
	params := map[string]any{}

	// strip all the leading and trailing white space
	s := strings.TrimSpace(raw)

	for len(s) > 0 {
		// split on the first "=" - before is the key, after is the value;
		// ok is false when there's no "=" at all, i.e. a malformed pair
		before, after, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("malformed parameters near %q", s)
		}

		// get the key value i.e. from the start of the current string till the equal to sign
		key := strings.TrimSpace(before)
		if key == "" {
			return nil, fmt.Errorf("empty parameter name near %q", s)
		}

		// extract the remaining string
		s = strings.TrimSpace(after)

		var (
			val any
			err error
		)

		// switch on the type of value
		// NOTE: The remaining string here will be after the equal sign, hence we can use the HasPrefix func
		switch {
		case strings.HasPrefix(s, `"`):
			val, s, err = parseQuoted(s)
		case strings.HasPrefix(s, "["):
			val, s, err = parseArray(s)
		default:
			val, s = parseBare(s)
		}

		if err != nil {
			return nil, err
		}

		// add the key value to the map
		params[key] = val
		// Clean the string for the next iteration
		s = strings.TrimSpace(s)
	}

	return params, nil
}

func parseQuoted(s string) (value string, rest string, err error) {
	// s starts with the opening quote; cut s[1:] at the next quote.
	// value is the text up to the closing quote, remainder is the rest,
	// ok is false if there's no closing quote (unterminated)
	value, rest, ok := strings.Cut(s[1:], `"`)
	if !ok {
		return "", "", fmt.Errorf("unterminated quoted value")
	}
	return value, rest, nil
}

func parseArray(s string) (value []string, rest string, err error) {
	// cut at the closing "]" - inner is everything inside the brackets,
	// rest is what follows; ok is false if there's no "]" (unterminated)
	inner, rest, ok := strings.Cut(s, "]")
	if !ok {
		return nil, "", fmt.Errorf("unterminated array value")
	}
	inner = strings.TrimPrefix(inner, "[") // drop the leading "["

	var items []string
	for item := range strings.SplitSeq(inner, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		items = append(items, strings.Trim(item, `"`))
	}

	return items, rest, nil
}
func parseBare(s string) (value string, rest string) {
	i := strings.IndexAny(s, " \t\r\n")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i:]
}
