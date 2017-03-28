package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
)

//
// from: https://github.com/golang/example/tree/master/gotypes#typeandvalue
//
// running: go run ./cmd/goerd/main.go -path cmd/traverse/|dot -Tsvg > out.svg
func main() {

	var (
		path     = flag.String("path", "", "path parse")
		hostport = flag.String("http", "", "Host:port for web server")
	)

	flag.Parse()

	if *path == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *hostport != "" {
		webServe(*hostport, *path)
	} else {
		dotRender(os.Stdout, inspectDir(*path))
	}
}

func webServe(hp, path string) {
	http.HandleFunc("/graph", func(w http.ResponseWriter, r *http.Request) {
		var dot bytes.Buffer
		dotRender(&dot, inspectDir(path))
		w.Header().Set("content-type", "text/plain")
		w.Write(dot.Bytes())
	})
	http.Handle("/", http.FileServer(http.Dir("examples/d3")))
	http.ListenAndServe(hp, nil)
}

type NamedType struct {
	Id   *ast.Ident
	Type ast.Expr
}

func dotRender(out io.Writer, pkgTypes map[string]map[string]NamedType) {

	fmt.Fprintf(out, "digraph %q { \n", "GoERD")
	var buf bytes.Buffer
	for pkg, types := range pkgTypes {

		fmt.Fprintf(out, "subgraph %q {\n", pkg)
		fmt.Fprintf(out, "label=%q;\n", pkg)

		// Nodes
		var i int
		for _, snode := range types {
			i++
			buf.Reset()

			switch t := snode.Type.(type) {
			case *ast.Ident:
				var label = fmt.Sprintf(`%s\ %s`, typeName(snode.Id), t.Name)
				fmt.Fprintf(out, " \"node-%s\" [shape=ellipse,label=\"%s\"];\n", typeName(snode.Id), label)
			case *ast.SelectorExpr:
				var label = fmt.Sprintf(`%s\ %s`, typeName(snode.Id), typeName(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=ellipse,label=\"%s\"];\n", typeName(snode.Id), label)
			case *ast.ChanType:
				var label = fmt.Sprintf(`%s\ %s`, typeName(snode.Id), typeName(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=box,label=\"%s\"];\n", typeName(snode.Id), label)
			case *ast.FuncType:
				var label = fmt.Sprintf(`%s\ %s`, typeName(snode.Id), typeName(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=rectangle,label=\"%s\"];\n", typeName(snode.Id), label)
			case *ast.ArrayType:
				var label = fmt.Sprintf(`%s\ %s`, typeName(snode.Id), typeName(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=rectangle,label=\"%s\"];\n", typeName(snode.Id), label)
			case *ast.MapType:
				var label = fmt.Sprintf(`%s\ %s`, typeName(snode.Id), typeName(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=rectangle,label=\"%s\"];\n", typeName(snode.Id), label)
			case *ast.InterfaceType:
				fmt.Fprintf(&buf, `{%s\ interface|`, snode.Id.Name)
				for i, f := range t.Methods.List {
					if i > 0 {
						fmt.Fprintf(&buf, `|`)
					}
					fmt.Fprintf(&buf, `<f%d>`, i)
					// a,b,c Type
					for ii, n := range f.Names {
						fmt.Fprintf(&buf, "%s", n.Name)
						if ii > 0 {
							fmt.Fprintf(&buf, `\,`)
						}
					}
					if len(f.Names) > 0 {
						fmt.Fprintf(&buf, `\ `)
					}
					fmt.Fprintf(&buf, `%s`, typeName(f.Type))
				}
				fmt.Fprintf(&buf, `}`)
				fmt.Fprintf(out, " \"node-%s\" [shape=Mrecord,label=\"%s\"];\n", typeName(snode.Id), buf.String())
			case *ast.StructType:
				fmt.Fprintf(&buf, `{%s|`, snode.Id.Name)
				for i, f := range t.Fields.List {
					if i > 0 {
						fmt.Fprintf(&buf, "|")
					}
					fmt.Fprintf(&buf, `<f%d>`, i)

					for ii, n := range f.Names {
						if ii > 0 {
							fmt.Fprintf(&buf, `\,\ `)
						}
						fmt.Fprintf(&buf, `%s`, n.Name)
					}
					if len(f.Names) > 0 {
						fmt.Fprintf(&buf, `\ `)
					}
					fmt.Fprintf(&buf, `%s`, typeName(f.Type))
				}
				fmt.Fprintf(&buf, `}`)
				fmt.Fprintf(out, " \"node-%s\" [shape=record,label=\"%s\"];\n", typeName(snode.Id), buf.String())
			default:
				fmt.Fprintf(os.Stderr, "MISSED: %s: %#v\n ", typeName(t), snode)
			}
		}

		// Edges
		for _, snode := range types {
			switch t := snode.Type.(type) {
			// TODO: exhaustive switch
			case *ast.FuncType:
				for i, typ := range dependants(t) {
					var from = fmt.Sprintf(`"node-%s":f%d`, snode.Id, i)
					var to = fmt.Sprintf("node-%s", typ)
					if _, ok := types[typ]; ok {
						fmt.Fprintf(out, "%s -> %q;\n", from, to)
					}
				}
			case *ast.ChanType:
				for i, typ := range dependants(t) {
					var from = fmt.Sprintf(`"node-%s":f%d`, snode.Id, i)
					var to = fmt.Sprintf("node-%s", typ)
					if _, ok := types[typ]; ok {
						fmt.Fprintf(out, "%s -> %q;\n", from, to)
					}
				}
			case *ast.InterfaceType:
				for i, f := range t.Methods.List {
					var from = fmt.Sprintf(`"node-%s":f%d`, snode.Id.Name, i)
					for _, typ := range dependants(f.Type) {
						var to = fmt.Sprintf("node-%s", typ)
						if _, ok := types[typ]; ok {
							fmt.Fprintf(out, "%s -> %q;\n", from, to)
						}
					}
				}
			case *ast.StructType:
				for i, f := range t.Fields.List {
					var from = fmt.Sprintf(`"node-%s":f%d`, snode.Id.Name, i)
					for _, typ := range dependants(f.Type) {
						var to = fmt.Sprintf("node-%s", typ)
						if _, ok := types[typ]; ok {
							fmt.Fprintf(out, "%s -> %q;\n", from, to)
						}
					}
				}
			}
		}

		fmt.Fprintf(out, "}\n")
	}
	fmt.Fprintf(out, "}\n\n")
}

func inspectDir(path string) map[string]map[string]NamedType {
	var (
		fset        = token.NewFileSet()
		filter      = func(n os.FileInfo) bool { return true }
		pkgmap, err = parser.ParseDir(fset, path, filter, 0)

		types = make(map[string]map[string]NamedType)
	)

	if err != nil {
		log.Fatal("parser error:", err) // parse error
	}

	for pkgName, pkg := range pkgmap {
		types[pkgName] = make(map[string]NamedType)

		for fname, f := range pkg.Files {
			fmt.Fprintln(os.Stderr, "File:", fname)

			ast.Inspect(f, func(n ast.Node) bool {
				switch nodeType := n.(type) {
				case *ast.TypeSpec:
					types[pkgName][nodeType.Name.Name] = NamedType{
						Id:   nodeType.Name,
						Type: nodeType.Type,
					}

					return false
				}

				return true
			})
		}

		// for n, _ := range pkg.Imports {
		// 	inspectDir(n)
		// }
	}

	return types
}

func typeName(n interface{}) string {
	switch t := n.(type) {
	case nil:
		return "nil"
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeName(t.X) + "." + typeName(t.Sel)
	case *ast.Object:
		return t.Name
	case *ast.StarExpr:
		return `\*` + typeName(t.X)
	case *ast.InterfaceType:
		// TODO:
		return `interface\{\}`
	case *ast.MapType:
		return `map\[` + typeName(t.Key) + `\]` + typeName(t.Value)
	case *ast.ChanType:
		return `chan\ ` + typeName(t.Value)
	case *ast.StructType:
		// TODO:
		return `struct\ \{\}` //+ typeName(t.)
	case *ast.Ellipsis:
		return `\.\.\.` + typeName(t.Elt)
	case *ast.Field:
		// ignoring names
		return typeName(t.Type)

	case *ast.FuncType:
		var buf bytes.Buffer
		fmt.Fprintf(&buf, `func\(`)
		if t.Params != nil && len(t.Params.List) > 0 {
			for i, p := range t.Params.List {
				if i > 0 {
					fmt.Fprintf(&buf, `\,`)
				}
				fmt.Fprintf(&buf, "%s", typeName(p))
			}
		}
		fmt.Fprintf(&buf, `\)`)

		if t.Results != nil && len(t.Results.List) > 0 {
			fmt.Fprintf(&buf, `\ \(`)
			for i, r := range t.Results.List {
				if i > 0 {
					fmt.Fprintf(&buf, `\,`)
				}
				fmt.Fprintf(&buf, "%s", typeName(r))
			}
			fmt.Fprintf(&buf, `\)`)
		}

		return buf.String()
	case *ast.ArrayType:
		return `\[\]` + typeName(t.Elt)
	default:
		return fmt.Sprintf("%#v", n)
	}
}

// collect all the types node n dependants on
func dependants(n interface{}) []string {
	switch t := n.(type) {
	case nil:
		return nil
	case *ast.Ident:
		return []string{t.Name}
	case *ast.SelectorExpr:
		// TODO: why t.X is an expression?
		return []string{typeName(t.X) + "." + t.Sel.Name}
	case *ast.Object:
		return []string{t.Name}
	case *ast.Field:
		return dependants(t.Type)
	case *ast.StarExpr:
		return dependants(t.X)
	case *ast.MapType:
		return append(dependants(t.Key), dependants(t.Value)...)
	case *ast.ChanType:
		return dependants(t.Value)
	case *ast.InterfaceType:
		if t.Methods == nil {
			return nil
		}
		var types []string
		for _, v := range t.Methods.List {
			types = append(types, dependants(v.Type)...)
		}
		return types
	case *ast.StructType:
		var types []string
		for _, v := range t.Fields.List {
			types = append(types, dependants(v.Type)...)
		}
		return types
	case *ast.FuncType:
		var types []string

		if t.Params != nil {
			for _, v := range t.Params.List {
				types = append(types, dependants(v.Type)...)
			}
		}

		if t.Results != nil {
			for _, v := range t.Results.List {
				types = append(types, dependants(v.Type)...)
			}
		}

		return types

	case *ast.ArrayType:
		return dependants(t.Elt)
	default:
		return []string{fmt.Sprintf("%#v", n)}
	}
}
