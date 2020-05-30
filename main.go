package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emicklei/dot"
)

const ext string = ".wiki"

type Wiki struct {
	// Root directory of vimwiki structure
	root string
	// Connections from a file to its links
	graph map[string][]string
	// Directories to rename during processing
	remap map[string]string
	// Enable clustered plotting of files in sub directories
	cluster bool
}

func newWiki(dir string, remap map[string]string, cluster bool) *Wiki {
	wiki := Wiki{
		root:    dir,
		remap:   remap,
		graph:   make(map[string][]string),
		cluster: cluster,
	}
	return &wiki
}

// example: go run main.go example | dot -Tpng > test.png && open test.png
func main() {

	// fall back to current directory if no directory given
	var dir string
	if len(os.Args) == 1 {
		dir, _ = os.Executable()
		fmt.Fprintf(os.Stderr, "warning: using current directory: '%s'\n", dir)
	} else {
		if os.Args[1] != "-h" {
			dir = os.Args[1]
			os.Args = os.Args[1:]
		}
	}

	cluster := flag.Bool("cluster", false, "cluster nodes in sub directories")
	diary := flag.Bool("diary", false, "collapse all diary entries under a single `diary.wiki` node")
	flag.Parse()

	// remap any path that contains `diary` into `diary.wiki`
	remap := make(map[string]string)
	if !*diary {
		remap["diary"] = "diary.wiki"
	}

	// setup vimwiki struct
	wiki := newWiki(dir, remap, *cluster)

	// any trailing arguments are considered directories to skip
	subDirToSkip := []string{".git"}
	for _, dir := range flag.Args() {
		subDirToSkip = append(subDirToSkip, dir)
	}

	// walk directories and build graph
	if err := wiki.Walk(subDirToSkip); err != nil {
		log.Fatalf("Error when walking directories: %v", err)
	}

	// convert to a dot-graph for visualisation
	g := wiki.Dot(dot.Directed)
	g.Attr("rankdir", "LR")
	g.Write(os.Stdout)
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

	// match [[ ]] to detect links with format: [[link]]
	re, err := regexp.Compile(`\[\[([^\[\]]*)\]\]`)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		for _, match := range re.FindAllString(scanner.Text(), -1) {

			// [[file]] -> dir/file.wiki
			match = strings.Trim(match, "[]") + ext
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

			// prevent (possibly many) duplicates
			if unique(match, wiki.graph[key]) {
				wiki.graph[key] = append(wiki.graph[key], match)
			}
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
