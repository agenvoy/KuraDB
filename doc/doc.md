# KuraDB - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.25 or higher
- OpenAI API Key (for embedding vector generation)
- macOS or Linux (relies on POSIX filesystem semantics)

## Installation

### Using go install

```bash
go install github.com/agenvoy/kuradb/cmd/app@latest
```

### Build from Source

```bash
git clone https://github.com/pardnchiu/KuraDB.git
cd KuraDB
make build
# binary output to bin/kura
```

### Using Makefile

```bash
# Build and start
make app

# Add a database
make add name=my_docs

# List registered databases
make list

# Remove a database
make remove name=my_docs

# Rename a database
make edit old=my_docs new=my_archive
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | Yes | OpenAI API key for the text-embedding-3-small model |

KuraDB reads `OPENAI_API_KEY` from the system keychain. Ensure it is set before starting.

### Config Directory

All data is stored under `~/.config/kuradb/`:

| Path | Purpose |
|------|---------|
| `db.json` | Database registry (JSON) |
| `global.db` | Global SQLite (query cache) |
| `{name}/data.db` | Per-database SQLite (file_data) |
| `{name}/inbox/` | Watched directory — drop files here for auto-indexing |
| `{name}/record.json` | Filesystem snapshot for change detection |
| `endpoint` | Written with the HTTP server address |

Each database gets a symlink at `~/Kura_{name}` → `~/.config/kuradb/{name}/inbox/` for easy drag-and-drop.

## Usage

### Start the Server

```bash
kura
```

On startup the server:
1. Loads all registered databases
2. Rebuilds the vector cache from SQLite
3. Starts the file watcher (polls every 10 seconds)
4. Starts the embedding scheduler (processes one batch every 5 seconds, up to 64 chunks per batch)
5. Starts the HTTP API on a random port, writing the address to `~/.config/kuradb/endpoint`

### Manage Databases

```bash
# Add a database
kura add my_docs

# List all databases
kura list

# Rename
kura edit my_docs my_archive

# Remove (requires typing 'yes' to confirm)
kura remove my_archive
```

### Query API

All endpoints are read-only GET.

#### Health Check

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/health"
# {"status":"ok"}
```

#### List Databases

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/list"
# {"dbs":["my_docs"]}
```

#### Semantic Search

Uses OpenAI embeddings for vector similarity search:

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/semantic?db=my_docs&q=what+is+RAG&limit=5"
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `db` | Yes | — | Target database name |
| `q` | Yes | — | Query string |
| `limit` | No | `10` | Max results (up to 100) |

Response format:

```json
{
  "results": [
    {
      "source": "/path/to/file.md",
      "matches": [
        {"chunk": 0, "content": "RAG stands for Retrieval-Augmented Generation..."}
      ]
    }
  ]
}
```

#### Keyword Search

Uses Chinese tokenizer (gse) for keyword matching:

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/keyword?db=my_docs&q=database&limit=10"
```

Parameters are the same as semantic search.

### Indexing Files

Drop files into the watched directory for automatic indexing:

```bash
cp document.md ~/Kura_my_docs/
# Within 10s: watcher detects change → parses and chunks → writes to SQLite
# Within 5s: embedder picks up pending chunks → calls OpenAI embedding → updates vector cache
```

Supported file formats:

| Format | Extension | Parser |
|--------|-----------|--------|
| Markdown / Plain text | `.md`, `.txt`, `.go`, `.py`, etc. | Markdown chunker |
| PDF | `.pdf` | PDF parser |
| Word | `.docx` | DOCX parser |
| PowerPoint | `.pptx` | PPTX parser |
| CSV / TSV | `.csv`, `.tsv` | Tabular parser |
| Excel | `.xlsx` | XLSX tabular parser |

## CLI Reference

### Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| `kura` | `kura` | Start server, loading all registered databases |
| `add` | `kura add <name>` | Register a new database, create directory and symlink |
| `list` | `kura list` | List registered databases |
| `remove` | `kura remove <name>` | Unregister and delete a database (interactive confirmation) |
| `edit` | `kura edit <old> <new>` | Rename a database |
| `help` | `kura help` | Show usage message |

### Server Behavior

| Behavior | Interval | Description |
|----------|----------|-------------|
| File watch polling | 10s | Scans inbox directory, compares file size and mtime for change detection |
| Embedding schedule | 5s | Fetches `is_embed=FALSE` chunks from SQLite, batch-calls OpenAI |
| Embedding batch size | 64 | Up to 64 chunks per batch |
| HTTP port | 10000–65535 random | Up to 10 bind attempts; writes address to endpoint file on success |

### Search Pipeline

**Semantic search** uses a two-stage strategy:

1. **Source filtering**: ranks sources by cosine similarity of their source-level vectors (average of all chunk vectors for that source), selecting the top N most relevant sources
2. **Chunk matching**: performs precise cosine calculation only on chunks within the filtered sources, returning top-K

This design dramatically reduces computation on large datasets while maintaining search quality.

**Keyword search** uses the gse Chinese tokenizer to split queries into keywords, then matches via SQLite `LIKE`, filtering out `dismiss=TRUE` deleted files.

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)