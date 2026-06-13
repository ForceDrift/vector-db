package main

import (
	"fmt"
	"math"
	"sync"
)

type DistanceType string

const (
	L2     DistanceType = "l2"
	Cosine DistanceType = "cosine"
	Dot    DistanceType = "dot"
)

type Vector struct {
	ID     uint64    `json:"id"`
	Values []float64 `json:"values"`
}

type SearchResult struct {
	ID    uint64  `json:"id"`
	Score float64 `json:"score"`
}

type Store struct {
	mu     sync.RWMutex
	data   map[uint64][]float64
	dim    int
}

func NewStore() *Store {
	return &Store{data: make(map[uint64][]float64)}
}

func (s *Store) Insert(id uint64, values []float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(values) == 0 {
		return fmt.Errorf("empty vector")
	}
	if _, exists := s.data[id]; exists {
		return fmt.Errorf("duplicate id %d", id)
	}
	if s.dim == 0 {
		s.dim = len(values)
	} else if len(values) != s.dim {
		return fmt.Errorf("expected dimension %d, got %d", s.dim, len(values))
	}
	vec := make([]float64, len(values))
	copy(vec, values)
	s.data[id] = vec
	return nil
}

func (s *Store) Remove(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[id]; !exists {
		return fmt.Errorf("id %d not found", id)
	}
	delete(s.data, id)
	return nil
}

func (s *Store) GetAll() []Vector {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Vector, 0, len(s.data))
	for id, vec := range s.data {
		v := make([]float64, len(vec))
		copy(v, vec)
		out = append(out, Vector{ID: id, Values: v})
	}
	return out
}

func (s *Store) Get(id uint64) ([]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vec, exists := s.data[id]
	if !exists {
		return nil, fmt.Errorf("id %d not found", id)
	}
	out := make([]float64, len(vec))
	copy(out, vec)
	return out, nil
}

func (s *Store) Search(query []float64, k int, distance DistanceType) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if k <= 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, k)
	var worst float64

	for id, vec := range s.data {
		d := computeDistance(query, vec, distance)
		if len(results) < k {
			results = append(results, SearchResult{ID: id, Score: d})
			if len(results) == k {
				worst = results[k-1].Score
				sortResults(results)
			}
		} else if d < worst {
			results[k-1] = SearchResult{ID: id, Score: d}
			sortResults(results)
			worst = results[k-1].Score
		}
	}

	return results, nil
}

func computeDistance(a, b []float64, dt DistanceType) float64 {
	switch dt {
	case L2:
		return l2(a, b)
	case Cosine:
		return cosine(a, b)
	default:
		return dot(a, b)
	}
}

func l2(a, b []float64) float64 {
	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

func cosine(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	return 1.0 - dot/(math.Sqrt(normA)*math.Sqrt(normB))
}

func dot(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func sortResults(r []SearchResult) {
	for i := 0; i < len(r); i++ {
		for j := i + 1; j < len(r); j++ {
			if r[j].Score < r[i].Score {
				r[i], r[j] = r[j], r[i]
			}
		}
	}
}
