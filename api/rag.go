package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RAGClient struct {
	model  string
	client *http.Client
}

func NewRAGClient(model string) *RAGClient {
	if model == "" {
		model = "llama3.2:1b"
	}
	return &RAGClient{
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

type ollamaReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResp struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (r *RAGClient) Ask(question string, contextTexts []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("You are a helpful assistant. Use the following context to answer the question.\n\n")
	sb.WriteString("Context:\n")
	for i, t := range contextTexts {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
	}
	sb.WriteString(fmt.Sprintf("\nQuestion: %s\n\nAnswer:", question))

	body := ollamaReq{
		Model:  r.model,
		Prompt: sb.String(),
		Stream: false,
	}

	data, _ := json.Marshal(body)
	resp, err := r.client.Post(
		"http://127.0.0.1:11434/api/generate",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var or ollamaResp
	if err := json.Unmarshal(raw, &or); err != nil {
		return "", fmt.Errorf("parse ollama response: %w", err)
	}

	return strings.TrimSpace(or.Response), nil
}
