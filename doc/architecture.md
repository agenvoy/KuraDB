# KuraDB - Architecture

> Back to [README](../README.md)

## Overview

```mermaid
graph TB
    subgraph Entry
        CLI[CLI Commands]
        HTTP[HTTP API :random port]
    end

    subgraph API Layer
        Router[Gin Router]
        Health[Health Check]
        List[Database List]
        Semantic[Semantic Search]
        Keyword[Keyword Search]
    end

    subgraph Index Pipeline
        Watcher[File Watcher<br/>10s polling]
        Parser[File Parser<br/>PDF/DOCX/PPTX/CSV/XLSX/MD]
        SQLite[(Per-DB SQLite<br/>file_data)]
        Embedder[Embedding Scheduler<br/>5s batch]
        OpenAI[OpenAI API<br/>text-embedding-3-small]
    end

    subgraph Query Layer
        QCache[Query Cache<br/>memory + SQLite]
        Segmenter[Chinese Tokenizer<br/>gse]
        Vector[Vector Cache<br/>in-memory cosine search]
    end

    CLI -->|add/list/remove/edit| Registry[(Registry<br/>db.json)]
    HTTP --> Router
    Router --> Health
    Router --> List
    Router --> Semantic
    Router --> Keyword

    Watcher --> Parser
    Parser --> SQLite
    SQLite --> Embedder
    Embedder --> OpenAI
    OpenAI --> Embedder
    Embedder --> Vector
    Embedder --> SQLite

    Semantic --> QCache
    QCache --> OpenAI
    Semantic --> Vector
    Vector --> SQLite

    Keyword --> Segmenter
    Segmenter --> SQLite

    Registry -->|load on startup| Watcher
```

## Module: cmd/app (Entry Point)

`main.go` manages the full service lifecycle, initializing all subsystems in order.

```mermaid
graph TB
    subgraph main
        Start[Start] --> Args{CLI args?}
        Args -->|add/list/remove/edit| CLI[Execute CLI command]
        Args -->|none| Server[runServer]
        Server --> Config[Load config directory]
        Config --> Registry[Load registry]
        Registry --> OpenAI[Init OpenAI client]
        OpenAI --> GlobalDB[Open Global SQLite]
        GlobalDB --> QCache[Load query cache]
        QCache --> Segmenter[Init tokenizer]
        Segmenter --> Vector[Init vector cache]
        Vector --> PerDB[Load each database]
        PerDB --> Watcher[Start file watcher]
        PerDB --> Embedder[Start embedding scheduler]
        PerDB --> HTTP[Start HTTP server]
        HTTP --> Wait[Wait for SIGINT/SIGTERM]
        Wait --> Shutdown[Close all resources]
    end
```

## Module: internal/api (HTTP API Layer)

A read-only REST API built on Gin, exposing only query endpoints.

```mermaid
graph TB
    subgraph API
        Router["Router()"] --> Health["GET /api/health"]
        Router --> List["GET /api/list"]
        Router --> Semantic["GET /api/semantic"]
        Router --> Keyword["GET /api/keyword"]
        Semantic --> QDB["queryDB middleware<br/>validates db param"]
        Keyword --> QDB
    end

    subgraph Handler
        SemanticHandler["Semantic()"] --> GetSemantic["getSemantic()"]
        GetSemantic --> QCache["Query cache lookup"]
        QCache -->|miss| EmbedAPI["OpenAI EmbedBatch"]
        QCache -->|hit| VectorSearch["vector.Search()"]
        EmbedAPI --> VectorSearch
        VectorSearch --> GetByIDs["GetByIDs()"]
        GetByIDs --> Group["group() aggregation"]

        KeywordHandler["Keyword()"] --> Tokenize["segmenter.Tokenize()"]
        Tokenize --> SearchKW["SearchKeyword()"]
        SearchKW --> Group
    end
```

## Module: internal/database (Data Layer)

SQLite is the single source of truth, managed via `go-sqlkit` with read/write connection pooling.

```mermaid
graph TB
    subgraph Database
        DB["DB struct<br/>Read *sql.DB<br/>Write *sql.DB"]
        OpenPerDB["OpenPerDB()<br/>+ file_data schema"]
        OpenGlobal["OpenGlobal()<br/>+ query_cache schema"]
    end

    subgraph Registry
        Reg["Registry struct<br/>path + sync.Mutex"]
        Load["Load()"]
        Add["Add()"]
        Remove["Remove()"]
        Rename["Rename()"]
        Has["Has()"]
    end

    subgraph Handler
        Upsert["Upsert()<br/>INSERT OR REPLACE"]
        Dismiss["Dismiss()<br/>soft delete"]
        GetByIDs["GetByIDs()<br/>lookup by ID"]
        SearchKeyword["SearchKeyword()<br/>LIKE matching"]
        ListPending["ListPending()<br/>unembedded chunks"]
        ListEmbedded["ListEmbedded()<br/>embedded chunks"]
        UpdateEmbedding["UpdateEmbedding()<br/>write vectors"]
        SaveQueryCache["SaveQueryCache()"]
        LoadQueryCache["LoadQueryCache()"]
    end

    DB --> Handler
    Reg -->|JSON file| FS[(~/.config/kuradb/db.json)]
```

### file_data Schema

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment primary key |
| `source` | TEXT NOT NULL | Source file path |
| `chunk` | INTEGER NOT NULL | Chunk index |
| `total` | INTEGER NOT NULL | Total chunks in file |
| `content` | TEXT NOT NULL | Chunk text content |
| `embedding` | BLOB | OpenAI vector (512-dim float32) |
| `is_embed` | BOOLEAN | Whether embedding is complete |
| `dismiss` | BOOLEAN | Soft delete flag |
| `created_at` | TIMESTAMP | Creation time |
| `updated_at` | TIMESTAMP | Last update time |

Unique constraint: `(source, chunk)`.

## Module: internal/filesystem (File Watching & Parsing)

```mermaid
graph TB
    subgraph Watcher
        WalkFiles["WalkFiles()<br/>recursive directory scan"]
        Snapshot["File snapshot diff<br/>size + mtime"]
        Record["record.json<br/>persistent snapshot"]
    end

    subgraph Parser
        Dispatch{Extension check}
        Dispatch -->|.md/.txt/.go...| MD["Markdown parser"]
        Dispatch -->|.pdf| PDF["PDF parser"]
        Dispatch -->|.docx| DOCX["DOCX parser"]
        Dispatch -->|.pptx| PPTX["PPTX parser"]
        Dispatch -->|.csv/.tsv| CSV["CSV tabular parser"]
        Dispatch -->|.xlsx| XLSX["XLSX tabular parser"]
        Dispatch -->|other| Sniff["sniff.go<br/>MIME detection"]
    end

    subgraph Cleanup
        DismissRemoved["dismissRemoved()<br/>recursively mark deleted files"]
    end

    WalkFiles --> Snapshot
    Snapshot -->|changed| Dispatch
    Dispatch --> Upsert["databaseHandler.Upsert()"]
    Snapshot -->|unchanged| Skip[Skip]
    WalkFiles -->|file removed| DismissRemoved
    DismissRemoved --> Dismiss["databaseHandler.Dismiss()"]
```

## Module: internal/openai (Embedding Client)

```mermaid
graph TB
    subgraph OpenAI
        New["New()<br/>reads API key from keychain"]
        EmbedBatch["EmbedBatch()<br/>batch call /v1/embeddings"]
        Encode["Encode()<br/>[]float32 → []byte"]
        Decode["Decode()<br/>[]byte → []float32"]
        Dim["Dim() → 512"]
    end

    subgraph Cache
        QCache["Cache struct<br/>sync.RWMutex + map"]
        Get["Get()"]
        Set["Set()"]
        Preload["Preload()"]
        OnSet["OnSet callback<br/>async write to SQLite"]
    end

    New --> Keychain[(System Keychain<br/>OPENAI_API_KEY)]
    EmbedBatch --> API["POST api.openai.com/v1/embeddings<br/>model: text-embedding-3-small<br/>dimensions: 512"]
```

## Module: internal/vector (Vector Cache & Search)

```mermaid
graph TB
    subgraph Cache
        VCache["Cache struct<br/>dbBuckets map"]
        InitBucket["InitBucket()"]
    end

    subgraph Bucket
        B["Bucket struct"]
        IDV["idVectors<br/>map[int64][]float32"]
        IDSrc["idSource<br/>map[int64]string"]
        SrcChunks["sourceChunks<br/>map[string][]int64"]
        SrcVecs["sourceVectors<br/>map[string][]float32"]
    end

    subgraph Operations
        Set["Set()<br/>write single vector"]
        Rebuild["Rebuild()<br/>rebuild source-level vector"]
        RebuildAll["RebuildAll()<br/>rebuild all sources"]
        Search["Search()<br/>two-stage cosine search"]
    end

    subgraph SearchDetail["Search Pipeline"]
        Stage1["Stage 1: Source Filtering<br/>cosine(query, source_vector)<br/>select top N sources"]
        Stage2["Stage 2: Chunk Matching<br/>parallel cosine(query, chunk_vector)<br/>return top-K"]
        Cosine["cosine()<br/>dot(a,b) / (|a|*|b|)"]
    end

    VCache --> Bucket
    Bucket --> Operations
    Search --> Stage1
    Stage1 --> Stage2
    Stage2 --> Cosine
```

## Module: internal/utils/segmenter (Chinese Tokenizer)

```mermaid
graph LR
    Tokenize["Tokenize()"] --> GSE["gse tokenizer"]
    GSE --> Keywords["[]string keywords"]
    Keywords --> SearchKW["SearchKeyword()<br/>SQLite LIKE"]
```

## Data Flow: File Indexing

```mermaid
sequenceDiagram
    participant User
    participant FS as Filesystem
    participant Watcher as Watcher (10s)
    participant Parser as Parser
    participant SQLite as SQLite
    participant Embedder as Embedder (5s)
    participant OpenAI as OpenAI API
    participant Vector as Vector Cache

    User->>FS: Drop file into ~/Kura_{name}/
    Watcher->>FS: Scan directory, diff snapshot
    Watcher->>Watcher: Detect change (size/mtime)
    Watcher->>Parser: Select parser by extension
    Parser->>Parser: Parse and chunk
    Parser->>SQLite: Upsert chunks (is_embed=FALSE)
    Embedder->>SQLite: ListPending (is_embed=FALSE)
    SQLite-->>Embedder: Return unembedded chunks
    Embedder->>OpenAI: EmbedBatch (up to 64)
    OpenAI-->>Embedder: Return 512-dim vectors
    Embedder->>SQLite: UpdateEmbedding (is_embed=TRUE)
    Embedder->>Vector: Set + Rebuild
```

## Data Flow: Semantic Search

```mermaid
sequenceDiagram
    participant Client
    participant API as HTTP API
    participant QCache as Query Cache
    participant OpenAI as OpenAI API
    participant Vector as Vector Cache
    participant SQLite as SQLite

    Client->>API: GET /api/semantic?db=X&q=...
    API->>QCache: Get(query)
    alt cache hit
        QCache-->>API: Return vector
    else cache miss
        API->>OpenAI: EmbedBatch([query])
        OpenAI-->>API: Return vector
        API->>QCache: Set(query, vector)
    end
    API->>Vector: Search(db, vector, limit)
    Vector->>Vector: Stage 1: Source filtering
    Vector->>Vector: Stage 2: Chunk matching
    Vector-->>API: top-K hits (id + score)
    API->>SQLite: GetByIDs(ids)
    SQLite-->>API: Return chunk content
    API->>API: group() by source
    API-->>Client: JSON response
```

## State Machine: File Lifecycle

```mermaid
stateDiagram-v2
    [*] --> New: File dropped into inbox
    New --> Parsed: Watcher detects and parses
    Parsed --> Embedded: Embedder calls OpenAI
    Embedded --> Updated: File content changes
    Updated --> Parsed: Re-parse
    Embedded --> Dismissed: File removed from inbox
    Parsed --> Dismissed: File removed from inbox
    Dismissed --> [*]
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)