package katex

import "os"

// CSS KaTeX CSS content
var CSS = loadFile("static/katex/katex.min.css")

// JS KaTeX JS content
var JS = loadFile("static/katex/katex.min.js")

func loadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
