#pragma once

#include <cstdint>
#include <string>
#include <vector>

#include "../storage/vector_store.hpp"

class snapshot {
public:
  static bool save(const vector_store &store, const std::string &path);
  static bool load(vector_store &store, const std::string &path);
};
