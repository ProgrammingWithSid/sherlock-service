# Build stage
FROM rust:1.75-slim AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy Cargo files (Cargo.lock will be generated during build if missing)
COPY Cargo.toml ./

# Copy source
COPY src ./src

# Build release (cargo will generate Cargo.lock automatically)
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/target/release/sherlock-indexer /app/sherlock-indexer

EXPOSE 8081

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

CMD ["/app/sherlock-indexer"]
