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
| `global.db` | 全域 SQLite（query cache） |
| `{name}/data.db` | 每個資料庫的 SQLite（file_data） |
| `{name}/inbox/` | 監控目錄，放入檔案即自動索引 |
| `{name}/record.json` | 檔案系統快照，用於變更偵測 |
| `endpoint` | 寫入 HTTP 伺服器位址 |

每個資料庫會在 `~/` 下建立符號連結 `Kura_{name}` → `~/.config/kuradb/{name}/inbox/`，方便拖放檔案。

## 使用方式

### 啟動伺服器

```bash
kura
```

伺服器啟動後會：
1. 載入所有已註冊的資料庫
2. 從 SQLite 重建向量快取
3. 啟動檔案監控器（每 10 秒輪詢）
4. 啟動 embedding 排程器（每 5 秒處理一批，每批最多 64 個區塊）
5. 在隨機埠啟動 HTTP API，並將位址寫入 `~/.config/kuradb/endpoint`

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

#### 語意搜尋

使用 OpenAI embedding 進行向量相似度搜尋：

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/semantic?db=my_docs&q=什麼是RAG&limit=5"
```

| 參數 | 必要 | 預設 | 說明 |
|-----------|----------|---------|-------------|
| `db` | 是 | — | 目標資料庫名稱 |
| `q` | 是 | — | 查詢字串 |
| `limit` | 否 | `10` | 回傳筆數上限（最大 100） |

回傳格式：

```json
{
  "results": [
    {
      "source": "/path/to/file.md",
      "matches": [
        {"chunk": 0, "content": "RAG 是檢索增強生成..."}
      ]
    }
  ]
}
```

#### 關鍵字搜尋

使用中文斷詞（gse）進行關鍵字比對：

```bash
curl "$(cat ~/.config/kuradb/endpoint)/api/keyword?db=my_docs&q=資料庫&limit=10"
```

參數與語意搜尋相同。

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
| `kura` | `kura` | 啟動伺服器，載入所有已註冊資料庫 |
| `add` | `kura add <name>` | 註冊新資料庫，建立目錄與符號連結 |
| `list` | `kura list` | 列出已註冊資料庫 |
| `remove` | `kura remove <name>` | 取消註冊並刪除資料庫（需互動確認） |
| `edit` | `kura edit <old> <new>` | 重新命名資料庫 |
| `help` | `kura help` | 顯示說明訊息 |

### 伺服器行為

| 行為 | 間隔 | 說明 |
|---------|----------|-------------|
| 檔案監控輪詢 | 10 秒 | 掃描 inbox 目錄，比對檔案大小與 mtime 偵測變更 |
| Embedding 排程 | 5 秒 | 從 SQLite 取出 `is_embed=FALSE` 的區塊，批次呼叫 OpenAI |
| Embedding 批次大小 | 64 | 每次最多處理 64 個區塊 |
| HTTP 埠 | 10000–65535 隨機 | 最多嘗試 10 次綁定，成功後寫入 endpoint 檔案 |

### 搜尋流程

**語意搜尋**採用兩階段策略：

1. **來源過濾**：先以來源層級向量（source vector，該來源所有區塊向量的平均值）進行 cosine 相似度排序，選出前 N 個最相關來源
2. **區塊比對**：僅對篩選後的來源內區塊進行精確 cosine 計算，取 top-K

此設計大幅減少大規模資料集下的計算量，同時維持搜尋品質。

**關鍵字搜尋**使用 gse 中文斷詞器將查詢拆分為關鍵詞，再以 SQLite `LIKE` 進行比對，過濾 `dismiss=TRUE` 的已刪除檔案。

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)