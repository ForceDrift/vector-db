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

#ifdef __cplusplus
}
#endif
