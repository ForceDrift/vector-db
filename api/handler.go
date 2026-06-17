package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type insertReq struct {
	ID     uint64    `json:"id"`
	Values []float64 `json:"values"`
}

type insertTextReq struct {
	ID   uint64 `json:"id"`
	Text string `json:"text"`
}

type searchTextReq struct {
	Text     string       `json:"text"`
	K        int          `json:"k"`
	Distance DistanceType `json:"distance"`
}

type askReq struct {
	Question string `json:"question"`
	K        int    `json:"k"`
}

type askResp struct {
	Answer  string              `json:"answer"`
	Sources []askSource         `json:"sources"`
}

type askSource struct {
	ID   uint64 `json:"id"`
	Text string `json:"text"`
}

type removeReq struct {
	ID uint64 `json:"id"`
}

type getReq struct {
	ID uint64 `json:"id"`
}

type searchReq struct {
	Query    []float64    `json:"query"`
	K        int          `json:"k"`
	Distance DistanceType `json:"distance"`
}

type apiResp struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func respond(w http.ResponseWriter, status int, resp apiResp) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func handleInsert(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req insertReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		if err := s.Insert(req.ID, req.Values); err != nil {
			respond(w, http.StatusConflict, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true})
	}
}

func handleRemove(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req removeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		if err := s.Remove(req.ID); err != nil {
			respond(w, http.StatusNotFound, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true})
	}
}

func handleGetAll(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vectors := s.GetAll()
		respond(w, http.StatusOK, apiResp{Success: true, Data: vectors})
	}
}

func handleGet(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req getReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		vec, err := s.Get(req.ID)
		if err != nil {
			respond(w, http.StatusNotFound, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true, Data: Vector{ID: req.ID, Values: vec}})
	}
}

func handleSearch(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req searchReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		results, err := s.Search(req.Query, req.K, req.Distance)
		if err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true, Data: results})
	}
}

func handleInsertText(s *Store, ec *EmbedClient, ts *TextStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req insertTextReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		if req.Text == "" {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: "missing 'text' field"})
			return
		}
		vec, err := ec.EmbedText(req.Text)
		if err != nil {
			respond(w, http.StatusInternalServerError, apiResp{Success: false, Error: err.Error()})
			return
		}
		if err := s.Insert(req.ID, vec); err != nil {
			respond(w, http.StatusConflict, apiResp{Success: false, Error: err.Error()})
			return
		}
		ts.Set(req.ID, req.Text)
		respond(w, http.StatusOK, apiResp{Success: true, Data: map[string]interface{}{
			"id":        req.ID,
			"dimension": len(vec),
		}})
	}
}

func handleSearchText(s *Store, ec *EmbedClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req searchTextReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		if req.Text == "" {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: "missing 'text' field"})
			return
		}
		k := req.K
		if k <= 0 {
			k = 10
		}
		dist := req.Distance
		if dist == "" {
			dist = L2
		}
		vec, err := ec.EmbedText(req.Text)
		if err != nil {
			respond(w, http.StatusInternalServerError, apiResp{Success: false, Error: err.Error()})
			return
		}
		results, err := s.Search(vec, k, dist)
		if err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true, Data: results})
	}
}

func handleAsk(s *Store, ec *EmbedClient, rc *RAGClient, ts *TextStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req askReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: err.Error()})
			return
		}
		if req.Question == "" {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: "missing 'question' field"})
			return
		}
		k := req.K
		if k <= 0 {
			k = 5
		}

		vec, err := ec.EmbedText(req.Question)
		if err != nil {
			respond(w, http.StatusInternalServerError, apiResp{Success: false, Error: "embed: " + err.Error()})
			return
		}

		results, err := s.Search(vec, k, L2)
		if err != nil {
			respond(w, http.StatusBadRequest, apiResp{Success: false, Error: "search: " + err.Error()})
			return
		}

		var ids []uint64
		for _, res := range results {
			ids = append(ids, res.ID)
		}
		texts := ts.GetBatch(ids)

		var contextTexts []string
		var sources []askSource
		for _, res := range results {
			if t, ok := texts[res.ID]; ok {
				contextTexts = append(contextTexts, t)
				sources = append(sources, askSource{ID: res.ID, Text: t})
			}
		}

		if len(contextTexts) == 0 {
			respond(w, http.StatusOK, apiResp{Success: true, Data: askResp{
				Answer:  "No relevant context found in the database.",
				Sources: nil,
			}})
			return
		}

		answer, err := rc.Ask(req.Question, contextTexts)
		if err != nil {
			respond(w, http.StatusInternalServerError, apiResp{Success: false, Error: "ollama: " + err.Error()})
			return
		}

		respond(w, http.StatusOK, apiResp{Success: true, Data: askResp{
			Answer:  answer,
			Sources: sources,
		}})
	}
}

func handlePersistSave(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.SaveSnapshot(); err != nil {
			respond(w, http.StatusInternalServerError, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true, Data: "checkpoint saved"})
	}
}

func handlePersistLoad(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Reset()
		if err := s.LoadSnapshot(); err != nil {
			respond(w, http.StatusInternalServerError, apiResp{Success: false, Error: err.Error()})
			return
		}
		respond(w, http.StatusOK, apiResp{Success: true, Data: "state reloaded from disk"})
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, apiResp{Success: true})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		println(r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func apiHandler(s *Store, ec *EmbedClient, rc *RAGClient, ts *TextStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/insert", handleInsert(s))
	mux.HandleFunc("/api/insert-text", handleInsertText(s, ec, ts))
	mux.HandleFunc("/api/remove", handleRemove(s))
	mux.HandleFunc("/api/get", handleGet(s))
	mux.HandleFunc("/api/getall", handleGetAll(s))
	mux.HandleFunc("/api/search", handleSearch(s))
	mux.HandleFunc("/api/search-text", handleSearchText(s, ec))
	mux.HandleFunc("/api/ask", handleAsk(s, ec, rc, ts))
	mux.HandleFunc("/api/persist/save", handlePersistSave(s))
	mux.HandleFunc("/api/persist/load", handlePersistLoad(s))
	return corsMiddleware(loggingMiddleware(mux))
}

func parsePort(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return "8080"
	}
	if p, err := strconv.Atoi(parts[len(parts)-1]); err == nil && p > 0 {
		return strconv.Itoa(p)
	}
	return "8080"
}
