# syntax=docker/dockerfile:1

# ------------------------------
# Builder stage: build Drogon and the app
# ------------------------------
FROM ubuntu:24.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    git \
    ca-certificates \
    libjsoncpp-dev \
    uuid-dev \
    zlib1g-dev \
    libssl-dev \
    libpq-dev && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Build and install Drogon
WORKDIR /opt
RUN git clone --depth 1 https://github.com/drogonframework/drogon.git && \
    cd drogon && \
    git submodule update --init --recursive && \
    cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DBUILD_EXAMPLES=OFF -DBUILD_CTL=ON && \
    cmake --build build --target install

# Build the application
WORKDIR /src
COPY . .
# Clean any host build cache/directories copied into the image
RUN rm -rf build CMakeCache.txt && \
    cmake -S . -B build -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build --target transaction-service

# ------------------------------
# Runtime stage: minimal image to run the app
# ------------------------------
FROM ubuntu:24.04 AS runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libjsoncpp-dev \
    uuid-runtime \
    zlib1g \
    libssl3 \
    libpq5 && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy Drogon shared libs from builder
COPY --from=builder /usr/local/lib/libdrogon* /usr/local/lib/
COPY --from=builder /usr/local/lib/libtrantor* /usr/local/lib/
RUN ldconfig

# Copy app binary and config
WORKDIR /app
COPY --from=builder /src/build/transaction-service /app/transaction-service
COPY config.json /app/config.json

# Default DB host override for Docker Desktop on macOS
ENV DB_HOST=host.docker.internal

# Update config.json host with env (best-effort) and start server
ENTRYPOINT ["/bin/sh","-c","sed -i \"s/\\\"host\\\": \\\"localhost\\\"/\\\"host\\\": \\\"$DB_HOST\\\"/\" /app/config.json || true; exec /app/transaction-service"]

EXPOSE 5555