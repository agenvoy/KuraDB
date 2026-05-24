package agenvoy

// const (
// 	timeoutSeconds = 15
// 	defaultLimit   = 10
// 	maxLimit       = 100

// 	toolKeyword  = "rag_search_keyword"
// 	toolSemantic = "rag_search_semantic"
// 	toolListDB   = "rag_list_db"
// )

// type endpoint struct {
// 	URL         string `json:"url"`
// 	Method      string `json:"method"`
// 	ContentType string `json:"content_type"`
// 	Timeout     int    `json:"timeout"`
// }

// type parameter struct {
// 	Type        string `json:"type"`
// 	Description string `json:"description"`
// 	Required    bool   `json:"required"`
// 	Default     any    `json:"default,omitempty"`
// }

// type response struct {
// 	Format string `json:"format"`
// }

// type tool struct {
// 	Name        string               `json:"name"`
// 	Description string               `json:"description"`
// 	Endpoint    endpoint             `json:"endpoint"`
// 	Parameters  map[string]parameter `json:"parameters"`
// 	Response    response             `json:"response"`
// }

// func Register(baseURL string, dbNames []string) error {
// 	if baseURL == "" {
// 		return errors.New("baseURL is required")
// 	}

// 	dir, err := toolsDir()
// 	if err != nil {
// 		return err
// 	}
// 	if err := os.MkdirAll(dir, 0o755); err != nil {
// 		return fmt.Errorf("os.MkdirAll %s: %w", dir, err)
// 	}

// 	if err := cleanupManifests(dir); err != nil {
// 		return fmt.Errorf("cleanupManifests: %w", err)
// 	}

// 	tools := []tool{
// 		keywordTool(baseURL, dbNames),
// 		semanticTool(baseURL, dbNames),
// 		listTool(baseURL),
// 	}
// 	for _, t := range tools {
// 		path := filepath.Join(dir, t.Name+".json")
// 		if err := go_pkg_filesystem.WriteJSON(path, t, true); err != nil {
// 			return fmt.Errorf("go_pkg_filesystem.WriteJSON %s: %w", path, err)
// 		}
// 	}
// 	return nil
// }

// func Unregister() error {
// 	dir, err := toolsDir()
// 	if err != nil {
// 		return err
// 	}
// 	return cleanupManifests(dir)
// }

// func cleanupManifests(dir string) error {
// 	var firstErr error
// 	for _, name := range []string{toolKeyword, toolSemantic, toolListDB} {
// 		path := filepath.Join(dir, name+".json")
// 		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
// 			if firstErr == nil {
// 				firstErr = fmt.Errorf("os.Remove %s: %w", path, err)
// 			}
// 		}
// 	}
// 	return firstErr
// }

// func toolsDir() (string, error) {
// 	home, err := os.UserHomeDir()
// 	if err != nil {
// 		return "", fmt.Errorf("os.UserHomeDir: %w", err)
// 	}
// 	if home == "" {
// 		return "", errors.New("home directory is empty")
// 	}
// 	return filepath.Join(home, ".config", "agenvoy", "tools", "api"), nil
// }

// func limitParam() parameter {
// 	return parameter{
// 		Type:        "integer",
// 		Description: fmt.Sprintf("Max chunks to return (1-%d). Invalid values fall back to %d.", maxLimit, defaultLimit),
// 		Required:    false,
// 		Default:     defaultLimit,
// 	}
// }

// func dbParam(dbNames []string) parameter {
// 	desc := "Target RAG database name."
// 	if len(dbNames) > 0 {
// 		desc += " Currently loaded: " + strings.Join(dbNames, ", ") + "."
// 	} else {
// 		desc += " No databases are currently loaded."
// 	}
// 	desc += " Call rag_list_db to discover available databases at runtime."
// 	return parameter{
// 		Type:        "string",
// 		Description: desc,
// 		Required:    true,
// 	}
// }

// func keywordTool(baseURL string, dbNames []string) tool {
// 	return tool{
// 		Name:        toolKeyword,
// 		Description: "Search user's RAG knowledge base by exact token match. Use when query targets a precise string (filename, English term, person name, specific symbol). For natural-language / synonym queries, use semantic search instead.",
// 		Endpoint: endpoint{
// 			URL:         baseURL + "/api/keyword",
// 			Method:      "GET",
// 			ContentType: "json",
// 			Timeout:     timeoutSeconds,
// 		},
// 		Parameters: map[string]parameter{
// 			"db": dbParam(dbNames),
// 			"q": {
// 				Type:        "string",
// 				Description: "Search query. Natural-language input is tokenized into keywords; stopwords are removed.",
// 				Required:    true,
// 			},
// 			"limit": limitParam(),
// 		},
// 		Response: response{Format: "json"},
// 	}
// }

// func semanticTool(baseURL string, dbNames []string) tool {
// 	return tool{
// 		Name:        toolSemantic,
// 		Description: "Search user's RAG knowledge base by meaning. Use when query asks about (1) any document/PDF/note content, (2) topics/concepts the user may have ingested, (3) named entities that could be in user's files (e.g. 'X 寫了啥', 'X 詳細資料', '介紹 X'). Prefer this over keyword search for natural-language and synonym queries.",
// 		Endpoint: endpoint{
// 			URL:         baseURL + "/api/semantic",
// 			Method:      "GET",
// 			ContentType: "json",
// 			Timeout:     timeoutSeconds,
// 		},
// 		Parameters: map[string]parameter{
// 			"db": dbParam(dbNames),
// 			"q": {
// 				Type:        "string",
// 				Description: "Natural-language query; semantic similarity is computed against indexed chunk embeddings.",
// 				Required:    true,
// 			},
// 			"limit": limitParam(),
// 		},
// 		Response: response{Format: "json"},
// 	}
// }

// func listTool(baseURL string) tool {
// 	return tool{
// 		Name:        toolListDB,
// 		Description: "List user's RAG knowledge base databases (each db = a group of ingested files). Call this first before any RAG search to discover available db names and decide which to query.",
// 		Endpoint: endpoint{
// 			URL:         baseURL + "/api/list",
// 			Method:      "GET",
// 			ContentType: "json",
// 			Timeout:     timeoutSeconds,
// 		},
// 		Parameters: map[string]parameter{},
// 		Response:   response{Format: "json"},
// 	}
// }
