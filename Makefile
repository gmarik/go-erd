.PHONY: examples

GOROOT:=$(shell go env GOROOT)
examples_dir=./examples


examples:
	go run main.go -path ${GOROOT}/src/net/http/ |tee ${examples_dir}/net-http.dot |dot -Tsvg > ${examples_dir}/net-http.svg
	go run main.go -path ${GOROOT}/src/go/types/ |tee ${examples_dir}/go-types.dot |dot -Tsvg > ${examples_dir}/go-types.svg
	go run main.go -path ${GOROOT}/src/go/ast/ 	 |tee ${examples_dir}/go-ast.dot   |dot -Tsvg > ${examples_dir}/go-ast.svg
