
#include "vector_store.hpp"

#include <cstdint>
#include <optional>
#include <vector>

errors vector_store::insert(uint64_t id, const std::vector<float> &vector) {
  if (vector.empty()) {
    return errors::invalid_vector;
  }
  if (store_.find(id) != store_.end()) {
    return errors::duplicate;
  }
  if (dim_ == 0) {
    dim_ = vector.size();
  } else if (vector.size() != dim_) {
    return errors::invalid_vector;
  }
  store_[id] = vector;
  return errors::ok;
}

errors vector_store::remove(uint64_t id) {
  if (store_.find(id) == store_.end()) {
    return errors::not_found;
  }
  store_.erase(id);
  return errors::ok;
}

std::optional<std::vector<float>> vector_store::get(uint64_t id) const {
  auto it = store_.find(id);
  if (it == store_.end()) {
    return std::nullopt;
  }
  return it->second;
}
