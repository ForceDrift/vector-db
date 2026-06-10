#include <iostream>
#include <vector>

#include "src/search/bruteforce_search.hpp"

int main() {
  std::vector<std::pair<std::uint64_t, std::vector<float>>> data = {
      {1, {1.0f, 0.0f, 0.0f}}, {2, {0.0f, 1.0f, 0.0f}}, {3, {0.0f, 0.0f, 1.0f}},
      {4, {1.0f, 1.0f, 1.0f}}, {5, {2.0f, 2.0f, 2.0f}},
  };

  std::vector<float> query = {0.9f, 0.1f, 0.0f};

  std::cout << "Query: [" << query[0] << ", " << query[1] << ", " << query[2]
            << "]\n\n";

  // L2 distance
  {
    bruteforce_search<distance_type::l2> searcher;
    auto results = searcher.search(data, query, 3);

    std::cout << "--- L2 (top-3, lower = closer) ---\n";
    for (const auto &r : results) {
      std::cout << "  id=" << r.id << "  score=" << r.score << "\n";
    }
    std::cout << "\n";
  }

  // Cosine distance
  {
    bruteforce_search<distance_type::cosine> searcher;
    auto results = searcher.search(data, query, 3);

    std::cout << "--- Cosine (top-3, lower = closer) ---\n";
    for (const auto &r : results) {
      std::cout << "  id=" << r.id << "  score=" << r.score << "\n";
    }
    std::cout << "\n";
  }

  {
    bruteforce_search<distance_type::dot> searcher;
    auto results = searcher.search(data, query, 3);

    std::cout << "--- Dot product (top-3, higher = closer) ---\n";
    for (const auto &r : results) {
      std::cout << "  id=" << r.id << "  score=" << r.score << "\n";
    }
    std::cout << "\n";
  }

  {
    bruteforce_search<distance_type::l2> searcher;
    auto results = searcher.search(data, query, 100);

    std::cout << "--- L2 (top-100, returns all " << results.size()
              << " vectors) ---\n";
    for (const auto &r : results) {
      std::cout << "  id=" << r.id << "  score=" << r.score << "\n";
    }
    std::cout << "\n";
  }

  {
    bruteforce_search<distance_type::l2> searcher;
    auto results = searcher.search(data, query, 0);
    std::cout << "--- L2 (k=0, returns " << results.size() << " results) ---\n";
  }

  return 0;
}
