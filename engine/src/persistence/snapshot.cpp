#include "snapshot.hpp"

#include <fcntl.h>
#include <unistd.h>

#include <cstring>

bool snapshot::save(const vector_store &store, const std::string &path) {
  int fd = ::open(path.c_str(), O_CREAT | O_WRONLY | O_TRUNC, 0644);
  if (fd == -1) {
    return false;
  }

  auto all = store.all();
  auto count = static_cast<uint64_t>(all.size());
  ::write(fd, &count, sizeof(count));

  for (const auto &[id, vec] : all) {
    auto dim = static_cast<uint64_t>(vec.size());
    ::write(fd, &id, sizeof(id));
    ::write(fd, &dim, sizeof(dim));
    if (dim > 0) {
      ::write(fd, vec.data(), dim * sizeof(float));
    }
  }

  ::fsync(fd);
  ::close(fd);
  return true;
}

bool snapshot::load(vector_store &store, const std::string &path) {
  int fd = ::open(path.c_str(), O_RDONLY);
  if (fd == -1) {
    return false;
  }

  uint64_t count;
  if (::read(fd, &count, sizeof(count)) != static_cast<ssize_t>(sizeof(count))) {
    ::close(fd);
    return false;
  }

  for (uint64_t i = 0; i < count; ++i) {
    uint64_t id;
    uint64_t dim;

    if (::read(fd, &id, sizeof(id)) != static_cast<ssize_t>(sizeof(id))) {
      break;
    }
    if (::read(fd, &dim, sizeof(dim)) != static_cast<ssize_t>(sizeof(dim))) {
      break;
    }

    std::vector<float> vec(dim);
    if (dim > 0 &&
        ::read(fd, vec.data(), dim * sizeof(float)) !=
            static_cast<ssize_t>(dim * sizeof(float))) {
      break;
    }
    store.insert(id, vec);
  }

  ::close(fd);
  return true;
}
