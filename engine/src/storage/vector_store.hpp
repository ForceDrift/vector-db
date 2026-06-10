#include <cstddef>
#include <cstdint>
#include <optional>
#include <type_traits>
#include <unordered_map>
#include <vector>

enum class errors { ok, not_found, duplicate, invalid_vector };

class vector_store {
private:
  std::unordered_map<std::uint64_t, std::vector<float>> store_;
  std::size_t dim_ = 0;

public:
  vector_store() = default;
  ~vector_store() = default;
  vector_store(vector_store &&) = delete;
  vector_store(const vector_store &) = delete;
  vector_store &operator=(vector_store &&) = default;
  vector_store &operator=(const vector_store &) = default;

  errors insert(uint64_t id, const std::vector<float> &vector);
  errors remove(uint64_t id);
  errors get(uint64_t id);
};

// for math do template metaprogarmming, simd cuda const expr, lambdas
