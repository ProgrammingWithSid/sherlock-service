use crate::symbol::CodeSymbol;
use anyhow::{Context, Result};
use std::path::Path;
use tree_sitter::{Language, Parser};
use tree_sitter_rust as ts_rust;
use tree_sitter_javascript as ts_js;
use tree_sitter_typescript as ts_ts;
use tree_sitter_go as ts_go;
use tree_sitter_python as ts_py;
use tree_sitter_java as ts_java;
use tree_sitter_cpp as ts_cpp;

pub struct ParserService {
    parsers: std::collections::HashMap<String, Language>,
}

impl ParserService {
    pub fn new() -> Self {
        let mut parsers = std::collections::HashMap::new();
        
        // Initialize parsers for each language
        parsers.insert("rust".to_string(), ts_rust::language());
        parsers.insert("javascript".to_string(), ts_js::language());
        parsers.insert("typescript".to_string(), ts_ts::language_typescript());
        parsers.insert("tsx".to_string(), ts_ts::language_tsx());
        parsers.insert("go".to_string(), ts_go::language());
        parsers.insert("python".to_string(), ts_py::language());
        parsers.insert("java".to_string(), ts_java::language());
        parsers.insert("cpp".to_string(), ts_cpp::language());
        
        Self { parsers }
    }

    fn detect_language(file_path: &str) -> Option<String> {
        let ext = Path::new(file_path)
            .extension()?
            .to_str()?
            .to_lowercase();
        
        match ext.as_str() {
            "rs" => Some("rust".to_string()),
            "js" | "jsx" | "mjs" | "cjs" => Some("javascript".to_string()),
            "ts" => Some("typescript".to_string()),
            "tsx" => Some("tsx".to_string()),
            "go" => Some("go".to_string()),
            "py" => Some("python".to_string()),
            "java" => Some("java".to_string()),
            "cpp" | "cc" | "cxx" | "c" | "h" | "hpp" => Some("cpp".to_string()),
            _ => None,
        }
    }

    pub async fn extract_symbols(&self, file_path: &str) -> Result<Vec<CodeSymbol>> {
        let language_name = Self::detect_language(file_path)
            .context("Unsupported file type")?;
        
        let language = self.parsers.get(&language_name)
            .context("Language parser not available")?;

        let source_code = tokio::fs::read_to_string(file_path).await
            .context("Failed to read file")?;

        let mut parser = Parser::new();
        parser.set_language(*language)?;

        let tree = parser.parse(&source_code, None)
            .context("Failed to parse file")?;

        let root_node = tree.root_node();
        let symbols = self.extract_from_tree(&root_node, &source_code, file_path, &language_name)?;

        Ok(symbols)
    }

    pub async fn extract_dependencies(&self, file_path: &str) -> Result<Vec<CodeSymbol>> {
        // For now, return empty - will implement dependency extraction
        // This would analyze imports/use statements
        Ok(vec![])
    }

    pub async fn get_chunk_hash(&self, file_path: &str, start_line: Option<i32>, end_line: Option<i32>) -> Result<String> {
        let source_code = tokio::fs::read_to_string(file_path).await
            .context("Failed to read file")?;

        let lines: Vec<&str> = source_code.lines().collect();
        let start = start_line.unwrap_or(1).max(1) as usize - 1;
        let end = end_line.unwrap_or(lines.len() as i32).min(lines.len() as i32) as usize;

        if start >= lines.len() || end > lines.len() || start >= end {
            return Err(anyhow::anyhow!("Invalid line range"));
        }

        let chunk: String = lines[start..end].join("\n");
        
        // Simple hash (in production, use SHA256)
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        let mut hasher = DefaultHasher::new();
        chunk.hash(&mut hasher);
        Ok(format!("{:x}", hasher.finish()))
    }

    fn extract_from_tree(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        language: &str,
    ) -> Result<Vec<CodeSymbol>> {
        let mut symbols = Vec::new();
        self.walk_tree(node, source, file_path, language, &mut symbols)?;
        Ok(symbols)
    }

    fn walk_tree(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        language: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        // Extract symbols based on language
        match language {
            "rust" => self.extract_rust_symbols(node, source, file_path, symbols)?,
            "javascript" | "typescript" | "tsx" => self.extract_js_symbols(node, source, file_path, symbols)?,
            "go" => self.extract_go_symbols(node, source, file_path, symbols)?,
            "python" => self.extract_python_symbols(node, source, file_path, symbols)?,
            "java" => self.extract_java_symbols(node, source, file_path, symbols)?,
            "cpp" => self.extract_cpp_symbols(node, source, file_path, symbols)?,
            _ => {}
        }

        // Recursively process children
        for i in 0..node.child_count() {
            if let Some(child) = node.child(i) {
                self.walk_tree(&child, source, file_path, language, symbols)?;
            }
        }

        Ok(())
    }

    fn extract_rust_symbols(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        match node.kind() {
            "function_item" | "impl_item" | "struct_item" | "enum_item" | "trait_item" | "type_item" | "const_item" | "static_item" => {
                if let Some(name_node) = node.child_by_field_name("name") {
                    let name = name_node.utf8_text(source.as_bytes())?.to_string();
                    let symbol_type = match node.kind() {
                        "function_item" => "function",
                        "impl_item" => "impl",
                        "struct_item" => "struct",
                        "enum_item" => "enum",
                        "trait_item" => "trait",
                        "type_item" => "type",
                        "const_item" => "const",
                        "static_item" => "static",
                        _ => "unknown",
                    };

                    // Check if exported (pub keyword)
                    let exported = node.child(0)
                        .map(|n| n.kind() == "visibility_modifier")
                        .unwrap_or(false);

                    let signature = self.extract_signature(node, source).ok();
                    symbols.push(CodeSymbol {
                        id: format!("{}_{}_{}", file_path, name, node.start_position().row),
                        symbol_name: name,
                        symbol_type: symbol_type.to_string(),
                        file_path: file_path.to_string(),
                        line_start: node.start_position().row as i32 + 1,
                        line_end: node.end_position().row as i32 + 1,
                        signature,
                        dependencies: vec![],
                        exported,
                        visibility: if exported { Some("public".to_string()) } else { Some("private".to_string()) },
                    });
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn extract_js_symbols(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        match node.kind() {
            "function_declaration" | "function" | "method_definition" | "class_declaration" | "variable_declaration" => {
                if let Some(name_node) = node.child_by_field_name("name") {
                    let name = name_node.utf8_text(source.as_bytes())?.to_string();
                    let symbol_type = match node.kind() {
                        "function_declaration" | "function" => "function",
                        "method_definition" => "method",
                        "class_declaration" => "class",
                        "variable_declaration" => "variable",
                        _ => "unknown",
                    };

                    let signature = self.extract_signature(node, source).ok();
                    symbols.push(CodeSymbol {
                        id: format!("{}_{}_{}", file_path, name, node.start_position().row),
                        symbol_name: name,
                        symbol_type: symbol_type.to_string(),
                        file_path: file_path.to_string(),
                        line_start: node.start_position().row as i32 + 1,
                        line_end: node.end_position().row as i32 + 1,
                        signature,
                        dependencies: vec![],
                        exported: false, // Would need to check export keyword
                        visibility: None,
                    });
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn extract_go_symbols(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        match node.kind() {
            "function_declaration" | "method_declaration" | "type_declaration" => {
                if let Some(name_node) = node.child_by_field_name("name") {
                    let name = name_node.utf8_text(source.as_bytes())?.to_string();
                    let symbol_type = match node.kind() {
                        "function_declaration" => "function",
                        "method_declaration" => "method",
                        "type_declaration" => "type",
                        _ => "unknown",
                    };

                    let signature = self.extract_signature(node, source).ok();
                    symbols.push(CodeSymbol {
                        id: format!("{}_{}_{}", file_path, name, node.start_position().row),
                        symbol_name: name,
                        symbol_type: symbol_type.to_string(),
                        file_path: file_path.to_string(),
                        line_start: node.start_position().row as i32 + 1,
                        line_end: node.end_position().row as i32 + 1,
                        signature,
                        dependencies: vec![],
                        exported: name.chars().next().map(|c| c.is_uppercase()).unwrap_or(false),
                        visibility: None,
                    });
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn extract_python_symbols(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        match node.kind() {
            "function_definition" | "class_definition" => {
                if let Some(name_node) = node.child_by_field_name("name") {
                    let name = name_node.utf8_text(source.as_bytes())?.to_string();
                    let symbol_type = match node.kind() {
                        "function_definition" => "function",
                        "class_definition" => "class",
                        _ => "unknown",
                    };

                    symbols.push(CodeSymbol {
                        id: format!("{}_{}_{}", file_path, name, node.start_position().row),
                        symbol_name: name,
                        symbol_type: symbol_type.to_string(),
                        file_path: file_path.to_string(),
                        line_start: node.start_position().row as i32 + 1,
                        line_end: node.end_position().row as i32 + 1,
                        signature: Some(self.extract_signature(node, source)?),
                        dependencies: vec![],
                        exported: false,
                        visibility: None,
                    });
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn extract_java_symbols(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        match node.kind() {
            "class_declaration" | "interface_declaration" | "method_declaration" => {
                if let Some(name_node) = node.child_by_field_name("name") {
                    let name = name_node.utf8_text(source.as_bytes())?.to_string();
                    let symbol_type = match node.kind() {
                        "class_declaration" => "class",
                        "interface_declaration" => "interface",
                        "method_declaration" => "method",
                        _ => "unknown",
                    };

                    let signature = self.extract_signature(node, source).ok();
                    symbols.push(CodeSymbol {
                        id: format!("{}_{}_{}", file_path, name, node.start_position().row),
                        symbol_name: name,
                        symbol_type: symbol_type.to_string(),
                        file_path: file_path.to_string(),
                        line_start: node.start_position().row as i32 + 1,
                        line_end: node.end_position().row as i32 + 1,
                        signature,
                        dependencies: vec![],
                        exported: true, // Java methods are typically public
                        visibility: Some("public".to_string()),
                    });
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn extract_cpp_symbols(
        &self,
        node: &tree_sitter::Node,
        source: &str,
        file_path: &str,
        symbols: &mut Vec<CodeSymbol>,
    ) -> Result<()> {
        match node.kind() {
            "function_definition" | "class_specifier" | "namespace_definition" => {
                if let Some(name_node) = node.child_by_field_name("name") {
                    let name = name_node.utf8_text(source.as_bytes())?.to_string();
                    let symbol_type = match node.kind() {
                        "function_definition" => "function",
                        "class_specifier" => "class",
                        "namespace_definition" => "namespace",
                        _ => "unknown",
                    };

                    symbols.push(CodeSymbol {
                        id: format!("{}_{}_{}", file_path, name, node.start_position().row),
                        symbol_name: name,
                        symbol_type: symbol_type.to_string(),
                        file_path: file_path.to_string(),
                        line_start: node.start_position().row as i32 + 1,
                        line_end: node.end_position().row as i32 + 1,
                        signature: Some(self.extract_signature(node, source)?),
                        dependencies: vec![],
                        exported: false,
                        visibility: None,
                    });
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn extract_signature(&self, node: &tree_sitter::Node, source: &str) -> Result<String> {
        let start_byte = node.start_byte();
        let end_byte = node.end_byte().min(source.len());
        
        // Extract first line as signature (simplified)
        let text = &source[start_byte..end_byte];
        let first_line = text.lines().next().unwrap_or("").trim();
        Ok(first_line.to_string())
    }
}

