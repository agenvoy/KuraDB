package openai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	go_pkg_keychain "github.com/pardnchiu/go-pkg/filesystem/keychain"
)

const (
	baseURL        = "https://api.openai.com/v1"
	model          = "text-embedding-3-small"
	dim            = 512
	requestTimeout = 1 * time.Minute
)

type OpenAI struct {
	apiKey string
	client *http.Client
}

func Dim() int { return dim }

type Embedder interface {
	EmbedBatch(ctx context.Context, texts []string) ([]Vector, error)
}

func New() (*OpenAI, error) {
	apiKey := go_pkg_keychain.Get("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	return &OpenAI{
		apiKey: apiKey,
		client: &http.Client{Timeout: requestTimeout},
	}, nil
}
