# 🚀 AST2LLM-Go | [![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](LICENSE) [![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE) [![Build](https://img.shields.io/badge/Build-Passing-brightgreen)](LICENSE)

**Local AST-powered code enhancement for LLMs**  
Automatically enriches your code prompts with deep structural understanding through Abstract Syntax Tree (AST) analysis.

## ✨ Features

- 🧠 **Deep Code Understanding** - Analyzes Go code structure, dependencies, and relationships
- 🔒 **100% Local** - All processing happens on your machine, no code leaves your environment
- ⚡ **LLM Integration** - Seamless integration with your development workflow
- 📊 **Dependency Analysis** - Maps function calls, imports, and package relationships
- 🛠 **Code Enhancement** - Improves code quality with better documentation and error handling
- 🔍 **Symbol Focus** - Prioritize specific symbols in context for targeted improvements

## 🚀 Quick Start

1. **Install the binary**
   ```bash
   curl -L https://github.com/vlad/ast2llm-go/releases/latest/download/ast2llm-go -o /usr/local/bin/ast2llm-go
   chmod +x /usr/local/bin/ast2llm-go
   ```

2. **Run the server**
   ```bash
   ast2llm-go --daemon
   ```

3. **Use in your IDE**
   - Install the AST2LLM extension for your IDE
   - Configure the extension to use the local server
   - Start enhancing your code prompts!

## 💡 Usage Example

TBD

## 🚧 Current Status (MVP)

### ✅ Working Today
- Go language support with full AST analysis
- Basic dependency graph generation
- Function and package relationship mapping
- Local server implementation
- Code enhancement capabilities

### 🗓 Roadmap
- [ ] Python/TypeScript support
- [ ] Multi-file context analysis
- [ ] Advanced code refactoring suggestions
- [ ] Performance optimizations

## 🤝 Contributing

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
