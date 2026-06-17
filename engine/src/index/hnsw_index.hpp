#pragma once

#include <cstdint>
#include <mutex>
#include <string>
#include <vector>

#include "hnswlib/hnswlib.h"

enum class hnsw_distance_type { l2, ip };

struct hnsw_search_result {
  std::uint64_t id;
  float score;
};

class hnsw_index {
public:
  hnsw_index(size_t dim, hnsw_distance_type dist_type,
             size_t max_elements = 10000, size_t M = 16,
             size_t ef_construction = 200, size_t random_seed = 100);

  ~hnsw_index();

  hnsw_index(const hnsw_index &) = delete;
  hnsw_index &operator=(const hnsw_index &) = delete;
  hnsw_index(hnsw_index &&) = delete;
  hnsw_index &operator=(hnsw_index &&) = delete;

  bool insert(std::uint64_t id, const std::vector<float> &vector);

  std::vector<hnsw_search_result> search(const std::vector<float> &query,
                                         size_t k, size_t ef = 50);

  bool save(const std::string &path);
  bool load(const std::string &path, size_t max_elements = 0);

  size_t size() const;
  size_t dimension() const;
  void set_ef(size_t ef);

private:
  hnswlib::HierarchicalNSW<float> *index_ = nullptr;
  hnswlib::SpaceInterface<float> *space_ = nullptr;
  size_t dim_ = 0;
  hnsw_distance_type dist_type_;
  mutable std::mutex mtx_;
};
