#pragma once

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Opaque handle */
typedef void* vector_db_t;

/* Error codes */
#define VDB_OK             0
#define VDB_NOT_FOUND      1
#define VDB_DUPLICATE      2
#define VDB_INVALID_VECTOR 3

/* Distance types */
#define VDB_DIST_L2     0
#define VDB_DIST_COSINE 1
#define VDB_DIST_DOT    2

/* Create / destroy */
vector_db_t vdb_create(void);
void        vdb_destroy(vector_db_t db);

/* CRUD */
int  vdb_insert(vector_db_t db, uint64_t id, const float* values, size_t dim);
int  vdb_remove(vector_db_t db, uint64_t id);
int  vdb_get(vector_db_t db, uint64_t id, float** out_values, size_t* out_dim);
void vdb_free_buffer(float* values);

/* Get all */
typedef struct {
  uint64_t* ids;
  float*    values;   /* flat: [vec0_0, vec0_1, ..., vec1_0, vec1_1, ...] */
  size_t*   dims;
  size_t    count;
} vdb_vectors_t;

vdb_vectors_t vdb_get_all(vector_db_t db);
void          vdb_free_vectors(vdb_vectors_t* v);

/* Search */
typedef struct {
  uint64_t* ids;
  float*    scores;
  size_t    count;
} vdb_search_result_t;

vdb_search_result_t vdb_search(vector_db_t db, const float* query,
                                size_t dim, size_t k, int distance_type);
void vdb_free_search(vdb_search_result_t* r);

/* Persistence — WAL */
int  vdb_enable_wal(vector_db_t db, const char* path);
void vdb_disable_wal(vector_db_t db);
int  vdb_replay_wal(vector_db_t db);
int  vdb_truncate_wal(vector_db_t db);

/* Persistence — snapshot */
int vdb_snapshot_save(vector_db_t db, const char* path);
int vdb_snapshot_load(vector_db_t db, const char* path);

/* Atomically save snapshot + truncate WAL */
int vdb_checkpoint(vector_db_t db, const char* snapshot_path);

/* ------------------------------------------------------------------ */
/*  HNSW index (ANN)                                                   */
/* ------------------------------------------------------------------ */

/* Distance types for HNSW */
#define VDB_HNSW_L2 0
#define VDB_HNSW_IP 1   /* inner product */

/* Opaque handle for HNSW index */
typedef void* hnsw_index_t;

/* Create a new HNSW index.
   dim — vector dimensionality
   dist_type — VDB_HNSW_L2 or VDB_HNSW_IP
   max_elements — initial capacity
   M — HNSW graph parameter (default 16)
   ef_construction — build quality (default 200) */
hnsw_index_t vdb_hnsw_create(size_t dim, int dist_type,
                              size_t max_elements, size_t M,
                              size_t ef_construction);

/* Destroy an HNSW index */
void vdb_hnsw_destroy(hnsw_index_t idx);

/* Insert a vector. Returns 0 on success, non-zero on failure. */
int vdb_hnsw_insert(hnsw_index_t idx, uint64_t id,
                    const float* values, size_t dim);

/* Search k-nearest neighbors.
   query — query vector
   dim — dimension
   k — number of results
   ef — search width (higher = more accurate but slower)
   out_ids — output array of ids (caller must free with vdb_free_buffer)
   out_scores — output array of scores (caller must free with vdb_free_buffer)
   out_count — number of results returned
   Returns 0 on success. */
int vdb_hnsw_search(hnsw_index_t idx, const float* query,
                    size_t dim, size_t k, size_t ef,
                    uint64_t** out_ids, float** out_scores,
                    size_t* out_count);

/* Save index to file. Returns 0 on success. */
int vdb_hnsw_save(hnsw_index_t idx, const char* path);

/* Load index from file. dim must match the saved index.
   max_elements — if > 0, resize capacity. */
int vdb_hnsw_load(hnsw_index_t idx, const char* path,
                  size_t dim, size_t max_elements);

/* Return number of elements in the index. */
size_t vdb_hnsw_size(hnsw_index_t idx);

#ifdef __cplusplus
}
#endif
