package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/vlad/ast2llm-go/internal/parser"
	"github.com/vlad/ast2llm-go/internal/types"
)

func main() {
	// Define flags
	filePath := flag.String("file", "", "Analyze single file")
	projectPath := flag.String("project", "", "Analyze entire project")
	jsonOutput := flag.Bool("json", false, "Enable JSON output")

	// Parse flags
	flag.Parse()

	// Get the actual file path from the flag value
	file := *filePath
	if file == "" && flag.NArg() > 0 {
		file = flag.Arg(0)
	}

	p := parser.New()

	switch {
	case file != "":
		analyzeFile(p, file, *jsonOutput)
	case *projectPath != "":
		analyzeProject(p, *projectPath, *jsonOutput)
	default:
		color.Red("Error: specify --file or --project flag")
		flag.Usage()
		os.Exit(1)
	}
}

// Cache for BuildGraph results
var (
	graphCache     = make(map[string]*types.DependencyGraph)
	graphCacheLock sync.RWMutex
	cacheTimeout   = 5 * time.Minute
)

func analyzeFile(p *parser.FileParser, path string, jsonOut bool) {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		color.Red("Error resolving path: %v", err)
		return
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		color.Red("Error: file %s does not exist", absPath)
		return
	}

	src, err := os.ReadFile(absPath)
	if err != nil {
		color.Red("Error reading file: %v", err)
		return
	}

	file, err := p.Parse(absPath, src)
	if err != nil {
		color.Red("Error parsing file: %v", err)
		return
	}

	deps := p.ExtractDeps(file)
	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(deps)
	} else {
		color.Green("Dependencies for %s:", absPath)
		for _, dep := range deps {
			fmt.Println("-", dep)
		}
	}
}

func analyzeProject(p *parser.FileParser, path string, jsonOut bool) {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		color.Red("Error resolving path: %v", err)
		return
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		color.Red("Error: directory %s does not exist", absPath)
		return
	}

	// Check cache first
	graphCacheLock.RLock()
	if cached, ok := graphCache[absPath]; ok {
		graphCacheLock.RUnlock()
		if jsonOut {
			json.NewEncoder(os.Stdout).Encode(cached)
		} else {
			printGraph(cached)
		}
		return
	}
	graphCacheLock.RUnlock()

	// Create progress bar
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Analyzing project..."),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Start progress bar
	go func() {
		for {
			bar.Add(1)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Build graph
	graph, err := p.BuildGraph(absPath)
	if err != nil {
		bar.Finish()
		color.Red("Error building graph: %v", err)
		return
	}

	// Stop progress bar
	bar.Finish()

	// Cache the result
	graphCacheLock.Lock()
	graphCache[absPath] = graph
	graphCacheLock.Unlock()

	// Start cache cleanup timer
	go func() {
		time.Sleep(cacheTimeout)
		graphCacheLock.Lock()
		delete(graphCache, absPath)
		graphCacheLock.Unlock()
	}()

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(graph)
	} else {
		printGraph(graph)
	}
}

func printGraph(graph *types.DependencyGraph) {
	color.Cyan("Dependency graph:")
	for pkg, node := range graph.Nodes {
		fmt.Printf("%s:\n", color.YellowString(pkg))
		fmt.Printf("  Functions: %v\n", node.Functions)
		fmt.Printf("  Depends on: %v\n", node.DependsOn)
		fmt.Printf("  Files: %v\n", node.Files)
	}
}
