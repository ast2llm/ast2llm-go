package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/vlad/ast2llm-go/internal/parser"
	ourtypes "github.com/vlad/ast2llm-go/internal/types" // Alias ourtypes
)

func main() {
	// Define flags
	projectPath := flag.String("project", "", "Analyze entire project")
	jsonOutput := flag.Bool("json", false, "Enable JSON output")

	// Parse flags
	flag.Parse()

	p := parser.New()

	switch {
	case *projectPath != "":
		analyzeProject(p, *projectPath, *jsonOutput)
	default:
		color.Red("Error: specify --project flag")
		flag.Usage()
		os.Exit(1)
	}
}

// Cache for FileInfo results
var (
	fileInfoCache     = make(map[string]map[string]*ourtypes.FileInfo) // Cache now stores a map of fileInfos
	fileInfoCacheLock sync.RWMutex
	cacheTimeout      = 5 * time.Minute
)

func analyzeProject(p *parser.ProjectParser, path string, jsonOut bool) {
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
	fileInfoCacheLock.RLock()
	if cached, ok := fileInfoCache[absPath]; ok {
		fileInfoCacheLock.RUnlock()
		if jsonOut {
			json.NewEncoder(os.Stdout).Encode(cached)
		} else {
			printProjectFileInfo(cached)
		}
		return
	}
	fileInfoCacheLock.RUnlock()

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

	// Parse project
	fileInfos, err := p.ParseProject(absPath)
	if err != nil {
		bar.Finish()
		color.Red("Error parsing project: %v", err)
		return
	}

	// Stop progress bar
	bar.Finish()

	// Cache the result
	fileInfoCacheLock.Lock()
	fileInfoCache[absPath] = fileInfos
	fileInfoCacheLock.Unlock()

	// Start cache cleanup timer
	go func() {
		time.Sleep(cacheTimeout)
		fileInfoCacheLock.Lock()
		delete(fileInfoCache, absPath)
		fileInfoCacheLock.Unlock()
	}()

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(fileInfos)
	} else {
		printProjectFileInfo(fileInfos)
	}
}

func printProjectFileInfo(fileInfos map[string]*ourtypes.FileInfo) {
	color.Cyan("Project Information:")

	for filePath, fileInfo := range fileInfos {
		fmt.Printf("\n--- File: %s ---\n", color.YellowString(filePath))
		fmt.Printf("  Package Name: %s\n", fileInfo.PackageName)

		fmt.Printf("  Imports:\n")
		if len(fileInfo.Imports) == 0 {
			fmt.Println("    (None)")
		} else {
			for _, imp := range fileInfo.Imports {
				fmt.Printf("    - %s\n", imp)
			}
		}

		fmt.Printf("  Functions:\n")
		if len(fileInfo.Functions) == 0 {
			fmt.Println("    (None)")
		} else {
			for _, fn := range fileInfo.Functions {
				fmt.Printf("    - %s\n", fn)
			}
		}

		fmt.Printf("  Local Structs:\n")
		if len(fileInfo.Structs) == 0 {
			fmt.Println("    (None)")
		} else {
			for _, s := range fileInfo.Structs {
				fmt.Printf("    - %s (Comment: %q)\n", color.MagentaString(s.Name), s.Comment)
				if len(s.Fields) > 0 {
					fmt.Println("      Fields:")
					for _, f := range s.Fields {
						fmt.Printf("        - %s %s\n", f.Name, f.Type)
					}
				}
				if len(s.Methods) > 0 {
					fmt.Println("      Methods:")
					for _, m := range s.Methods {
						fmt.Printf("        - %s(%s) (%s) (Comment: %q)\n", m.Name, strings.Join(m.Parameters, ", "), strings.Join(m.ReturnTypes, ", "), m.Comment)
					}
				}
			}
		}

		fmt.Printf("  Used Imported Structs:\n")
		if len(fileInfo.UsedImportedStructs) == 0 {
			fmt.Println("    (None)")
		} else {
			for _, s := range fileInfo.UsedImportedStructs {
				fmt.Printf("    - %s\n", s.Name)
			}
		}
	}
}
