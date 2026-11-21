// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//

package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// work is one entry of the bibliography.
type work struct {
	Type      string   `yaml:"type"`
	Year      string   `yaml:"year"`
	Title     string   `yaml:"title"`
	Container string   `yaml:"container"`
	Detail    string   `yaml:"detail"`
	Publisher string   `yaml:"publisher"`
	URL       string   `yaml:"url"`
	label     string   // the in-text citation, such as "Ajmani (2014b)"
	date      string   // the year as printed in the reference, with its letter
	Authors   []string `yaml:"authors"`
}

// prepare computes the in-text citation of every work. Works by the same first author in the same year get a
// letter, in the order of their titles, so that citations stay unambiguous.
func (t *taxonomy) prepare() error {
	groups := map[string][]string{}

	for id, w := range t.Sources {
		if len(w.Authors) == 0 || w.Title == "" || w.Year == "" {
			return fmt.Errorf("source %s: authors, year, and title are required", id)
		}

		switch w.Type {
		case "book", "article", "web", "talk", "report", "software":
		default:
			return fmt.Errorf("source %s: unknown type %q", id, w.Type)
		}

		key := surname(w.Authors[0]) + "|" + w.Year
		groups[key] = append(groups[key], id)
	}

	for _, ids := range groups {
		slices.SortFunc(ids, func(a, b string) int { return strings.Compare(t.Sources[a].Title, t.Sources[b].Title) })

		for i, id := range ids {
			w := t.Sources[id]
			year := w.Year

			if len(ids) > 1 {
				year += string(rune('a' + i))
			}

			w.date = year
			w.label = citeNames(w.Authors) + " (" + year + ")"
			t.Sources[id] = w
		}
	}

	return nil
}

func isPerson(author string) bool { return strings.Contains(author, ",") }

func surname(author string) string {
	name, _, _ := strings.Cut(author, ",")

	return name
}

// initials turns "Edsger W." into "E. W.".
func initials(given string) string {
	var parts []string

	for name := range strings.FieldsSeq(given) {
		for piece := range strings.SplitSeq(name, "-") {
			parts = append(parts, string(unicode.ToUpper([]rune(piece)[0]))+".")
		}
	}

	return strings.Join(parts, " ")
}

func citeNames(authors []string) string {
	names := make([]string, len(authors))
	for i, a := range authors {
		names[i] = surname(a)
	}

	switch len(names) {
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}

	return names[0] + " et al."
}

func refNames(authors []string) string {
	names := make([]string, len(authors))

	for i, a := range authors {
		if !isPerson(a) {
			names[i] = a
			continue
		}

		last, given, _ := strings.Cut(a, ",")
		names[i] = last + ", " + initials(given)
	}

	switch len(names) {
	case 1:
		return names[0]
	case 2:
		return names[0] + ", & " + names[1]
	}

	return strings.Join(names[:len(names)-1], ", ") + ", & " + names[len(names)-1]
}

// reference formats the entry in APA style.
func (w work) reference() string {
	end := func(s string) string {
		if strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") {
			return s
		}

		return s + "."
	}

	parts := []string{end(refNames(w.Authors)), "(" + w.date + ")."}

	switch w.Type {
	case "book":
		parts = append(parts, end("*"+w.Title+"*"), end(w.Publisher))
	case "report":
		title := "*" + w.Title + "*"
		if w.Detail != "" {
			title += " (" + w.Detail + ")"
		}

		parts = append(parts, end(title), end(w.Publisher))
	case "software":
		parts = append(parts, end("*"+w.Title+"* [Computer software]"), end(w.Publisher))
	case "talk":
		parts = append(parts, end("*"+w.Title+"* [Conference presentation]"), end(w.Container))
	case "article":
		parts = append(parts, end(w.Title), end("*"+w.Container+"*, "+w.Detail))
	default:
		parts = append(parts, end(w.Title), end("*"+w.Container+"*"))
	}

	if w.URL != "" {
		parts = append(parts, "<"+w.URL+">")
	}

	return strings.Join(slices.DeleteFunc(parts, func(s string) bool { return s == "." || s == "" }), " ")
}

// sortedSources lists the source ids as a bibliography is ordered: by author, then year, then title.
func (t *taxonomy) sortedSources() []string {
	ids := make([]string, 0, len(t.Sources))
	for id := range t.Sources {
		ids = append(ids, id)
	}

	slices.SortFunc(ids, func(a, b string) int {
		wa, wb := t.Sources[a], t.Sources[b]

		return strings.Compare(
			strings.ToLower(refNames(wa.Authors))+"|"+wa.Year+"|"+strings.ToLower(wa.Title),
			strings.ToLower(refNames(wb.Authors))+"|"+wb.Year+"|"+strings.ToLower(wb.Title),
		)
	})

	return ids
}

// renderBibliography lists the references, with the pages that cite each of them.
func (s *site) renderBibliography(from string) string {
	var out []string

	for _, id := range s.tax.sortedSources() {
		w := s.tax.Sources[id]
		entry := `<span id="source-` + id + `"></span>` + w.reference()

		var cited []string

		for _, p := range s.sorted(func(*pattern) bool { return true }) {
			for _, src := range p.Sources {
				if src.ID != id {
					continue
				}

				item := fmt.Sprintf("[%s](%s)", p.Title, link(from, p.path))
				if src.At != "" {
					item += " (" + src.At + ")"
				}

				cited = append(cited, item)
			}
		}

		if len(cited) == 0 {
			continue // a bibliography lists what is cited
		}

		entry += "  \n    <small>Cited in: " + strings.Join(cited, "; ") + "</small>"

		out = append(out, entry)
	}

	return strings.Join(out, "\n\n")
}
