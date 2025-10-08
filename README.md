# AI Code Review Tool

A command-line tool that analyzes Go code files and provides intelligent code review suggestions using AI.

## Features

- 🔍 **Automated Go code analysis** - Scans directories for Go files
- 🤖 **AI-powered reviews** - Uses configurable AI models for code analysis
- 🚀 **Concurrent processing** - Reviews multiple files simultaneously
- 📊 **Detailed reporting** - Provides security, performance, and architecture recommendations
- ⚙️ **Configurable** - Customizable API endpoints, models, and processing limits

## Installation

```bash
go mod tidy
go build -o aireview .
```

## Usage

### Basic usage
```bash
./aireview --path ./my-project
```

### Advanced options
```bash
./aireview \
  --path ./my-project \
  --url https://api.openai.com/v1/chat/completions \
  --api-key sk-your-openai-key \
  --model gpt-4 \
  --max-size 2097152 \
  --concurrency 5
```

```bash
# Multiple local LM Studio hosts (round-robin + failover)
./aireview \
  --path ./my-project \
  --model llama2 \
  --urls http://127.0.0.1:1234/v1/chat/completions,http://127.0.0.1:1235/v1/chat/completions \
  --concurrency 8
```

### Write report to a file

To keep logs on stdout/stderr and write the review content to a file:

```bash
./aireview --path ./my-project --report-file ./review.md
```

### Using environment variables (recommended for API keys)
```bash
export AIREVIEW_API_KEY="sk-your-openai-key"
./aireview --path ./my-project --url https://api.openai.com/v1/chat/completions --model gpt-4
```

### Command-line options

- `--path, -p`: Path to the project directory for review (default: ".")
- `--url, -u`: URL to the AI API endpoint (default: "http://127.0.0.1:1234/v1/chat/completions")
- `--urls`: Comma-separated list of AI API endpoints (overrides `--url`), used in round-robin for parallelism and automatic failover
- `--api-key, -k`: API key for authentication (can also use AIREVIEW_API_KEY env var)
- `--model, -m`: AI model to use for code review (default: "devstral-small-2507-mlx")
- `--max-size`: Maximum file size in bytes to process (default: 10485760)
- `--concurrency, -c`: Maximum number of concurrent reviews (default: 10)
- `--report-file`: Path to write the review report (Markdown). If empty, the report is printed to stdout.

## Architecture

The tool is organized into several packages:

- `cmd/` - CLI command structure using Cobra
- `internal/config/` - Configuration management and validation
- `internal/reviewer/` - AI API integration and review logic
- `internal/scanner/` - File system scanning and filtering

## Security Features

- File size limits to prevent resource exhaustion
- Path validation to prevent directory traversal attacks
- HTTP timeout configuration
- Concurrent processing limits
- Input validation and error handling
- Secure API key handling via environment variables
- Automatic detection of online services requiring authentication

### API Key Security

**⚠️ Important:** Never hardcode API keys in your code or commit them to version control.

**Recommended approach:**
```bash
# Set environment variable (recommended)
export AIREVIEW_API_KEY="your-api-key-here"
./aireview --path ./project
```

**Alternative approaches:**
```bash
# Pass via command line (less secure - visible in process list)
./aireview --api-key "your-api-key" --path ./project

# Read from file (ensure file has proper permissions)
echo "your-api-key" > ~/.aireview_key
chmod 600 ~/.aireview_key
export AIREVIEW_API_KEY=$(cat ~/.aireview_key)
```

## Requirements

- Go 1.21 or later
- Access to an AI API endpoint (OpenAI-compatible)

## API Compatibility

The tool is designed to work with OpenAI-compatible APIs, including:
- OpenAI GPT models
- Local LLM servers (like LM Studio, Ollama)
- Azure OpenAI Service
- Other OpenAI-compatible endpoints

## Troubleshooting

### Common API Errors

**404 Model Not Found**
```
model not found (404): check if model 'gpt-4' exists and you have access to it
```
- **Solution**: Use current model names like `gpt-4o-mini`, `gpt-3.5-turbo`, or `gpt-4o`
- **Check**: Your OpenAI account has access to the requested model

**401 Authentication Failed**
```
authentication failed (401): check your API key
```
- **Solution**: Verify your API key is correct and active
- **Check**: API key format should start with `sk-` for OpenAI

**429 Rate Limit Exceeded**
```
rate limit exceeded (429): too many requests, please wait and try again
```
- **Solution**: Reduce `--concurrency` parameter (default: 10)
- **Wait**: Rate limits reset over time, try again later
- **Upgrade**: Consider upgrading your OpenAI plan for higher limits

### Recommended Models

**OpenAI (current models as of 2024)**:
```bash
# Fast and cost-effective
./aireview --model gpt-4o-mini --url https://api.openai.com/v1/chat/completions

# High quality
./aireview --model gpt-4o --url https://api.openai.com/v1/chat/completions

# Legacy (if available)
./aireview --model gpt-3.5-turbo --url https://api.openai.com/v1/chat/completions
```

**Local models**:
```bash
# LM Studio, Ollama, or other local servers
./aireview --model llama2 --url http://localhost:1234/v1/chat/completions

# Multiple LM Studio hosts (round-robin across endpoints)
./aireview --model llama2 \
  --urls http://localhost:1234/v1/chat/completions,http://localhost:1235/v1/chat/completions \
  --concurrency 8
```

### Performance Tips

- **Balance concurrency with hosts**: For multiple local hosts, set `--concurrency` around `hosts × cores_per_host` for good throughput
- **Reduce concurrency** for rate-limited APIs: `--concurrency 1`
- **Limit file size** for faster processing: `--max-size 512000` (500KB)
- **Use environment variables** for API keys to avoid exposing them in command history

### Multi-host behavior

- **Round-robin dispatch**: Each review request is sent to the next endpoint in `--urls`.
- **Automatic failover**: If an endpoint returns an error or is unavailable, the tool retries the request on the next endpoint until one succeeds or all fail.
