# Cargo - Batch Processing for Go

A high-performance batch processing library for Go that automatically batches items based on size or timeout, with MongoDB integration example.

## Features

- **Automatic batching** - Flushes based on batch size or timeout
- **Concurrent workers** - Process items with multiple goroutines
- **Thread-safe** - Safe for concurrent use
- **Flexible handlers** - Custom batch processing logic
- **MongoDB example** - Ready-to-use database integration

## Quick Start

### Using Docker

```bash
# Start MongoDB and run the app
docker compose up --build

# Run in detached mode
docker compose up -d

# View logs
docker compose logs -f app

# Stop everything
docker compose down
```

### Local Development

```bash
# Build
go build -o app ./cmd/main.go

# Run with custom parameters
./app -count=1000000 -batch-size=10000 -workers=6 -timeout=5s

# With custom MongoDB URI
./app -mongo-uri="mongodb://localhost:27017"
```

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-count` | 100 | Number of items to generate |
| `-batch-size` | 10 | Batch size for flushing |
| `-timeout` | 2s | Flush timeout duration |
| `-workers` | 4 | Number of worker goroutines |
| `-mongo-uri` | mongodb://mongodb:27017 | MongoDB connection URI |

## Usage Example

```go
// Create a handler function
handler := func(ctx context.Context, batch []any) error {
    // Process your batch
    _, err := collection.InsertMany(ctx, batch)
    return err
}

// Initialize cargo
c, err := cargo.NewCargo(batchSize, timeout, handler)
if err != nil {
    log.Fatal(err)
}
defer c.Close()

// Add items (thread-safe)
for item := range items {
    c.Add(item)
}
```

## Architecture

```
Generator → Workers → Cargo → MongoDB
   (1M)      (6)    (10K batch)  (cargodb)
```

1. **Generator** - Creates random items
2. **Workers** - Concurrent goroutines add items to cargo
3. **Cargo** - Batches items and flushes on size/timeout
4. **MongoDB** - Stores batched data

## Performance

Inserting 1 million documents:
- **Generation**: ~300ms
- **Total time**: ~2.4s
- **Throughput**: ~416K docs/sec

## Project Structure

```
.
├── cargo/          # Batch processing package
├── cmd/            # Application entry point
├── generator/      # Random data generator
├── mongodb/        # MongoDB client & repository
├── Dockerfile      # Container image
└── docker-compose.yml
```

## Verify Data

```bash
# Connect to MongoDB
docker compose exec mongodb mongosh

# Check inserted data
use cargodb
db.items.countDocuments()
db.items.findOne()
```
