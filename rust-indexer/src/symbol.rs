use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct CodeSymbol {
    pub id: String,
    pub symbol_name: String,
    pub symbol_type: String, // "function", "class", "method", "struct", "enum", etc.
    pub file_path: String,
    pub line_start: i32,
    pub line_end: i32,
    pub signature: Option<String>,
    pub dependencies: Vec<String>,
    pub exported: bool,
    pub visibility: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ExtractRequest {
    pub start_line: Option<i32>,
    pub end_line: Option<i32>,
}

#[derive(Debug, Serialize)]
pub struct ExtractResponse {
    pub symbols: Vec<CodeSymbol>,
    pub success: bool,
}
