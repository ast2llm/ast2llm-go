package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/vlad/ast2llm-go/internal/parser"
)

func main() {
	filePath := flag.String("file", "", "Path to the Go file to parse")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("Please provide a file path using -file flag")
	}

	// Read file content
	content, err := ioutil.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Create parser and parse file
	p := parser.New()
	file, err := p.Parse(*filePath, content)
	if err != nil {
		log.Fatalf("Error parsing file: %v", err)
	}

	// Extract dependencies
	info := p.ExtractDeps(file)

	// Print results
	fmt.Printf("Package: %s\n", info.PackageName)
	fmt.Printf("Imports: %v\n", info.Imports)
	fmt.Printf("Functions: %v\n", info.Functions)
}
