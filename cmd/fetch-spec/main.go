// running:
// curl https://golang.org/ref/spec | go run fetch-ebnf.go
//
// Inspired by: https://gist.github.com/chewxy/d9642a9552973dfc0731
//
package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/net/html"
)

func main() {
	log.SetOutput(os.Stderr)

	doc, err := html.Parse(os.Stdin)
	if err != nil {
		log.Fatalf("reading body: %s\n", err)
	}

	var printer = func(n *html.Node) {
		walk(n, func(nn *html.Node) bool {
			if nn.Type != html.TextNode {
				// continue as there could be more text nodes
				return true
			}

			fmt.Fprint(os.Stdout, nn.Data)
			// continue printing text
			return true
		})

		fmt.Fprint(os.Stdout, "\n")
	}

	walk(doc, func(n *html.Node) bool {
		if !(n.Type == html.ElementNode && n.Data == "pre") {
			return true
		}

		var isEbnf = false
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == "ebnf" {
				isEbnf = true
				break
			}
		}

		if !isEbnf {
			return true
		}

		printer(n.FirstChild)

		return false
	})
}

func walk(n *html.Node, f func(*html.Node) bool) {
	if n == nil {
		return
	}

	if f(n) {
		// Recursion
		// depth first
		walk(n.FirstChild, f)
	}

	// // then breadth
	walk(n.NextSibling, f)
}
