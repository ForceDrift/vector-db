package main

/*
#cgo CFLAGS: -I../engine/include
#cgo LDFLAGS: -L../engine -lvectordb
#include "vector_db_c_api.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
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
	db C.vector_db_t
}

func NewStore() *Store {
	return &Store{db: C.vdb_create()}
}

func (s *Store) Close() {
	if s.db != nil {
		C.vdb_destroy(s.db)
		s.db = nil
	}
}

func float64sToC(values []float64) *C.float {
	buf := make([]C.float, len(values))
	for i, v := range values {
		buf[i] = C.float(v)
	}
	return &buf[0]
}

func cFloatsToFloat64s(c *C.float, n int) []float64 {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = float64(*(*C.float)(unsafe.Pointer(uintptr(unsafe.Pointer(c)) + uintptr(i)*unsafe.Sizeof(*c))))
	}
	return out
}

func (s *Store) Insert(id uint64, values []float64) error {
	if len(values) == 0 {
		return fmt.Errorf("empty vector")
	}
	cValues := float64sToC(values)
	code := C.vdb_insert(s.db, C.uint64_t(id), cValues, C.size_t(len(values)))
	switch code {
	case C.VDB_OK:
		return nil
	case C.VDB_DUPLICATE:
		return fmt.Errorf("duplicate id %d", id)
	case C.VDB_INVALID_VECTOR:
		return fmt.Errorf("invalid vector")
	default:
		return fmt.Errorf("unknown error %d", code)
	}
}

func (s *Store) Remove(id uint64) error {
	code := C.vdb_remove(s.db, C.uint64_t(id))
	switch code {
	case C.VDB_OK:
		return nil
	case C.VDB_NOT_FOUND:
		return fmt.Errorf("id %d not found", id)
	default:
		return fmt.Errorf("unknown error %d", code)
	}
}

func (s *Store) Get(id uint64) ([]float64, error) {
	var cValues *C.float
	var cDim C.size_t
	ret := C.vdb_get(s.db, C.uint64_t(id), &cValues, &cDim)
	if ret != 0 {
		return nil, fmt.Errorf("id %d not found", id)
	}
	defer C.vdb_free_buffer(cValues)

	return cFloatsToFloat64s(cValues, int(cDim)), nil
}

func (s *Store) GetAll() []Vector {
	cVec := C.vdb_get_all(s.db)
	defer C.vdb_free_vectors(&cVec)

	count := int(cVec.count)
	if count == 0 {
		return nil
	}

	cIDs := unsafe.Slice(cVec.ids, count)
	cDims := unsafe.Slice(cVec.dims, count)

	total := 0
	for i := 0; i < count; i++ {
		total += int(cDims[i])
	}
	cVals := unsafe.Slice(cVec.values, total)

	out := make([]Vector, count)
	vi := 0
	for i := 0; i < count; i++ {
		dim := int(cDims[i])
		vec := make([]float64, dim)
		for j := 0; j < dim; j++ {
			vec[j] = float64(cVals[vi])
			vi++
		}
		out[i] = Vector{ID: uint64(cIDs[i]), Values: vec}
	}
	return out
}

func (s *Store) Search(query []float64, k int, distance DistanceType) ([]SearchResult, error) {
	if k <= 0 {
		return nil, nil
	}

	var ctype C.int
	switch distance {
	case L2:
		ctype = C.VDB_DIST_L2
	case Cosine:
		ctype = C.VDB_DIST_COSINE
	default:
		ctype = C.VDB_DIST_DOT
	}

	cQuery := float64sToC(query)
	result := C.vdb_search(s.db, cQuery, C.size_t(len(query)), C.size_t(k), ctype)
	defer C.vdb_free_search(&result)

	count := int(result.count)
	out := make([]SearchResult, count)
	cIDs := unsafe.Slice(result.ids, count)
	cScores := unsafe.Slice(result.scores, count)

	for i := 0; i < count; i++ {
		out[i] = SearchResult{ID: uint64(cIDs[i]), Score: float64(cScores[i])}
	}

	return out, nil
}

func (s *Store) Reset() {
	s.Close()
	s.db = C.vdb_create()
}
