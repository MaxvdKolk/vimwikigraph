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
	// When any path matches this string, it is ignored in the resulting
	// graphs.
	ignorePath string

	// Contains all regular expressions to match links
	wikilink     *regexp.Regexp
	markdownlink *regexp.Regexp
	ignored      *regexp.Regexp
}

func newWiki(dir string, remap map[string]string, cluster bool, ignore string) (*Wiki, error) {
	wiki := Wiki{
		root:       dir,
		remap:      remap,
		graph:      make(map[string][]string),
		ignorePath: ignore,
		cluster:    cluster,
	}
	err := wiki.CompileExpressions()
	return &wiki, err
}

// Walk walks over all directories in wiki.root except for any directory
// contained in subDirToSkip.
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
		if wiki.IgnorePath(path) {
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
	wiki.wikilink = wikilink

	markdownlink, err := regexp.Compile(markdownref)
	if err != nil {
		return err
	}
	wiki.markdownlink = markdownlink

	if wiki.ignorePath != "" {
		ignored, err := regexp.Compile(wiki.ignorePath)
		if err != nil {
			return err
		}
		wiki.ignored = ignored
	}

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

func (wiki *Wiki) IgnorePath(path string) bool {
	// When no regexes are provided to be ignored, always accpet the files
	if wiki.ignored == nil {
		return false
	}

	// Otherwise, return true if any match with the given regex is observed,
	// in that case the link should not be added to the graph
	return wiki.ignored.Match([]byte(path))
}

// Add adds path to the wiki.graph when it contains links to other files.
//
// Only the relative paths are considered between the passed path and wiki.root.
func (wiki *Wiki) Add(path string) error {
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
			// do not insert links to ignored paths
			if wiki.IgnorePath(link) {
				continue
			}

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
// Only nodes, and their connections, are drawn if their sum of edges
// is greater than the provided level. For `level = 0` all nodes
// are inserted.
//
// If wiki.cluster == true any nodes that correspond to a subdirectory are
// inserted in the corresponding subgraph of that subdirectory. By default, the
// visualisation will highlight these subgraphs.
func (wiki *Wiki) Dot(level int, opts ...dot.GraphOption) *dot.Graph {
	graph := dot.NewGraph()
	for _, opt := range opts {
		opt.Apply(graph)
	}

	var a, b dot.Node

	for k, val := range wiki.graph {

		// skip nodes with less edges
		if len(val) < level {
			continue
		}

		// insert in subgraph if wiki and in subdirectory
		// FIXME move into func?
		dir, _ := filepath.Split(k)
		if wiki.cluster && dir != "" {
			subgraph := graph.Subgraph(dir, dot.ClusterOption{})
			a = subgraph.Node(k)
		} else {
			a = graph.Node(k)
		}

		for _, v := range val {
			// insert in subgraph if wiki and in subdirectory
			dir, _ := filepath.Split(v)
			if wiki.cluster && dir != "" {
				subgraph := graph.Subgraph(dir, dot.ClusterOption{})
				b = subgraph.Node(v)
			} else {
				b = graph.Node(v)
			}

			// only insert unique edges
			if len(graph.FindEdges(a, b)) == 0 {
				graph.Edge(a, b)
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
