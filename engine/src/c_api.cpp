#include "vector_db_c_api.h"

#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <mutex>
#include <string>
#include <vector>

#include "persistence/snapshot.hpp"
#include "persistence/wal.hpp"
#include "search/bruteforce_search.hpp"
#include "storage/vector_store.hpp"

#include "index/hnsw_index.hpp"

/* ------------------------------------------------------------------ */
/*  Internal context                                                   */
/* ------------------------------------------------------------------ */

struct vdb_ctx {
  vector_store store;
  wal *wal_ptr = nullptr;
  std::mutex mtx;
};

static vdb_ctx *as_ctx(vector_db_t db) {
  return static_cast<vdb_ctx *>(db);
}

static vector_store &as_store(vector_db_t db) {
  return as_ctx(db)->store;
}

/* ------------------------------------------------------------------ */
/*  Create / destroy                                                   */
/* ------------------------------------------------------------------ */

vector_db_t vdb_create(void) {
  return new vdb_ctx();
}

void vdb_destroy(vector_db_t db) {
  auto ctx = as_ctx(db);
  {
    std::lock_guard<std::mutex> lock(ctx->mtx);
    delete ctx->wal_ptr;
    ctx->wal_ptr = nullptr;
  }
  delete ctx;
}

/* ------------------------------------------------------------------ */
/*  CRUD                                                               */
/* ------------------------------------------------------------------ */

int vdb_insert(vector_db_t db, uint64_t id, const float *values,
               size_t dim) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  std::vector<float> vec(values, values + dim);
  int code = static_cast<int>(ctx->store.insert(id, vec));
  if (code == VDB_OK && ctx->wal_ptr) {
    ctx->wal_ptr->log_insert(id, vec);
  }
  return code;
}

int vdb_remove(vector_db_t db, uint64_t id) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  int code = static_cast<int>(ctx->store.remove(id));
  if (code == VDB_OK && ctx->wal_ptr) {
    ctx->wal_ptr->log_remove(id);
  }
  return code;
}

int vdb_get(vector_db_t db, uint64_t id, float **out_values,
            size_t *out_dim) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  auto opt = ctx->store.get(id);
  if (!opt.has_value()) {
    *out_values = nullptr;
    *out_dim = 0;
    return -1;
  }
  auto &vec = opt.value();
  *out_dim = vec.size();
  *out_values = static_cast<float *>(std::malloc(vec.size() * sizeof(float)));
  std::memcpy(*out_values, vec.data(), vec.size() * sizeof(float));
  return 0;
}

void vdb_free_buffer(float *values) {
  std::free(values);
}

/* ------------------------------------------------------------------ */
/*  Get all                                                            */
/* ------------------------------------------------------------------ */

vdb_vectors_t vdb_get_all(vector_db_t db) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  auto &map = ctx->store.all();
  vdb_vectors_t result{};
  result.count = map.size();

  result.ids =
      static_cast<uint64_t *>(std::malloc(result.count * sizeof(uint64_t)));
  result.dims =
      static_cast<size_t *>(std::malloc(result.count * sizeof(size_t)));

  size_t total = 0;
  for (auto &[id, vec] : map) {
    total += vec.size();
  }

  result.values =
      static_cast<float *>(std::malloc(total * sizeof(float)));

  size_t vi = 0, ii = 0;
  for (auto &[id, vec] : map) {
    result.ids[ii] = id;
    result.dims[ii] = vec.size();
    std::memcpy(result.values + vi, vec.data(), vec.size() * sizeof(float));
    vi += vec.size();
    ++ii;
  }

  return result;
}

void vdb_free_vectors(vdb_vectors_t *v) {
  if (v) {
    std::free(v->ids);
    std::free(v->values);
    std::free(v->dims);
    v->ids = nullptr;
    v->values = nullptr;
    v->dims = nullptr;
    v->count = 0;
  }
}

/* ------------------------------------------------------------------ */
/*  Search                                                             */
/* ------------------------------------------------------------------ */

vdb_search_result_t vdb_search(vector_db_t db, const float *query,
                                size_t dim, size_t k, int distance_type) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  auto &map = ctx->store.all();

  using pair_t = std::pair<uint64_t, std::vector<float>>;
  std::vector<pair_t> data;
  data.reserve(map.size());
  for (auto &[id, vec] : map) {
    data.emplace_back(id, vec);
  }

  std::vector<float> qvec(query, query + dim);

  std::vector<search_result> results;
  if (distance_type == VDB_DIST_L2) {
    bruteforce_search<distance_type::l2> s;
    results = s.search(data, qvec, k);
  } else if (distance_type == VDB_DIST_COSINE) {
    bruteforce_search<distance_type::cosine> s;
    results = s.search(data, qvec, k);
  } else {
    bruteforce_search<distance_type::dot> s;
    results = s.search(data, qvec, k);
  }

  vdb_search_result_t out{};
  out.count = results.size();
  out.ids =
      static_cast<uint64_t *>(std::malloc(out.count * sizeof(uint64_t)));
  out.scores =
      static_cast<float *>(std::malloc(out.count * sizeof(float)));

  for (size_t i = 0; i < out.count; ++i) {
    out.ids[i] = results[i].id;
    out.scores[i] = results[i].score;
  }

  return out;
}

void vdb_free_search(vdb_search_result_t *r) {
  if (r) {
    std::free(r->ids);
    std::free(r->scores);
    r->ids = nullptr;
    r->scores = nullptr;
    r->count = 0;
  }
}

/* ------------------------------------------------------------------ */
/*  Persistence — WAL                                                  */
/* ------------------------------------------------------------------ */

int vdb_enable_wal(vector_db_t db, const char *path) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  if (ctx->wal_ptr) {
    delete ctx->wal_ptr;
  }
  ctx->wal_ptr = new wal(std::string(path));
  return ctx->wal_ptr ? VDB_OK : VDB_INVALID_VECTOR;
}

void vdb_disable_wal(vector_db_t db) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  delete ctx->wal_ptr;
  ctx->wal_ptr = nullptr;
}

int vdb_replay_wal(vector_db_t db) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  if (!ctx->wal_ptr) {
    return VDB_NOT_FOUND;
  }
  ctx->wal_ptr->replay(ctx->store);
  return VDB_OK;
}

int vdb_truncate_wal(vector_db_t db) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  if (!ctx->wal_ptr) {
    return VDB_NOT_FOUND;
  }
  ctx->wal_ptr->truncate();
  return VDB_OK;
}

/* ------------------------------------------------------------------ */
/*  Persistence — snapshot                                             */
/* ------------------------------------------------------------------ */

int vdb_snapshot_save(vector_db_t db, const char *path) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  if (snapshot::save(ctx->store, std::string(path))) {
    return VDB_OK;
  }
  return VDB_INVALID_VECTOR;
}

int vdb_snapshot_load(vector_db_t db, const char *path) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  if (snapshot::load(ctx->store, std::string(path))) {
    return VDB_OK;
  }
  return VDB_INVALID_VECTOR;
}

/* ------------------------------------------------------------------ */
/*  Checkpoint                                                         */
/* ------------------------------------------------------------------ */

int vdb_checkpoint(vector_db_t db, const char *snapshot_path) {
  auto ctx = as_ctx(db);
  std::lock_guard<std::mutex> lock(ctx->mtx);
  if (!snapshot::save(ctx->store, std::string(snapshot_path))) {
    return VDB_INVALID_VECTOR;
  }
  if (ctx->wal_ptr) {
    ctx->wal_ptr->truncate();
  }
  return VDB_OK;
}

/* ------------------------------------------------------------------ */
/*  HNSW index (ANN)                                                   */
/* ------------------------------------------------------------------ */

static hnsw_index *as_hnsw(hnsw_index_t idx) {
  return static_cast<hnsw_index *>(idx);
}

hnsw_index_t vdb_hnsw_create(size_t dim, int dist_type,
                              size_t max_elements, size_t M,
                              size_t ef_construction) {
  auto dtype = (dist_type == VDB_HNSW_IP) ? hnsw_distance_type::ip
                                          : hnsw_distance_type::l2;
  try {
    return new hnsw_index(dim, dtype, max_elements, M, ef_construction);
  } catch (...) {
    return nullptr;
  }
}

void vdb_hnsw_destroy(hnsw_index_t idx) {
  delete as_hnsw(idx);
}

int vdb_hnsw_insert(hnsw_index_t idx, uint64_t id,
                    const float *values, size_t dim) {
  auto ix = as_hnsw(idx);
  std::vector<float> vec(values, values + dim);
  return ix->insert(id, vec) ? VDB_OK : VDB_INVALID_VECTOR;
}

int vdb_hnsw_search(hnsw_index_t idx, const float *query,
                    size_t dim, size_t k, size_t ef,
                    uint64_t **out_ids, float **out_scores,
                    size_t *out_count) {
  auto ix = as_hnsw(idx);
  std::vector<float> qvec(query, query + dim);
  auto results = ix->search(qvec, k, ef);

  *out_count = results.size();
  if (results.empty()) {
    *out_ids = nullptr;
    *out_scores = nullptr;
    return 0;
  }

  *out_ids = static_cast<uint64_t *>(std::malloc(results.size() * sizeof(uint64_t)));
  *out_scores = static_cast<float *>(std::malloc(results.size() * sizeof(float)));

  for (size_t i = 0; i < results.size(); ++i) {
    (*out_ids)[i] = results[i].id;
    (*out_scores)[i] = results[i].score;
  }

  return VDB_OK;
}

int vdb_hnsw_save(hnsw_index_t idx, const char *path) {
  return as_hnsw(idx)->save(std::string(path)) ? VDB_OK : VDB_INVALID_VECTOR;
}

int vdb_hnsw_load(hnsw_index_t idx, const char *path,
                  size_t dim, size_t max_elements) {
  auto ix = as_hnsw(idx);
  return ix->load(std::string(path), max_elements) ? VDB_OK : VDB_INVALID_VECTOR;
}

size_t vdb_hnsw_size(hnsw_index_t idx) {
  return as_hnsw(idx)->size();
}
