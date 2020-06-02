package main

import (
	"testing"
)

type match struct {
	text    string
	matches []string
	links   []string
	dir     []string
}

func TestMappingCollapse(t *testing.T) {
	cases := []match{
		match{
			text:  "[[diary/link]]",
			links: []string{"diary"},
			dir:   []string{"."},
		},
		match{
			text:  "[[../link]]",
			links: []string{"link.wiki"},
			dir:   []string{"diary"},
		},
		match{
			text:  "[[link]]",
			links: []string{"diary"},
			dir:   []string{"diary"},
		},
	}

	wiki := Wiki{}
	if err := wiki.CompileExpressions(); err != nil {
		t.Error(err)
	}
	wiki.remap = make(map[string]string)
	wiki.remap["diary"] = "diary"

	for _, c := range cases {
		for i, m := range wiki.Links(c.text) {

			_, link := wiki.Remap(c.dir[i], ".", m)

			if link != c.links[i] {
				t.Errorf("Expected link: %v:, got: %v", c.links[i], link)
			}
		}
	}
}

func TestMappingNoCollapse(t *testing.T) {
	cases := []match{
		match{
			text:    "[[diary/link]]",
			matches: []string{"[[diary/link]]"},
			links:   []string{"diary/link.wiki"},
			dir:     []string{"."},
		},
		match{
			text:    "[[../link]]",
			matches: []string{"[[../link]]"},
			links:   []string{"link.wiki"},
			dir:     []string{"diary.wiki"},
		},
		match{
			text:  "[[link]]",
			links: []string{"diary/link.wiki"},
			dir:   []string{"diary"},
		},
	}

	wiki := Wiki{}
	if err := wiki.CompileExpressions(); err != nil {
		t.Error(err)
	}

	for _, c := range cases {
		for i, m := range wiki.Links(c.text) {

			_, link := wiki.Remap(c.dir[i], ".", m)

			if link != c.links[i] {
				t.Errorf("Expected link: %v:, got: %v", c.links[i], link)
			}
		}
	}
}

func TestMatchParseMarkdownLinks(t *testing.T) {
	cases := []match{
		match{
			text:    "[link](url)",
			matches: []string{"[link](url)"},
			links:   []string{"url.md"},
		},
		match{
			text:    "[link](url.md)",
			matches: []string{"[link](url.md)"},
			links:   []string{"url.md"},
		},
		match{
			text:    "[link](vimwiki.wiki)",
			matches: []string{"[link](vimwiki.wiki)"},
			links:   []string{"vimwiki.wiki"},
		},
	}

	wiki := Wiki{}
	if err := wiki.CompileExpressions(); err != nil {
		t.Error(err)
	}

	for _, c := range cases {
		matches := wiki.MarkdownLinks(c.text)

		if len(matches) != len(c.matches) {
			t.Errorf("Expected %d matches, got %d matches", len(c.matches), len(matches))
		}

		for i, m := range matches {
			if m != c.matches[i] {
				t.Errorf("Expected match %v, got %v", c.matches[i], m)
			}
		}

		for i, m := range matches {
			link := wiki.ParseMarkdownLinks(m)
			if link != c.links[i] {
				t.Errorf("Expected link: %v, got %v", c.links[i], link)
			}
		}
	}
}

func TestMatchParseWikiLinks(t *testing.T) {
	cases := []match{
		match{
			text:    "[[link]]",
			matches: []string{"[[link]]"},
			links:   []string{"link.wiki"},
		},
		match{
			text:    "[[a]]\n[[b]]",
			matches: []string{"[[a]]", "[[b]]"},
			links:   []string{"a.wiki", "b.wiki"},
		},
		match{
			text:    "[[link|description]]",
			matches: []string{"[[link|description]]"},
			links:   []string{"link.wiki"},
		},
		match{
			text:    "[[link.wiki]]",
			matches: []string{"[[link.wiki]]"},
			links:   []string{"link.wiki"},
		},
		match{
			text:    "[[link.md]]",
			matches: []string{"[[link.md]]"},
			links:   []string{"link.md"},
		},
	}

	wiki := Wiki{}
	if err := wiki.CompileExpressions(); err != nil {
		t.Error(err)
	}

	for _, c := range cases {
		matches := wiki.WikiLinks(c.text)

		if len(matches) != len(c.matches) {
			t.Errorf("Expected %d matches, got %d matches", len(c.matches), len(matches))
		}

		for i, m := range matches {
			if m != c.matches[i] {
				t.Errorf("Expected match %v, got %v", c.matches[i], m)
			}
		}

		for i, m := range matches {
			link := wiki.ParseWikiLinks(m)
			if link != c.links[i] {
				t.Errorf("Expected link: %v, got %v", c.links[i], link)
			}
		}
	}
}
