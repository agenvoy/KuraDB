# KuraDB - 架構

> 返回 [README](./README.zh.md)

## 概覽

```mermaid
graph TB
    subgraph 入口層
        CLI[CLI 命令]
        HTTP[HTTP API :隨機埠]
    end

    subgraph API 層
        Router[Gin Router]
        Health[健康檢查]
        List[資料庫列表]
        Semantic[語意搜尋]
        Keyword[關鍵字搜尋]
    end

    subgraph 索引管線
        Watcher[檔案監控器<br/>10s 輪詢]
        Parser[檔案解析器<br/>PDF/DOCX/PPTX/CSV/XLSX/MD]
        SQLite[(Per-DB SQLite<br/>file_data)]
        Embedder[Embedding 排程器<br/>5s 批次處理]
        OpenAI[OpenAI API<br/>text-embedding-3-small]
    end

    subgraph 查詢層
        QCache[查詢快取<br/>記憶體 + SQLite]
        Segmenter[中文斷詞器<br/>gse]
        Vector[向量快取<br/>記憶體 cosine 搜尋]
    end

    CLI -->|add/list/remove/edit| Registry[(註冊表<br/>db.json)]
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

    Registry -->|啟動載入| Watcher
```

## 模組：cmd/app（入口點）

`main.go` 為整個服務的生命週期管理者，負責依序初始化所有子系統。

```mermaid
graph TB
    subgraph main
        Start[啟動] --> Args{CLI 參數?}
        Args -->|add/list/remove/edit| CLI[執行 CLI 命令]
        Args -->|無參數| Server[runServer]
        Server --> Config[載入設定目錄]
        Config --> Registry[載入註冊表]
        Registry --> OpenAI[初始化 OpenAI 客戶端]
        OpenAI --> GlobalDB[開啟 Global SQLite]
        GlobalDB --> QCache[載入查詢快取]
        QCache --> Segmenter[初始化斷詞器]
        Segmenter --> Vector[初始化向量快取]
        Vector --> PerDB[逐一載入每個資料庫]
        PerDB --> Watcher[啟動檔案監控]
        PerDB --> Embedder[啟動 Embedding 排程]
        PerDB --> HTTP[啟動 HTTP 伺服器]
        HTTP --> Wait[等待 SIGINT/SIGTERM]
        Wait --> Shutdown[關閉所有資源]
    end
```

## 模組：internal/api（HTTP API 層）

基於 Gin 框架的唯讀 REST API，僅暴露查詢端點。

```mermaid
graph TB
    subgraph API
        Router["Router()"] --> Health["GET /api/health"]
        Router --> List["GET /api/list"]
        Router --> Semantic["GET /api/semantic"]
        Router --> Keyword["GET /api/keyword"]
        Semantic --> QDB["queryDB 中介層<br/>驗證 db 參數"]
        Keyword --> QDB
    end

    subgraph Handler
        SemanticHandler["Semantic()"] --> GetSemantic["getSemantic()"]
        GetSemantic --> QCache["查詢快取查詢"]
        QCache -->|未命中| EmbedAPI["OpenAI EmbedBatch"]
        QCache -->|命中| VectorSearch["vector.Search()"]
        EmbedAPI --> VectorSearch
        VectorSearch --> GetByIDs["GetByIDs()"]
        GetByIDs --> Group["group() 分組"]

        KeywordHandler["Keyword()"] --> Tokenize["segmenter.Tokenize()"]
        Tokenize --> SearchKW["SearchKeyword()"]
        SearchKW --> Group
    end
```

## 模組：internal/database（資料層）

SQLite 為唯一資料源，透過 `go-sqlkit` 管理讀寫分離連線池。

```mermaid
graph TB
    subgraph Database
        DB["DB 結構體<br/>Read *sql.DB<br/>Write *sql.DB"]
        OpenPerDB["OpenPerDB()<br/>+ file_data schema"]
        OpenGlobal["OpenGlobal()<br/>+ query_cache schema"]
    end

    subgraph Registry
        Reg["Registry 結構體<br/>path + sync.Mutex"]
        Load["Load()"]
        Add["Add()"]
        Remove["Remove()"]
        Rename["Rename()"]
        Has["Has()"]
    end

    subgraph Handler
        Upsert["Upsert()<br/>INSERT OR REPLACE"]
        Dismiss["Dismiss()<br/>標記刪除"]
        GetByIDs["GetByIDs()<br/>依 ID 查詢"]
        SearchKeyword["SearchKeyword()<br/>LIKE 比對"]
        ListPending["ListPending()<br/>未嵌入區塊"]
        ListEmbedded["ListEmbedded()<br/>已嵌入區塊"]
        UpdateEmbedding["UpdateEmbedding()<br/>寫入向量"]
        SaveQueryCache["SaveQueryCache()"]
        LoadQueryCache["LoadQueryCache()"]
    end

    DB --> Handler
    Reg -->|JSON 檔案| FS[(~/.config/kuradb/db.json)]
```

### file_data 結構

| 欄位 | 型別 | 說明 |
|--------|------|-------------|
| `id` | INTEGER PK | 自動遞增主鍵 |
| `source` | TEXT NOT NULL | 來源檔案路徑 |
| `chunk` | INTEGER NOT NULL | 區塊編號 |
| `total` | INTEGER NOT NULL | 該檔案總區塊數 |
| `content` | TEXT NOT NULL | 區塊文字內容 |
| `embedding` | BLOB | OpenAI 向量（512 維 float32） |
| `is_embed` | BOOLEAN | 是否已完成嵌入 |
| `dismiss` | BOOLEAN | 軟刪除標記 |
| `created_at` | TIMESTAMP | 建立時間 |
| `updated_at` | TIMESTAMP | 更新時間 |

唯一約束：`(source, chunk)`。

## 模組：internal/filesystem（檔案監控與解析）

```mermaid
graph TB
    subgraph Watcher
        WalkFiles["WalkFiles()<br/>遞迴目錄掃描"]
        Snapshot["檔案快照比對<br/>size + mtime"]
        Record["record.json<br/>持久化快照"]
    end

    subgraph Parser
        Dispatch{副檔名判斷}
        Dispatch -->|.md/.txt/.go...| MD["Markdown 解析器"]
        Dispatch -->|.pdf| PDF["PDF 解析器"]
        Dispatch -->|.docx| DOCX["DOCX 解析器"]
        Dispatch -->|.pptx| PPTX["PPTX 解析器"]
        Dispatch -->|.csv/.tsv| CSV["CSV 表格解析"]
        Dispatch -->|.xlsx| XLSX["XLSX 表格解析"]
        Dispatch -->|其他| Sniff["sniff.go<br/>MIME 偵測"]
    end

    subgraph Cleanup
        DismissRemoved["dismissRemoved()<br/>遞迴標記已刪除檔案"]
    end

    WalkFiles --> Snapshot
    Snapshot -->|變更| Dispatch
    Dispatch --> Upsert["databaseHandler.Upsert()"]
    Snapshot -->|無變更| Skip[略過]
    WalkFiles -->|檔案已移除| DismissRemoved
    DismissRemoved --> Dismiss["databaseHandler.Dismiss()"]
```

## 模組：internal/openai（Embedding 客戶端）

```mermaid
graph TB
    subgraph OpenAI
        New["New()<br/>從鑰匙圈讀取 API Key"]
        EmbedBatch["EmbedBatch()<br/>批次呼叫 /v1/embeddings"]
        Encode["Encode()<br/>[]float32 → []byte"]
        Decode["Decode()<br/>[]byte → []float32"]
        Dim["Dim() → 512"]
    end

    subgraph Cache
        QCache["Cache 結構體<br/>sync.RWMutex + map"]
        Get["Get()"]
        Set["Set()"]
        Preload["Preload()"]
        OnSet["OnSet 回呼<br/>非同步寫入 SQLite"]
    end

    New --> Keychain[(系統鑰匙圈<br/>OPENAI_API_KEY)]
    EmbedBatch --> API["POST api.openai.com/v1/embeddings<br/>model: text-embedding-3-small<br/>dimensions: 512"]
```

## 模組：internal/vector（向量快取與搜尋）

```mermaid
graph TB
    subgraph Cache
        VCache["Cache 結構體<br/>dbBuckets map"]
        InitBucket["InitBucket()"]
    end

    subgraph Bucket
        B["Bucket 結構體"]
        IDV["idVectors<br/>map[int64][]float32"]
        IDSrc["idSource<br/>map[int64]string"]
        SrcChunks["sourceChunks<br/>map[string][]int64"]
        SrcVecs["sourceVectors<br/>map[string][]float32"]
    end

    subgraph Operations
        Set["Set()<br/>寫入單筆向量"]
        Rebuild["Rebuild()<br/>重建來源層級向量"]
        RebuildAll["RebuildAll()<br/>重建所有來源"]
        Search["Search()<br/>兩階段 cosine 搜尋"]
    end

    subgraph SearchDetail["Search 詳細流程"]
        Stage1["階段一：來源過濾<br/>cosine(query, source_vector)<br/>取前 N 個來源"]
        Stage2["階段二：區塊比對<br/>平行 cosine(query, chunk_vector)<br/>取 top-K"]
        Cosine["cosine()<br/>dot(a,b) / (|a|*|b|)"]
    end

    VCache --> Bucket
    Bucket --> Operations
    Search --> Stage1
    Stage1 --> Stage2
    Stage2 --> Cosine
```

## 模組：internal/utils/segmenter（中文斷詞）

```mermaid
graph LR
    Tokenize["Tokenize()"] --> GSE["gse 斷詞器"]
    GSE --> Keywords["[]string 關鍵詞"]
    Keywords --> SearchKW["SearchKeyword()<br/>SQLite LIKE"]
```

## 資料流：檔案索引

```mermaid
sequenceDiagram
    participant User as 使用者
    participant FS as 檔案系統
    participant Watcher as Watcher (10s)
    participant Parser as Parser
    participant SQLite as SQLite
    participant Embedder as Embedder (5s)
    participant OpenAI as OpenAI API
    participant Vector as Vector Cache

    User->>FS: 放入檔案至 ~/Kura_{name}/
    Watcher->>FS: 掃描目錄，比對快照
    Watcher->>Watcher: 偵測變更（size/mtime）
    Watcher->>Parser: 依副檔名選擇解析器
    Parser->>Parser: 解析並分段
    Parser->>SQLite: Upsert chunks (is_embed=FALSE)
    Embedder->>SQLite: ListPending (is_embed=FALSE)
    SQLite-->>Embedder: 回傳未嵌入區塊
    Embedder->>OpenAI: EmbedBatch (最多 64 筆)
    OpenAI-->>Embedder: 回傳 512 維向量
    Embedder->>SQLite: UpdateEmbedding (is_embed=TRUE)
    Embedder->>Vector: Set + Rebuild
```

## 資料流：語意搜尋

```mermaid
sequenceDiagram
    participant Client as 客戶端
    participant API as HTTP API
    participant QCache as Query Cache
    participant OpenAI as OpenAI API
    participant Vector as Vector Cache
    participant SQLite as SQLite

    Client->>API: GET /api/semantic?db=X&q=...
    API->>QCache: Get(query)
    alt 快取命中
        QCache-->>API: 回傳向量
    else 快取未命中
        API->>OpenAI: EmbedBatch([query])
        OpenAI-->>API: 回傳向量
        API->>QCache: Set(query, vector)
    end
    API->>Vector: Search(db, vector, limit)
    Vector->>Vector: 階段一：來源過濾
    Vector->>Vector: 階段二：區塊比對
    Vector-->>API: top-K hits (id + score)
    API->>SQLite: GetByIDs(ids)
    SQLite-->>API: 回傳區塊內容
    API->>API: group() 依來源分組
    API-->>Client: JSON 回應
```

## 狀態機：檔案生命週期

```mermaid
stateDiagram-v2
    [*] --> New: 檔案放入 inbox
    New --> Parsed: Watcher 偵測並解析
    Parsed --> Embedded: Embedder 呼叫 OpenAI
    Embedded --> Updated: 檔案內容變更
    Updated --> Parsed: 重新解析
    Embedded --> Dismissed: 檔案從 inbox 移除
    Parsed --> Dismissed: 檔案從 inbox 移除
    Dismissed --> [*]
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)