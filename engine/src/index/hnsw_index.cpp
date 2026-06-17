#include "hnsw_index.hpp"

#include <stdexcept>

hnsw_index::hnsw_index(size_t dim, hnsw_distance_type dist_type,
                       size_t max_elements, size_t M, size_t ef_construction,
                       size_t random_seed)
    : dim_(dim), dist_type_(dist_type) {

  if (dim == 0) {
    throw std::invalid_argument("dimension must be > 0");
  }

  switch (dist_type) {
  case hnsw_distance_type::l2:
    space_ = new hnswlib::L2Space(dim);
    break;
  case hnsw_distance_type::ip:
    space_ = new hnswlib::InnerProductSpace(dim);
    break;
  }

  index_ = new hnswlib::HierarchicalNSW<float>(space_, max_elements, M,
                                                ef_construction, random_seed);
}

hnsw_index::~hnsw_index() {
  delete index_;
  delete space_;
}

bool hnsw_index::insert(std::uint64_t id, const std::vector<float> &vector) {
  if (vector.size() != dim_) {
    return false;
  }
  try {
    std::lock_guard<std::mutex> lock(mtx_);
    index_->addPoint(vector.data(), id);
    return true;
  } catch (const std::exception &) {
    return false;
  }
}

std::vector<hnsw_search_result>
hnsw_index::search(const std::vector<float> &query, size_t k, size_t ef) {
  std::vector<hnsw_search_result> results;
  if (query.size() != dim_ || k == 0) {
    return results;
  }

  try {
    std::lock_guard<std::mutex> lock(mtx_);
    index_->setEf(ef);
    auto result = index_->searchKnnCloserFirst(query.data(), k);

    results.reserve(result.size());
    for (const auto &[score, id] : result) {
      results.push_back({static_cast<std::uint64_t>(id), score});
    }
  } catch (const std::exception &) {
  }

  return results;
}

bool hnsw_index::save(const std::string &path) {
  try {
    std::lock_guard<std::mutex> lock(mtx_);
    index_->saveIndex(path);
    return true;
  } catch (const std::exception &) {
    return false;
  }
}

bool hnsw_index::load(const std::string &path, size_t max_elements) {
  try {
    std::lock_guard<std::mutex> lock(mtx_);

    delete index_;
    index_ = nullptr;
    delete space_;
    space_ = nullptr;

    switch (dist_type_) {
    case hnsw_distance_type::l2:
      space_ = new hnswlib::L2Space(dim_);
      break;
    case hnsw_distance_type::ip:
      space_ = new hnswlib::InnerProductSpace(dim_);
      break;
    }

    if (max_elements > 0) {
      index_ = new hnswlib::HierarchicalNSW<float>(space_, path, false,
                                                    max_elements);
    } else {
      index_ = new hnswlib::HierarchicalNSW<float>(space_, path);
    }

    // hnswlib's loadIndex sets max_elements_ = 0 when loading an empty
    // index with max_elements_i = 0, which prevents further inserts.
    // Resize to a reasonable default if that happened.
    if (index_->max_elements_ == 0) {
      index_->resizeIndex(10000);
    }

    return true;
  } catch (const std::exception &) {
    return false;
  }
}

size_t hnsw_index::size() const {
  std::lock_guard<std::mutex> lock(mtx_);
  return index_->cur_element_count;
}

size_t hnsw_index::dimension() const { return dim_; }

void hnsw_index::set_ef(size_t ef) {
  std::lock_guard<std::mutex> lock(mtx_);
  index_->setEf(ef);
}
