package openai

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"unicode/utf8"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const maxInputRunes = 8000 // CJK: 1 char ≈ 1 token; API limit 8192 tokens

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

type Vector []float32

func (o *OpenAI) EmbedBatch(ctx context.Context, texts []string) ([]Vector, error) {
	if o == nil || o.client == nil {
		return nil, fmt.Errorf("openai: not initialized")
	}
	if len(texts) == 0 {
		return nil, nil
	}

	safe := make([]string, len(texts))
	for i, t := range texts {
		if utf8.RuneCountInString(t) > maxInputRunes {
			slog.Warn("openai: truncating oversized input",
				slog.Int("index", i),
				slog.Int("runes", utf8.RuneCountInString(t)),
				slog.Int("limit", maxInputRunes))
			runes := []rune(t)
			safe[i] = string(runes[:maxInputRunes])
		} else {
			safe[i] = t
		}
	}

	body := map[string]any{
		"input":      safe,
		"model":      model,
		"dimensions": dim,
	}
	headers := map[string]string{"Authorization": "Bearer " + o.apiKey}

	data, _, err := go_pkg_http.POST[embedResponse](ctx, o.client, baseURL+"/embeddings", headers, body, "")
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}
	if len(data.Data) != len(texts) {
		return nil, fmt.Errorf("openai: returned %d vectors, want %d", len(data.Data), len(texts))
	}

	out := make([]Vector, len(data.Data))
	for i, d := range data.Data {
		if len(d.Embedding) != dim {
			return nil, fmt.Errorf("openai: returned dim %d, want %d (index %d)", len(d.Embedding), dim, i)
		}
		v := make(Vector, len(d.Embedding))
		copy(v, d.Embedding)
		out[i] = v
	}
	return out, nil
}

func Encode(v Vector) []byte {
	b := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

func Decode(b []byte) (Vector, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("vector: blob length %d not multiple of 4", len(b))
	}
	n := len(b) / 4
	v := make(Vector, n)
	for i := range n {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v, nil
}
