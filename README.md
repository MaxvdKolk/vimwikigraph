## vimwikigraph - visualise vimwiki links using graphviz dot language
`vimwikigraph` walks all files in a vimwiki directory and builds a graph
between the encountered files and their internal references by wiki
links `[[wiki link]]`. The graph is converted to the dot language using
`dot`.

## Usage
```
./vimwikigraph $HOME/vimwiki | dot -Tpng > test.png && open test.png
```
`diary`: collapse all diary entries under a single node `diary.wiki`
`-cluster`: cluster subdirectories as subgraphs

## Examples
```
./vimwikigraph example | dot -Tpng > example.png
```
figure
```
./vimwikigraph example -diary | dot -Tpng > example.png
```
figure
```
./vimwikigraph example -diary -cluster | dot -Tpng > example.png
```

## Installation
```
go get github.com/maxvdkolk/vimwikigraph
```
