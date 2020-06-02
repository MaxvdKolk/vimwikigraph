package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emicklei/dot"
)

const wiki_ext string = ".wiki"
const wikiref string = `\[\[([^\[\]]*)\]\]`
const markdownref string = `\[(.*)\]\((.*)\)`

type Wiki struct {
	// Root directory of vimwiki structure
	root string
	// Connections from a file to its links
	graph map[string][]string
	// Directories to rename during processing
	remap map[string]string
	// Enable clustered plotting of files in sub directories
	cluster bool

	// Contains all regular expressions to match links
	wikilink     *regexp.Regexp
	markdownlink *regexp.Regexp
}

func newWiki(dir string, remap map[string]string, cluster bool) *Wiki {
	wiki := Wiki{
		root:    dir,
		remap:   remap,
		graph:   make(map[string][]string),
		cluster: cluster,
	}
	wiki.CompileExpressions()
	return &wiki
}

// Walk walks over all directories in wiki.root except for any directory
// contianed in subDirToSkip.
func (wiki *Wiki) Walk(subDirToSkip []string) error {
	err := filepath.Walk(wiki.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
		if info.IsDir() {
			for _, s := range subDirToSkip {
				if info.Name() == s {
					fmt.Fprintf(os.Stderr, "skipping: %v\n", info.Name())
					return filepath.SkipDir
				}
			}
			return nil
		}
		return wiki.Add(path)
	})
	return err
}

func (wiki *Wiki) Insert(key, value string) {
	// prevent (possibly many) duplicates
	if unique(value, wiki.graph[key]) {
		wiki.graph[key] = append(wiki.graph[key], value)
	}
}

func (wiki *Wiki) Remap(dir, key, match string) (string, string) {

	// joins current directory with link
	match = filepath.Join(dir, match)

	// apply remap naming, diary/file.wiki -> diary.wiki
	for k, v := range wiki.remap {
		if k == dir {
			key = v
		}
		if strings.Contains(match, k) {
			match = v
		}
	}

	return key, match
}

// Compile compiles all regex to match links with
func (wiki *Wiki) CompileExpressions() error {
	wikilink, err := regexp.Compile(wikiref)
	if err != nil {
		return err
	}
	markdownlink, err := regexp.Compile(markdownref)
	if err != nil {
		return err
	}
	wiki.wikilink, wiki.markdownlink = wikilink, markdownlink
	return nil
}

// Links returns all links available in text.
func (wiki *Wiki) Links(text string) []string {

	// wiki syntax
	wikilinks := wiki.WikiLinks(text)
	for i, m := range wikilinks {
		wikilinks[i] = wiki.ParseWikiLinks(m)
	}

	// markdown syntax
	markdownlinks := wiki.MarkdownLinks(text)
	for i, m := range markdownlinks {
		link := wiki.ParseMarkdownLinks(m)
		if link != "" {
			markdownlinks[i] = link
		}
	}
	return append(wikilinks, markdownlinks...)
}

// WikiLinks matches on all vimwiki syntax links in text.
func (wiki *Wiki) WikiLinks(text string) []string {
	return wiki.wikilink.FindAllString(text, -1)
}

// MarkdownLinks matches on all markdown syntax links in text.
func (wiki *Wiki) MarkdownLinks(text string) []string {
	return wiki.markdownlink.FindAllString(text, -1)
}

// ParseMarkdownLinks extracts the filename from markdown syntax links.
func (wiki *Wiki) ParseMarkdownLinks(link string) string {
	idx := strings.Index(link, "(")
	link = link[idx:]
	link = strings.Trim(link, "()")

	ext := filepath.Ext(link)
	if ext == ".md" || ext == ".wiki" {
		return link
	}

	// assume it refers to a local markdown file
	if ext == "" {
		return link + ".md"
	}

	// if ext is anything else, we should probably skip the file
	return ""
}

// ParseWikiLinks extracts the filename from vimwiki syntax links.
func (wiki *Wiki) ParseWikiLinks(link string) string {
	// [[file]] -> dir/file.wiki
	link = strings.Trim(link, "[]")

	// split of description [[link|description]]
	idx := strings.Index(link, "|")
	if idx > 0 {
		link = link[:idx]
	}

	ext := filepath.Ext(link)
	if ext != ".md" && ext != ".wiki" {
		link += ".wiki"
	}
	return link
}

// Add adds path to the wiki.graph when it contains links to other files.
//
// Only the relative paths are considered between the passed path and wiki.root.
func (wiki *Wiki) Add(path string) error {

	log.Println(path)

	key, err := filepath.Rel(wiki.root, path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(key) // current dir when in subdirectory

	// initialise a node
	if _, ok := wiki.graph[key]; !ok {
		wiki.graph[key] = make([]string, 0)
	}

	// open file to find links
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		for _, link := range wiki.Links(scanner.Text()) {

			// rename and/or collapse folders
			key, link = wiki.Remap(dir, key, link)

			// insert into the graph
			wiki.Insert(key, link)
		}
	}
	return scanner.Err()
}

// Dot converts wiki.graph into dot.Graph.
//
// If wiki.cluster == true any nodes that correspond to a subdirectory are
// inserted in the corresponding subgraph of that subdirectory. By default, the
// visualisation will highlight these subgraphs.
func (wiki *Wiki) Dot(opts ...dot.GraphOption) *dot.Graph {
	graph := dot.NewGraph()
	for _, opt := range opts {
		opt.Apply(graph)
	}

	for k, val := range wiki.graph {
		g := graph

		if wiki.cluster {
			dir, _ := filepath.Split(k)
			if dir != "" {
				g = graph.Subgraph(dir, dot.ClusterOption{})
			}
		}

		for _, v := range val {
			a := g.Node(k)
			b := g.Node(v)

			// prevent duplicates
			if len(g.FindEdges(a, b)) == 0 {
				g.Edge(a, b)
			}
		}
	}
	return graph
}

// unique returns true when s is not present in values
func unique(s string, vals []string) bool {
	for _, v := range vals {
		if s == v {
			return false
		}
	}
	return true
}
