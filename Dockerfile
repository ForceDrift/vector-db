# ============================================================
# Stage 1: Build C++ shared library
# ============================================================
FROM ubuntu:22.04 AS cpp-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY engine/ .

RUN g++ -std=c++17 -shared -fPIC -I include -o libvectordb.so \
    src/c_api.cpp \
    src/storage/vector_store.cpp \
    src/search/bruteforce_search.cpp \
    src/persistence/wal.cpp \
    src/persistence/snapshot.cpp

# ============================================================
# Stage 2: Build Go API (with cgo)
# ============================================================
FROM golang:1.24 AS go-builder

WORKDIR /app
COPY api/ .

COPY --from=cpp-builder /build/libvectordb.so /usr/local/lib/
COPY engine/include/vector_db_c_api.h /usr/local/include/

ENV CGO_ENABLED=1
RUN go build -o server .

# ============================================================
# Stage 3: Minimal runtime image
# ============================================================
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /app/server /server
COPY --from=cpp-builder /build/libvectordb.so /usr/local/lib/

RUN ldconfig

EXPOSE 8080

CMD ["/server"]
