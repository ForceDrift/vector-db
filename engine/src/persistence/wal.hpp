#pragma once

#include <cstdint>
#include <string>
#include <vector>

#include "../storage/vector_store.hpp"

class wal {
public:
  explicit wal(std::string path);
  ~wal();

  wal(const wal &) = delete;
  wal &operator=(const wal &) = delete;
  wal(wal &&) = default;
  wal &operator=(wal &&) = default;

  void log_insert(uint64_t id, const std::vector<float> &vector);
  void log_remove(uint64_t id);
  void replay(vector_store &store);
  void truncate();

private:
  enum class entry_type : uint8_t { insert = 0, remove = 1 };

  void write_entry(entry_type type, uint64_t id, const float *data,
                   std::size_t dim);

  std::string path_;
  int fd_;
};
