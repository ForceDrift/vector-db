package main

/*
#cgo CFLAGS: -I/usr/local/include -I../engine/include
#cgo LDFLAGS: -L/usr/local/lib -L../engine -lvectordb
#include "vector_db_c_api.h"
#include <stdlib.h>
*/
import "C"

import (
	"os"
	"strings"
	"unsafe"
)

const defaultIndexPath = "data/hnsw.bin"

const (
	VDB_HNSW_L2 = 0
	VDB_HNSW_IP = 1
)

type HNSW struct {
	idx C.hnsw_index_t
	dim int
}

func NewHNSW(dim int, distType int, maxElements int) *HNSW {
	idx := C.vdb_hnsw_create(C.size_t(dim), C.int(distType),
		C.size_t(maxElements), 16, 200)
	if idx == nil {
		return nil
	}
	return &HNSW{idx: idx, dim: dim}
}

func (h *HNSW) Destroy() {
	if h.idx != nil {
		C.vdb_hnsw_destroy(h.idx)
		h.idx = nil
	}
}

func (h *HNSW) Insert(id uint64, values []float64) bool {
	if len(values) != h.dim {
		return false
	}
	cValues := make([]C.float, len(values))
	for i, v := range values {
		cValues[i] = C.float(v)
	}
	code := C.vdb_hnsw_insert(h.idx, C.uint64_t(id), &cValues[0], C.size_t(len(values)))
	return code == 0
}

func (h *HNSW) Search(query []float64, k int, ef int) ([]uint64, []float64) {
	if len(query) != h.dim {
		return nil, nil
	}
	cQuery := make([]C.float, len(query))
	for i, v := range query {
		cQuery[i] = C.float(v)
	}

	var cIDs *C.uint64_t
	var cScores *C.float
	var cCount C.size_t

	code := C.vdb_hnsw_search(h.idx, &cQuery[0], C.size_t(len(query)),
		C.size_t(k), C.size_t(ef), &cIDs, &cScores, &cCount)
	if code != 0 || cCount == 0 {
		return nil, nil
	}
	defer C.free(unsafe.Pointer(cIDs))
	defer C.free(unsafe.Pointer(cScores))

	count := int(cCount)
	ids := make([]uint64, count)
	scores := make([]float64, count)
	idSlice := unsafe.Slice(cIDs, count)
	scoreSlice := unsafe.Slice(cScores, count)
	for i := 0; i < count; i++ {
		ids[i] = uint64(idSlice[i])
		scores[i] = float64(scoreSlice[i])
	}
	return ids, scores
}

func (h *HNSW) Save(path string) bool {
	if dir := path[:max(0, strings.LastIndex(path, "/"))]; dir != "" {
		os.MkdirAll(dir, 0755)
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return C.vdb_hnsw_save(h.idx, cpath) == 0
}

func (h *HNSW) Load(path string, maxElements int) bool {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return C.vdb_hnsw_load(h.idx, cpath, C.size_t(h.dim), C.size_t(maxElements)) == 0
}

func (h *HNSW) Size() int {
	return int(C.vdb_hnsw_size(h.idx))
}

func scoreBar(score float64, ids []uint64, scores []float64) string {
	if len(scores) <= 1 {
		return ""
	}
	min, max := scores[0], scores[0]
	for _, s := range scores {
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}
	width := 15
	var normalized float64
	if max > min {
		normalized = (score - min) / (max - min)
	} else {
		normalized = 0.5
	}
	filled := int(normalized * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return STYLE_GRAY + bar + STYLE_RESET
}
