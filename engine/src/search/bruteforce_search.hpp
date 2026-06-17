#pragma once

#include <algorithm>
#include <cstdint>
#include <functional>
#include <queue>
#include <vector>

#include "../vector/distance.hpp"

struct search_result {
  std::uint64_t id;
  float score;
};

template <distance_type DistType> class bruteforce_search {
public:
  template <typename VectorRange>
  std::vector<search_result> search(const VectorRange &vectors,
                                    const std::vector<float> &query,
                                    std::size_t k) const;
};

template <distance_type DistType>
template <typename VectorRange>
std::vector<search_result>
bruteforce_search<DistType>::search(const VectorRange &vectors,
                                    const std::vector<float> &query,
                                    std::size_t k) const {
  if (k == 0) {
    return {};
  }

  distance<DistType> dist;
  using pair_t = std::pair<float, std::uint64_t>;

  // L2 & Cosine: lower score = more similar → max-heap pops largest (worst)
  // Dot:         higher score = more similar → min-heap pops smallest (worst)
  static constexpr bool lower_is_better =
      DistType == distance_type::l2 || DistType == distance_type::cosine;

  using heap_type = std::conditional_t<
      lower_is_better,
      std::priority_queue<pair_t>,
      std::priority_queue<pair_t, std::vector<pair_t>, std::greater<pair_t>>>;

  heap_type heap;

  for (const auto &[id, vec] : vectors) {
    float d = dist.compute(query, vec);
    heap.emplace(d, id);
    if (heap.size() > k) {
      heap.pop();
    }
  }

  std::vector<search_result> results;
  results.reserve(heap.size());
  while (!heap.empty()) {
    results.push_back({heap.top().second, heap.top().first});
    heap.pop();
  }
  std::reverse(results.begin(), results.end());
  return results;
}
