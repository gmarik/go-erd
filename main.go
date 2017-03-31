package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

//
// from: https://github.com/golang/example/tree/master/gotypes#typeandvalue
//
// running: go run ./cmd/goerd/main.go -path cmd/traverse/|dot -Tsvg > out.svg
func main() {
	var (
		path = flag.String("path", "", "path parse")
	)

	flag.Parse()

	if *path == "" {
		flag.Usage()
		os.Exit(1)
	}

	dotRender(os.Stdout, inspectDir(*path))
}

type namedType struct {
	Ident *ast.Ident
	Type  ast.Expr
}

func dotRender(out *os.File, pkgTypes map[string]map[string]namedType) {
	fmt.Fprintf(out, "digraph %q { \n", "GoERD")

	var buf bytes.Buffer
	for pkg, types := range pkgTypes {

		fmt.Fprintf(out, "subgraph %q {\n", pkg)
		fmt.Fprintf(out, "label=%q;\n", pkg)

		// Nodes
		var i int
		for _, typ := range types {
			i++
			buf.Reset()

			switch t := typ.Type.(type) {
			case *ast.Ident:
				var label = fmt.Sprintf(`%s %s`, typ.Ident.Name, t.Name)
				fmt.Fprintf(out, " \"node-%s\" [shape=ellipse,label=\"%s\"];\n", typ.Ident.Name, escape(label))
			case *ast.SelectorExpr:
				var label = fmt.Sprintf(`%s %s`, typ.Ident.Name, toString(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=ellipse,label=\"%s\"];\n", typ.Ident.Name, escape(label))
			case *ast.ChanType:
				var label = fmt.Sprintf(`%s %s`, typ.Ident.Name, toString(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=box,label=\"%s\"];\n", typ.Ident.Name, escape(label))
			case *ast.FuncType:
				var label = fmt.Sprintf(`%s %s`, typ.Ident.Name, toString(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=rectangle,label=\"%s\"];\n", typ.Ident.Name, escape(label))
			case *ast.ArrayType:
				var label = fmt.Sprintf(`%s %s`, typ.Ident.Name, toString(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=rectangle,label=\"%s\"];\n", typ.Ident.Name, escape(label))
			case *ast.MapType:
				var label = fmt.Sprintf(`%s %s`, typ.Ident.Name, toString(t))
				fmt.Fprintf(out, " \"node-%s\" [shape=rectangle,label=\"%s\"];\n", typ.Ident.Name, escape(label))
			case *ast.InterfaceType:
				fmt.Fprintf(&buf, `%s interface|`, typ.Ident.Name)
				for i, f := range t.Methods.List {
					if i > 0 {
						fmt.Fprintf(&buf, `|`)
					}
					fmt.Fprintf(&buf, `<f%d>`, i)
					// a,b,c Type
					for ii, n := range f.Names {
						fmt.Fprintf(&buf, "%s", n.Name)
						if ii > 0 {
							fmt.Fprintf(&buf, `,`)
						}
					}
					if len(f.Names) > 0 {
						fmt.Fprintf(&buf, ` `)
					}
					fmt.Fprintf(&buf, `%s`, toString(f.Type))
				}
				fmt.Fprintf(out, " \"node-%s\" [shape=Mrecord,label=\"{%s}\"];\n", typ.Ident.Name, escape(buf.String()))
			case *ast.StructType:
				fmt.Fprintf(&buf, `%s|`, typ.Ident.Name)
				for i, f := range t.Fields.List {
					if i > 0 {
						fmt.Fprintf(&buf, "|")
					}
					fmt.Fprintf(&buf, `<f%d>`, i)

					for ii, n := range f.Names {
						if ii > 0 {
							fmt.Fprintf(&buf, `, `)
						}
						fmt.Fprintf(&buf, `%s`, n.Name)
					}
					if len(f.Names) > 0 {
						fmt.Fprintf(&buf, ` `)
					}
					fmt.Fprintf(&buf, `%s`, toString(f.Type))
				}
				fmt.Fprintf(out, " \"node-%s\" [shape=record,label=\"{%s}\"];\n", typ.Ident.Name, escape(buf.String()))
			default:
				fmt.Fprintf(os.Stderr, "MISSED: %s: %#v\n ", toString(t), typ)
			}
		}

		// Edges
		for _, ptype := range types {
			switch t := ptype.Type.(type) {
			// TODO: exhaustive switch
			case *ast.FuncType:
				for i, typ := range dependsOn(t) {
					var from = fmt.Sprintf(`"node-%s":f%d`, ptype.Ident.Name, i)
					var to = fmt.Sprintf("node-%s", typ)
					if _, ok := types[typ]; ok {
						fmt.Fprintf(out, "%s -> %q;\n", from, to)
					}
				}
			case *ast.ChanType:
				for i, typ := range dependsOn(t) {
					var from = fmt.Sprintf(`"node-%s":f%d`, ptype.Ident.Name, i)
					var to = fmt.Sprintf("node-%s", typ)
					if _, ok := types[typ]; ok {
						fmt.Fprintf(out, "%s -> %q;\n", from, to)
					}
				}
			case *ast.InterfaceType:
				for i, f := range t.Methods.List {
					var from = fmt.Sprintf(`"node-%s":f%d`, ptype.Ident.Name, i)
					for _, typ := range dependsOn(f.Type) {
						var to = fmt.Sprintf("node-%s", typ)
						if _, ok := types[typ]; ok {
							fmt.Fprintf(out, "%s -> %q;\n", from, to)
						}
					}
				}
			case *ast.StructType:
				for i, f := range t.Fields.List {
					var from = fmt.Sprintf(`"node-%s":f%d`, ptype.Ident.Name, i)
					for _, typ := range dependsOn(f.Type) {
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

func inspectDir(path string) map[string]map[string]namedType {
	var (
		fset        = token.NewFileSet()
		filter      = func(n os.FileInfo) bool { return true }
		pkgmap, err = parser.ParseDir(fset, path, filter, 0)

		types = make(map[string]map[string]namedType)
	)

	if err != nil {
		log.Fatal("parser error:", err)
	}

	for pkgName, pkg := range pkgmap {
		types[pkgName] = make(map[string]namedType)

		for fname, f := range pkg.Files {
			fmt.Fprintln(os.Stderr, "File:", fname)

			ast.Inspect(f, func(n ast.Node) bool {
				switch nodeType := n.(type) {
				// skip comments
				case *ast.CommentGroup, *ast.Comment:
					return false
				case *ast.TypeSpec:
					types[pkgName][nodeType.Name.Name] = namedType{
						Ident: nodeType.Name,
						Type:  nodeType.Type,
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

func escape(s string) string {
	for _, ch := range " '`[]{}()*" {
		s = strings.Replace(s, string(ch), `\`+string(ch), -1)
	}

	return s
}

func toString(n interface{}) string {
	switch t := n.(type) {
	case nil:
		return "nil"
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return toString(t.X) + "." + toString(t.Sel)
	case *ast.Object:
		return t.Name
	case *ast.StarExpr:
		return `*` + toString(t.X)
	case *ast.InterfaceType:
		// TODO:
		return `interface{}`
	case *ast.MapType:
		return `map[` + toString(t.Key) + `]` + toString(t.Value)
	case *ast.ChanType:
		return `chan ` + toString(t.Value)
	case *ast.StructType:
		// TODO:
		return `struct {}` //+ toString(t.)
	case *ast.Ellipsis:
		return `...` + toString(t.Elt)
	case *ast.Field:
		// ignoring names
		return toString(t.Type)

	case *ast.FuncType:
		var buf bytes.Buffer
		fmt.Fprint(&buf, `func(`)
		if t.Params != nil && len(t.Params.List) > 0 {
			for i, p := range t.Params.List {
				if i > 0 {
					fmt.Fprint(&buf, `, `)
				}
				fmt.Fprint(&buf, toString(p))
			}
		}
		fmt.Fprint(&buf, `)`)

		if t.Results != nil && len(t.Results.List) > 0 {
			fmt.Fprint(&buf, ` (`)
			for i, r := range t.Results.List {
				if i > 0 {
					fmt.Fprint(&buf, `, `)
				}
				fmt.Fprint(&buf, toString(r))
			}
			fmt.Fprint(&buf, `)`)
		}

		return buf.String()
	case *ast.ArrayType:
		return `[]` + toString(t.Elt)
	default:
		return fmt.Sprintf("%#v", n)
	}
}

// collect all the type names node n depends on
func dependsOn(n interface{}) []string {
	switch t := n.(type) {
	case nil:
		return nil
	case *ast.Ident:
		return []string{t.Name}
	case *ast.SelectorExpr:
		return []string{toString(t.X) + "." + t.Sel.Name}
	case *ast.Object:
		return []string{t.Name}
	case *ast.Field:
		return dependsOn(t.Type)
	case *ast.StarExpr:
		return dependsOn(t.X)
	case *ast.MapType:
		return append(dependsOn(t.Key), dependsOn(t.Value)...)
	case *ast.ChanType:
		return dependsOn(t.Value)
	case *ast.InterfaceType:
		if t.Methods == nil {
			return nil
		}
		var types []string
		for _, v := range t.Methods.List {
			types = append(types, dependsOn(v.Type)...)
		}
		return types
	case *ast.StructType:
		var types []string
		for _, v := range t.Fields.List {
			types = append(types, dependsOn(v.Type)...)
		}
		return types
	case *ast.FuncType:
		var types []string

		if t.Params != nil {
			for _, v := range t.Params.List {
				types = append(types, dependsOn(v.Type)...)
			}
		}

		if t.Results != nil {
			for _, v := range t.Results.List {
				types = append(types, dependsOn(v.Type)...)
			}
		}

		return types

	case *ast.ArrayType:
		return dependsOn(t.Elt)
	default:
		return []string{fmt.Sprintf("%#v", n)}
	}
}
