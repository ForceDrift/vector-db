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

func apiHandler(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/insert", handleInsert(s))
	mux.HandleFunc("/api/remove", handleRemove(s))
	mux.HandleFunc("/api/get", handleGet(s))
	mux.HandleFunc("/api/getall", handleGetAll(s))
	mux.HandleFunc("/api/search", handleSearch(s))
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
