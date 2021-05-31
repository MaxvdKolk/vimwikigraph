package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/emicklei/dot"
)

type match struct {
	text    string
	matches []string
	links   []string
	dir     []string
	ignore  string
}

func TestMappingCollapse(t *testing.T) {
	cases := []match{
		match{
			text:   "[[diary/link]]",
			links:  []string{"diary"},
			dir:    []string{"."},
			ignore: "",
		},
		match{
			text:   "[[../link]]",
			links:  []string{"link.wiki"},
			dir:    []string{"diary"},
			ignore: "",
		},
		match{
			text:   "[[link]]",
			links:  []string{"diary"},
			dir:    []string{"diary"},
			ignore: "",
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
			ignore:  "",
		},
		match{
			text:    "[[../link]]",
			matches: []string{"[[../link]]"},
			links:   []string{"link.wiki"},
			dir:     []string{"diary.wiki"},
			ignore:  "",
		},
		match{
			text:   "[[link]]",
			links:  []string{"diary/link.wiki"},
			dir:    []string{"diary"},
			ignore: "",
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
			ignore:  "",
		},
		match{
			text:    "[link](url.md)",
			matches: []string{"[link](url.md)"},
			links:   []string{"url.md"},
			ignore:  "",
		},
		match{
			text:    "[link](vimwiki.wiki)",
			matches: []string{"[link](vimwiki.wiki)"},
			links:   []string{"vimwiki.wiki"},
			ignore:  "",
		},
		match{
			text:    "![figure](image.png)",
			matches: []string{"[figure](image.png)"},
			links:   []string{""},
			ignore:  "",
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
			ignore:  "",
		},
		match{
			text:    "[[link|description]]",
			matches: []string{"[[link|description]]"},
			links:   []string{"link.wiki"},
			ignore:  "",
		},
		match{
			text:    "[[link.wiki]]",
			matches: []string{"[[link.wiki]]"},
			links:   []string{"link.wiki"},
			ignore:  "",
		},
		match{
			text:    "[[link.md]]",
			matches: []string{"[[link.md]]"},
			links:   []string{"link.md"},
			ignore:  "",
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

func TestNodeConnectionLevel(t *testing.T) {
	os.Chdir(".")
	dir, _ := os.Executable()
	t.Log(dir)
	wiki, err := newWiki("example", make(map[string]string), false, "")

	if err != nil {
		t.Errorf("Expected no error in constructor")
	}

	// build 0 < i < 10 entries with each i connections, where the current
	// nodes are not considered as connections
	wiki.graph = make(map[string][]string)
	for i := 0; i < 10; i++ {
		k := fmt.Sprintf("%d", i)
		wiki.graph[k] = make([]string, 0, 0)
		for j := 0; j < i; j++ {
			wiki.graph[k] = append(wiki.graph[k], fmt.Sprintf("%d00", j))
		}
	}

	// level zero: 9 entries + 10 connections: 19 nodes in total
	// each increment of the level should draw one node less
	for l := 0; l < 10; l++ {
		g := wiki.Dot(l, dot.Directed)
		nconn := len(g.FindNodes())
		exp := 19 - l
		if nconn != exp {
			t.Errorf("For level %v: exp: %v, got %v", l, exp, nconn)
		}
	}

	// higher than maximum connectivity should result in zero nodes
	nconn := len(wiki.Dot(10, dot.Directed).FindNodes())
	if nconn > 0 {
		t.Errorf("Expected 0 connections, got %v", nconn)
	}
}

func TestIgnorePaths(t *testing.T) {
	wiki, err := newWiki("example", make(map[string]string), false, "t*")
	if err != nil {
		t.Errorf("Expected no error in constructor")
	}
	if !wiki.IgnorePath("test") {
		t.Errorf("Path should be discarged given the regex")
	}
}
