# Go-ERD [WIP]

Entity Relationship Diagrams for Golang with GraphViz.

# Why

Visualize package's types and their inter-relationships to aid exploring and studying source code.

# Installation

```
go get github.com/gmarik/go-erd
```

# Use

```
# go-erd -path <path>
# ie
go-erd -path $(go env GOROOT)/src/go/ast/ |dot -Tsvg > out.svg
open out.svg
```

### go/ast

![go/ast](https://cdn.rawgit.com/gmarik/go-erd/master/examples/go-ast.svg)

### go/types

![go/ast](https://cdn.rawgit.com/gmarik/go-erd/master/examples/go-types.svg)

### net/http

Simple on the outside very complex on the inside.

![go/ast](https://cdn.rawgit.com/gmarik/go-erd/master/examples/net-http.svg)

## TODO

- [ ] cleanup
- [ ] exhaustive coverage for types
- [ ] flag to show only exported fields
