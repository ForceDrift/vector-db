package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type EmbedClient struct {
	cmd    *exec.Cmd
	port   int
	client *http.Client
}

type embedResponse struct {
	Embedding  []float64    `json:"embedding"`
	Dimension  int          `json:"dimension"`
	Embeddings [][]float64  `json:"embeddings"`
	Error      string       `json:"error,omitempty"`
}

type healthResponse struct {
	Status    string `json:"status"`
	Model     string `json:"model"`
	Dimension int    `json:"dimension"`
}

func resolveEmbedPaths() (script string, python string, workDir string) {
	cwd, _ := os.Getwd()

	candidates := []struct{ embedDir, venv string }{
		{filepath.Join(cwd, "../api/embed"), filepath.Join(cwd, "../api/.venv/bin/python3")},
		{filepath.Join(cwd, "embed"), filepath.Join(cwd, ".venv/bin/python3")},
		{filepath.Join(cwd, "api/embed"), filepath.Join(cwd, "api/.venv/bin/python3")},
	}
	for _, c := range candidates {
		s := filepath.Join(c.embedDir, "embed_server.py")
		if _, err := os.Stat(s); err == nil {
			return s, c.venv, c.embedDir
		}
	}
	// fallback: try relative to binary
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	s := filepath.Join(base, "../api/embed/embed_server.py")
	if _, err := os.Stat(s); err == nil {
		return s, filepath.Join(base, "../api/.venv/bin/python3"), filepath.Join(base, "../api/embed")
	}
	return "", "", ""
}

func NewEmbedClient() (*EmbedClient, error) {
	script, pythonPath, workDir := resolveEmbedPaths()
	if script == "" {
		return nil, fmt.Errorf("embed_server.py not found")
	}
	if _, err := os.Stat(pythonPath); err != nil {
		return nil, fmt.Errorf("python3 not found at %s: %w", pythonPath, err)
	}

	port := 8765

	cmd := exec.Command(pythonPath, script, fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start embed server: %w", err)
	}

	ec := &EmbedClient{
		cmd:    cmd,
		port:   port,
		client: &http.Client{Timeout: 120 * time.Second},
	}

	for i := 0; i < 60; i++ {
		time.Sleep(500 * time.Millisecond)
		if ec.Health() {
			return ec, nil
		}
	}

	cmd.Process.Kill()
	return nil, fmt.Errorf("embed server not ready after 30s")
}

func (ec *EmbedClient) Health() bool {
	resp, err := ec.client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", ec.port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var h healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return false
	}
	return h.Status == "ok"
}

func (ec *EmbedClient) EmbedText(text string) ([]float64, error) {
	body := map[string]string{"text": text}
	data, _ := json.Marshal(body)

	resp, err := ec.client.Post(
		fmt.Sprintf("http://127.0.0.1:%d/embed", ec.port),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var er embedResponse
	if err := json.Unmarshal(raw, &er); err != nil {
		return nil, fmt.Errorf("parse embed response: %w", err)
	}
	if er.Error != "" {
		return nil, fmt.Errorf("embed error: %s", er.Error)
	}
	return er.Embedding, nil
}

func (ec *EmbedClient) EmbedBatch(texts []string) ([][]float64, error) {
	body := map[string][]string{"texts": texts}
	data, _ := json.Marshal(body)

	resp, err := ec.client.Post(
		fmt.Sprintf("http://127.0.0.1:%d/batch-embed", ec.port),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("batch-embed request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var er embedResponse
	if err := json.Unmarshal(raw, &er); err != nil {
		return nil, fmt.Errorf("parse batch-embed response: %w", err)
	}
	if er.Error != "" {
		return nil, fmt.Errorf("batch-embed error: %s", er.Error)
	}
	return er.Embeddings, nil
}

func (ec *EmbedClient) Close() {
	if ec.cmd != nil && ec.cmd.Process != nil {
		ec.cmd.Process.Kill()
		ec.cmd.Wait()
	}
}
