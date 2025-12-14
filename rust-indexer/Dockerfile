# Build stage
FROM rust:1.75-slim as builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy Cargo files
COPY rust-indexer/Cargo.toml rust-indexer/Cargo.lock ./

# Copy source
COPY rust-indexer/src ./src

# Build release
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/target/release/sherlock-indexer /app/sherlock-indexer

# Install curl for healthcheck
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

EXPOSE 8081

CMD ["/app/sherlock-indexer"]

