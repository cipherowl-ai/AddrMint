# AddrMint

## Overview

AddrMint is a high-performance tool for generating large quantities of blockchain addresses (Ethereum, Bitcoin, and Solana). It features advanced concurrency optimizations for maximum throughput when generating millions or billions of addresses.

In CipherOwl we use this tool to generate addresses to test various of data workloads.

## Building

### Using the Makefile (Recommended)

The project includes a comprehensive Makefile to simplify building and managing the application:

```
# Build the application
make build

# Build with optimizations for production
make build-prod

# Cross-compile for multiple platforms (Linux, Windows, macOS)
make build-all

# Clean build artifacts
make clean

# Install dependencies
make deps

# Run tests
make test

# Run a continuous integration pipeline
make ci

# See all available commands
make help
```

The built binary will be available in the `build/` directory.

### Manual Build

```
go build -o addrmint main.go
```

## Usage

```
./addrmint --network [ethereum|bitcoin|solana] --count [number] --seed [optional_integer_seed] --workers [optional_worker_count] --batch-size [optional_batch_size] --output-buffer [optional_buffer_size] --output [optional_output_file] --generate-hash
```

### Parameters

- `--network`: The blockchain network (ethereum, bitcoin, or solana) (required)
- `--count`: Number of addresses to generate (default: 1)
- `--seed`: Random seed as an integer (default: 0, which generates a random seed)
- `--workers`: Number of concurrent workers (default: number of CPU cores)
- `--batch-size`: Number of addresses to batch before reporting progress (default: 1000)
- `--output-buffer`: Size of the output buffer for better throughput (default: 10000)
- `--output`: File path to save generated addresses (default: stdout)
- `--generate-hash`: Prefix each address with a SHA-256 hash (first 6 characters) and comma (default: false)

### Examples

Generate 10 Ethereum addresses:
```
./addrmint --network ethereum --count 10
```

Generate 1000 Bitcoin addresses with a specific seed and save to a file:
```
./addrmint --network bitcoin --count 1000 --seed 12345 --output bitcoin-addresses.txt
```

Generate 5 Solana addresses:
```
./addrmint --network solana --count 5
```

Generate 1 million Ethereum addresses using 16 workers and a large output buffer:
```
./addrmint --network ethereum --count 1000000 --workers 16 --output-buffer 50000 --output ethereum-addresses.txt
```

Generate 10 Ethereum addresses with hash prefixes:
```
./addrmint --network ethereum --count 10 --generate-hash
```

The same seed will always produce the same addresses:
```
./addrmint --network ethereum --count 5 --seed 42
```

## Performance Optimization

The tool is highly optimized for maximum throughput:

- Adaptive worker pool sizing based on the number of addresses to generate
- Memory pooling to reduce GC pressure during large generation tasks
- Thread-safe result collection with mutex-protected access
- Optimized channel buffer sizes for maximum throughput
- Efficient ordering of outputs while maintaining high throughput
- Visual progress bar for real-time generation tracking

Performance examples:
- 1 million Ethereum addresses: ~2-3 minutes on a standard 8-core CPU (up to 50% faster than v1)
- 1 billion addresses: Efficiently processes at maximum hardware throughput

## Features

- **Reproducible Generation**: Using the same seed always produces identical addresses
- **Visual Progress Bar**: Real-time progress indication for large generation tasks
- **File Output**: Direct output to file with the `--output` parameter
- **Hash Prefixing**: Option to prefix each address with a short SHA-256 hash using `--generate-hash`
- **Concurrent Generation**: Efficiently utilizes all available CPU cores
- **Memory Efficient**: Designed to handle extremely large generation tasks with minimal memory usage
- **Cross-Platform**: Builds available for Linux, Windows, and macOS (via `make build-all`)
- **Well Tested**: Comprehensive unit tests ensure reliability
- **CI Integration**: Ready for continuous integration pipelines

## Development

The project includes a Makefile with several useful targets for development:

```
# Format code
make fmt

# Run tests
make test

# Check for lint errors
make lint

# Run the complete CI pipeline
make ci

# Generate example outputs
make examples
```

## Testing

The project includes comprehensive unit tests for all core functionality. Run tests with:

```
make test
```

For continuous integration, use the combined target that runs dependencies verification, formatting, building, testing and linting:

```
make ci
```

## Notes

- If seed is 0 or not provided, a random seed will be generated
- Using a specific integer seed ensures reproducible address generation
- Progress information and visual bar are displayed on stderr
- Address output can be directed to a file using the `--output` parameter
- For generating billions of addresses, increase the output buffer size: `--output-buffer 100000`
- When using `--generate-hash`, each address is prefixed with a 6-character SHA-256 hash and a comma
