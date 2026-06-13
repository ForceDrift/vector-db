#pragma once

#include <cmath>
#include <cstddef>
#include <cstdint>
#include <pthread.h>
#include <vector>

enum class distance_type { l2, cosine, dot };

template <distance_type Type> class distance {
public:
  distance() = default;
  ~distance() = default;
  distance(distance &&) = default;
  distance(const distance &) = default;
  distance &operator=(distance &&) = default;
  distance &operator=(const distance &) = default;

  float compute(const std::vector<float> &a,
                const std::vector<float> &b) const {
    if constexpr (Type == distance_type::l2) {
      return calculate_l2(a, b);
    } else if constexpr (Type == distance_type::cosine) {
      return calculate_cosine(a, b);
    } else {
      return calculate_dot(a, b);
    }
  }

private:
  static constexpr float calculate_l2(const std::vector<float> &a,
                                      const std::vector<float> &b) noexcept {
    float sum = 0;
    for (auto i{0}; i < a.size(); ++i) {
      float diff = a[i] - b[i];
      sum += diff * diff;
    }
    return std::sqrt(sum);
  }

  static constexpr float
  calculate_cosine(const std::vector<float> &a,
                   const std::vector<float> &b) noexcept {
    float dot = 0, norm_a = 0, norm_b = 0;
    for (auto i{0}; i < a.size(); ++i) {
      dot += a[i] * b[i];
      norm_a += a[i] * a[i];
      norm_b += b[i] * b[i];
    }
    // 1 - a * b / |a||b|
    return 1.0f - dot / (std::sqrt(norm_a) * std::sqrt(norm_b));
  }

  static constexpr float calculate_dot(const std::vector<float> &a,
                                       const std::vector<float> &b) noexcept {
    float sum = 0;
    for (auto i{0}; i < a.size(); ++i) {
      sum += a[i] * b[i];
    }
    return sum;
  }
};
