#include "wal.hpp"

#include <fcntl.h>
#include <unistd.h>

#include <cstring>

wal::wal(std::string path) : path_(std::move(path)), fd_(-1) {
  fd_ = ::open(path_.c_str(), O_CREAT | O_RDWR | O_APPEND, 0644);
}

wal::~wal() {
  if (fd_ != -1) {
    ::close(fd_);
  }
}

void wal::write_entry(entry_type type, uint64_t id, const float *data,
                      std::size_t dim) {
  auto t = static_cast<uint8_t>(type);
  ::write(fd_, &t, sizeof(t));
  ::write(fd_, &id, sizeof(id));
  auto d = static_cast<uint64_t>(dim);
  ::write(fd_, &d, sizeof(d));
  if (data && dim > 0) {
    ::write(fd_, data, dim * sizeof(float));
  }
  ::fsync(fd_);
}

void wal::log_insert(uint64_t id, const std::vector<float> &vector) {
  write_entry(entry_type::insert, id, vector.data(), vector.size());
}

void wal::log_remove(uint64_t id) {
  write_entry(entry_type::remove, id, nullptr, 0);
}

void wal::replay(vector_store &store) {
  int rfd = ::open(path_.c_str(), O_RDONLY);
  if (rfd == -1) {
    return;
  }

  uint8_t t;
  uint64_t id;
  uint64_t dim;

  while (true) {
    if (::read(rfd, &t, sizeof(t)) != static_cast<ssize_t>(sizeof(t))) {
      break;
    }
    if (::read(rfd, &id, sizeof(id)) != static_cast<ssize_t>(sizeof(id))) {
      break;
    }
    if (::read(rfd, &dim, sizeof(dim)) != static_cast<ssize_t>(sizeof(dim))) {
      break;
    }

    if (static_cast<entry_type>(t) == entry_type::insert) {
      std::vector<float> vec(dim);
      if (dim > 0 && ::read(rfd, vec.data(), dim * sizeof(float)) !=
                         static_cast<ssize_t>(dim * sizeof(float))) {
        break;
      }
      store.insert(id, vec);
    } else {
      store.remove(id);
    }
  }

  ::close(rfd);
}

void wal::truncate() {
  ::ftruncate(fd_, 0);
  ::lseek(fd_, 0, SEEK_SET);
}
