# AST2LLM for Go 

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](LICENSE) [![License](https://img.shields.io/badge/License-Apache-blue.svg)](LICENSE) [![Build](https://img.shields.io/badge/Build-Passing-brightgreen)](LICENSE) [![Build Status](https://github.com/vprud/ast2llm-go/actions/workflows/go.yml/badge.svg)](https://github.com/vprud/ast2llm-go/actions/workflows/go.yml)

**Local AST-powered context enhancement tool for LLM**  

Automatically injects relevant code structure into your prompts using precise Go AST analysis.

![Demo](assets/demo.gif)

Our MCP server provides the `parse-go` tool that:

1. Analyzes your Go project structure
2. Identifies external type declarations
3. Packages context for LLM prompts
4. Delivers 3-5x faster context resolution than grap-based approaches

For example, the demo above shows the model calling the tool to get the missing information about the fields of the `MyDTO` structure. In response, the model received all the necessary information from the response:

```
Used Imported Structs (from this project, if available):
  Struct: testme/dto.MyDTO
    Fields:
      - Foo string
      - Bar int
```

## Requirements

- Go 1.24 or higher (if building from source)
- Supported MCP client (Cursor, Claude, VS Code, etc.)

## Installation

### Binaries

Download the latest release:

```bash
curl -L https://github.com/vlad/ast2llm-go/releases/latest/download/ast2llm-go -o /usr/local/bin/ast2llm-go
chmod +x /usr/local/bin/ast2llm-go
```

### Source Code

```bash
git clone https://github.com/vlad/ast2llm-go.git
cd ast2llm-go
make build
```

### Docker

```bash
docker run -d --name ast2llm-go \
  -v /path/to/your/code:/code \
  ghcr.io/vlad/ast2llm-go
```

## Setup in Clients

### Cursor

Add to your `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "go-ast": {
      "command": "/path/to/ast2llm-go",
      "args": []
    }
  }  
}
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "go-ast": {
      "command": "/path/to/ast2llm-go"
    }
  }
}
```

### Visual Studio Code

Add to your VS Code MCP config:

```json
{
  "servers": {
    "go-ast": {
      "type": "stdio",
      "command": "/path/to/ast2llm-go"
    }
  }
}
```

### Using Docker

For clients that support Docker-based MCP servers:

```json
{
  "mcpServers": {
    "go-ast": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-v",
        "/path/to/your/code:/code",
        "ghcr.io/vlad/ast2llm-go"
      ]
    }
  }
}
```

## Note About Current State
This MCP server is under active development and may have stability issues or incomplete functionality. We're working hard to improve it, but you might encounter:

- Occasional parsing errors
- Limited type support in current version
- Performance bottlenecks with large codebases

**Found an issue?**  
[Open a GitHub Issue](https://github.com/vprud/ast2llm-go/issues/new) to help us improve! We appreciate all bug reports and feature requests.

## Roadmap

### Language Support
- [x] Support for struct types
- [ ] Support for interface types
- [ ] Support for function types
- [ ] Support for global variables

### Multi-file Context
- [ ] Analyze multiple open files simultaneously
- [x] Cross-file dependency resolution
- [ ] Context-aware import optimization

### AST Representation
- [ ] Improved type hierarchy visualization
- [ ] Research optimal AST representation for LLMs

### Performance
- [ ] Incremental parsing
- [ ] AST caching
- [ ] Parallel analysis

## Contributing

We welcome contributions! Here's how you can help:

1. **Report Bugs**
   - Open an issue with a clear description
   - Include steps to reproduce
   - Add relevant logs/screenshots

2. **Suggest Features**
   - Open an issue with the feature request
   - Explain the use case and benefits
   - Include any relevant examples

3. **Submit Pull Requests**
   - Fork the repository
   - Create a feature branch
   - Add tests for new functionality
   - Ensure all tests pass
   - Submit a PR with a clear description
