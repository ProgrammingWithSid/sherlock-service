use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::Json,
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tower_http::cors::CorsLayer;

mod parser;
mod symbol;

use parser::ParserService;
use symbol::{CodeSymbol, ExtractRequest, ExtractResponse};

#[derive(Clone)]
struct AppState {
    parser: Arc<ParserService>,
}

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    let parser = Arc::new(ParserService::new());
    let state = AppState { parser };

    let app = Router::new()
        .route("/health", get(health_check))
        .route("/extract/:repo_path/*file_path", post(extract_symbols))
        .route("/extract-deps/:repo_path/*file_path", post(extract_dependencies))
        .route("/hash/:repo_path/*file_path", post(get_chunk_hash))
        .layer(CorsLayer::permissive())
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8081")
        .await
        .expect("Failed to bind to port 8081");

    tracing::info!("🚀 Rust indexer service listening on :8081");
    axum::serve(listener, app)
        .await
        .expect("Server failed");
}

async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "ok",
        "service": "sherlock-indexer",
        "version": "0.1.0"
    }))
}

async fn extract_symbols(
    State(state): State<AppState>,
    Path((repo_path, file_path)): Path<(String, String)>,
    Json(payload): Json<ExtractRequest>,
) -> Result<Json<ExtractResponse>, StatusCode> {
    let full_path = format!("{}/{}", repo_path, file_path);
    
    match state.parser.extract_symbols(&full_path).await {
        Ok(symbols) => Ok(Json(ExtractResponse {
            symbols,
            success: true,
        })),
        Err(e) => {
            tracing::error!("Failed to extract symbols: {}", e);
            Err(StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn extract_dependencies(
    State(state): State<AppState>,
    Path((repo_path, file_path)): Path<(String, String)>,
    Json(payload): Json<ExtractRequest>,
) -> Result<Json<ExtractResponse>, StatusCode> {
    let full_path = format!("{}/{}", repo_path, file_path);
    
    match state.parser.extract_dependencies(&full_path).await {
        Ok(deps) => Ok(Json(ExtractResponse {
            symbols: deps,
            success: true,
        })),
        Err(e) => {
            tracing::error!("Failed to extract dependencies: {}", e);
            Err(StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn get_chunk_hash(
    State(state): State<AppState>,
    Path((repo_path, file_path)): Path<(String, String)>,
    Json(payload): Json<ExtractRequest>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let full_path = format!("{}/{}", repo_path, file_path);
    
    match state.parser.get_chunk_hash(&full_path, payload.start_line, payload.end_line).await {
        Ok(hash) => Ok(Json(serde_json::json!({
            "hash": hash,
            "success": true
        }))),
        Err(e) => {
            tracing::error!("Failed to get chunk hash: {}", e);
            Err(StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

