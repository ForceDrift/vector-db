#include "vector_db_c_api.h"

#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <vector>

#include "search/bruteforce_search.hpp"
#include "storage/vector_store.hpp"

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

static vector_store *as_store(vector_db_t db) {
  return static_cast<vector_store *>(db);
}

/* ------------------------------------------------------------------ */
/*  Create / destroy                                                   */
/* ------------------------------------------------------------------ */

vector_db_t vdb_create(void) {
  return new vector_store();
}

void vdb_destroy(vector_db_t db) {
  delete as_store(db);
}

/* ------------------------------------------------------------------ */
/*  CRUD                                                               */
/* ------------------------------------------------------------------ */

int vdb_insert(vector_db_t db, uint64_t id, const float *values,
               size_t dim) {
  std::vector<float> vec(values, values + dim);
  return static_cast<int>(as_store(db)->insert(id, vec));
}

int vdb_remove(vector_db_t db, uint64_t id) {
  return static_cast<int>(as_store(db)->remove(id));
}

int vdb_get(vector_db_t db, uint64_t id, float **out_values,
            size_t *out_dim) {
  auto opt = as_store(db)->get(id);
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
  auto &map = as_store(db)->all();
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
  auto &map = as_store(db)->all();

  /* Build vector of pairs for bruteforce_search input */
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
