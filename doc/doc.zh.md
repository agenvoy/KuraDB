# KuraDB - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25 或更高版本
- OpenAI API Key（用於 embedding 向量生成）
- macOS 或 Linux（依賴 POSIX 檔案系統語義）

## 安裝

### 使用 go install

```bash
go install github.com/agenvoy/kuradb/cmd/app@latest
```

### 從原始碼建置

```bash
git clone https://github.com/pardnchiu/KuraDB.git
cd KuraDB
make build
# 二進位檔輸出至 bin/kura
```

### 使用 Makefile

```bash
# 建置並啟動
make app

# 新增資料庫
make add name=my_docs

# 列出已註冊資料庫
make list

# 移除資料庫
make remove name=my_docs

# 重新命名資料庫
make edit old=my_docs new=my_archive

# 固定／解除固定伺服器連接埠
make port set 8080
make port clear

# 停止執行中的伺服器
make stop
```

## 設定

### 環境變數

| 變數 | 必要 | 說明 |
|----------|----------|-------------|
| `OPENAI_API_KEY` | 是 | OpenAI API 金鑰，用於 text-embedding-3-small 模型 |

KuraDB 透過系統鑰匙圈讀取 `OPENAI_API_KEY`，啟動前請確保已設定。

### 設定目錄

所有資料存放於 `~/.config/kuradb/`：

| 路徑 | 用途 |
|------|------|
| `db.json` | 資料庫註冊表（JSON） |
| `config.json` | 伺服器設定（JSON），目前僅有固定 `port` |
| `global.db` | 全域 SQLite（query cache） |
| `{name}/data.db` | 每個資料庫的 SQLite（file_data） |
| `{name}/inbox/` | 監控目錄，放入檔案即自動索引 |
| `{name}/record.json` | 檔案系統快照，用於變更偵測 |
| `endpoint` | 寫入 HTTP 伺服器位址 |
| `daemon.log` | 背景伺服器的 stdout/stderr |

每個資料庫會在 `~/` 下建立符號連結 `Kura_{name}` → `~/.config/kuradb/{name}/inbox/`，方便拖放檔案。

## 使用方式

### 啟動伺服器

```bash
kura
```

`kura` 會 fork 進背景執行，伺服器就緒後立即返回（10 秒逾時）。伺服器啟動後會：
1. 載入所有已註冊的資料庫
2. 從 SQLite 重建向量快取
3. 啟動檔案監控器（每 10 秒輪詢）
4. 啟動 embedding 排程器（每 5 秒處理一批，每批最多 64 個區塊）
5. 啟動 HTTP API——若 `config.json` 有固定 `port` 則使用該埠，否則使用隨機埠——並將位址寫入 `~/.config/kuradb/endpoint`

### 管理資料庫

```bash
# 新增資料庫
kura add my_docs

# 列出所有資料庫
kura list

# 重新命名
kura edit my_docs my_archive

# 移除（需輸入 'yes' 確認）
kura remove my_archive

# 停止執行中的伺服器
kura stop

# 固定伺服器連接埠（會重啟伺服器）
kura port set 8080

# 解除固定連接埠（下次手動重啟後生效）
kura port clear
```

### 查詢 API

所有端點均為唯讀 GET。

#### 健康檢查

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/health"
# {"status":"ok"}
```

#### 列出資料庫

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/list"
# {"dbs":["my_docs"]}
```

#### 搜尋

同時執行關鍵字與語意搜尋，回傳兩者結果：

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/search?db=my_docs&q=什麼是RAG&limit=5"
```

| 參數 | 必要 | 預設 | 說明 |
|-----------|----------|---------|-------------|
| `db` | 是 | — | 目標資料庫名稱 |
| `q` | 是 | — | 查詢字串 |
| `limit` | 否 | `10` | 回傳筆數上限（最大 100） |
| `target` | 否 | 兩者皆執行 | 設為 `keyword` 或 `semantic` 只執行單一策略 |

- **關鍵字**：使用 gse 中文斷詞器拆分查詢，再以 SQLite `LIKE` 比對
- **語意**：使用 OpenAI embedding 進行向量相似度搜尋

回傳格式：

```json
{
  "keyword": [
    {
      "source": "/path/to/file.md",
      "matches": [
        {"chunk": 0, "content": "RAG 是檢索增強生成..."}
      ]
    }
  ],
  "semantic": [
    {
      "source": "/path/to/file.md",
      "matches": [
        {"chunk": 0, "content": "RAG 是檢索增強生成..."}
      ]
    }
  ]
}
```

未執行的 `target` 會整個從回應中省略（例如 `target=keyword` 只回傳 `keyword` 欄位）。

> `/api/semantic` 與 `/api/keyword` 仍可作為 `/api/search?target=semantic` 與 `/api/search?target=keyword` 的別名使用，但已標記為 deprecated，將於 v1.*.* 移除。

### 索引檔案

將檔案放入監控目錄即可自動索引：

```bash
cp document.md ~/Kura_my_docs/
# 10 秒內 watcher 偵測變更 → 解析分段 → 寫入 SQLite
# 5 秒內 embedder 取出 pending chunks → 呼叫 OpenAI embedding → 更新向量快取
```

支援的檔案格式：

| 格式 | 副檔名 | 解析方式 |
|--------|----------|-----------|
| Markdown / 純文字 | `.md`, `.txt`, `.go`, `.py` 等 | Markdown 解析器分段 |
| PDF | `.pdf` | PDF 解析器 |
| Word | `.docx` | DOCX 解析器 |
| PowerPoint | `.pptx` | PPTX 解析器 |
| CSV / TSV | `.csv`, `.tsv` | 表格解析器 |
| Excel | `.xlsx` | XLSX 表格解析器 |

## 命令列參考

### 指令

| 指令 | 語法 | 說明 |
|---------|--------|-------------|
| `kura` | `kura` | fork 伺服器進背景執行，載入所有已註冊資料庫 |
| `add` | `kura add <name>` | 註冊新資料庫，建立目錄與符號連結 |
| `list` | `kura list` | 列出已註冊資料庫 |
| `remove` | `kura remove <name>` | 取消註冊並刪除資料庫（需互動確認） |
| `edit` | `kura edit <old> <new>` | 重新命名資料庫 |
| `stop` | `kura stop` | 停止執行中的背景伺服器 |
| `port` | `kura port set <port>` \| `kura port clear` | 固定／解除固定 `config.json` 中的 HTTP 埠（`set` 會重啟伺服器；`clear` 於下次手動重啟後生效） |
| `help` | `kura help` | 顯示說明訊息 |

### 伺服器行為

| 行為 | 間隔 | 說明 |
|---------|----------|-------------|
| 檔案監控輪詢 | 10 秒 | 掃描 inbox 目錄，比對檔案大小與 mtime 偵測變更 |
| Embedding 排程 | 5 秒 | 從 SQLite 取出 `is_embed=FALSE` 的區塊，批次呼叫 OpenAI |
| Embedding 批次大小 | 64 | 每次最多處理 64 個區塊 |
| HTTP 埠 | `config.json` 固定的 `port`，否則 10000–65535 隨機 | 隨機模式：最多嘗試 10 次綁定，成功後寫入 endpoint 檔案 |

### 搜尋流程

**語意搜尋**採用兩階段策略：

1. **來源過濾**：先以來源層級向量（source vector，該來源所有區塊向量的平均值）進行 cosine 相似度排序，選出前 N 個最相關來源
2. **區塊比對**：僅對篩選後的來源內區塊進行精確 cosine 計算，取 top-K

此設計大幅減少大規模資料集下的計算量，同時維持搜尋品質。

**關鍵字搜尋**使用 gse 中文斷詞器將查詢拆分為關鍵詞，再以 SQLite `LIKE` 進行比對，過濾 `dismiss=TRUE` 的已刪除檔案。

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)