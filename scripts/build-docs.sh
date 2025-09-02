#!/bin/bash

# Build documentation script for Guvnor
# Converts markdown files to HTML using pandoc or basic conversion

set -e

DOCS_DIR="${1:-docs}"
BUILD_DIR="${2:-docs/site}"

echo "Building documentation..."
echo "Source: $DOCS_DIR"
echo "Output: $BUILD_DIR"

# Create build directory
mkdir -p "$BUILD_DIR"

# Check for pandoc
if command -v pandoc >/dev/null 2>&1; then
    echo "Using pandoc for conversion..."
    
    # Convert each markdown file to HTML
    for md_file in "$DOCS_DIR"/*.md; do
        if [[ -f "$md_file" ]]; then
            filename=$(basename "$md_file" .md)
            echo "Converting $filename.md -> $filename.html"
            
            # Choose template based on file type
            template="$DOCS_DIR/template-docs.html"
            if [[ "$filename" == "index" ]] || [[ "$filename" == "README" ]]; then
                template="$DOCS_DIR/template.html"
            fi
            
            pandoc "$md_file" \
                --from markdown \
                --to html5 \
                --standalone \
                --template="$template" \
                --output "$BUILD_DIR/$filename.html" \
                --metadata title="$filename - Guvnor Documentation" \
                --table-of-contents \
                --toc-depth=3 \
                2>/dev/null || \
            pandoc "$md_file" \
                --from markdown \
                --to html5 \
                --standalone \
                --template="$template" \
                --output "$BUILD_DIR/$filename.html" \
                --metadata title="$filename - Guvnor Documentation"
        fi
    done
    
    # Use custom index.html if it exists, otherwise create from README.md
    if [[ -f "$DOCS_DIR/index.html" ]]; then
        cp "$DOCS_DIR/index.html" "$BUILD_DIR/index.html"
    elif [[ -f "$BUILD_DIR/README.html" ]]; then
        cp "$BUILD_DIR/README.html" "$BUILD_DIR/index.html"
    fi
    
else
    echo "pandoc not found, using basic conversion..."
    
    # Basic HTML conversion without pandoc
    for md_file in "$DOCS_DIR"/*.md; do
        if [[ -f "$md_file" ]]; then
            filename=$(basename "$md_file" .md)
            echo "Converting $filename.md -> $filename.html"
            
            cat > "$BUILD_DIR/$filename.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Guvnor Documentation - $filename</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <header>
        <h1>Guvnor Documentation</h1>
        <nav>
            <a href="index.html">Home</a>
            <a href="configuration.html">Configuration</a>
            <a href="examples.html">Examples</a>
            <a href="systemd.html">SystemD</a>
        </nav>
    </header>
    <main>
        <pre>$(cat "$md_file")</pre>
    </main>
</body>
</html>
EOF
        fi
    done
    
    # Use custom index.html if it exists, otherwise create from README
    if [[ -f "$DOCS_DIR/index.html" ]]; then
        cp "$DOCS_DIR/index.html" "$BUILD_DIR/index.html"
    elif [[ -f "$BUILD_DIR/README.html" ]]; then
        cp "$BUILD_DIR/README.html" "$BUILD_DIR/index.html"
    fi
fi

# Copy custom index.html if exists (for non-pandoc mode too)
if [[ -f "$DOCS_DIR/index.html" ]]; then
    cp "$DOCS_DIR/index.html" "$BUILD_DIR/index.html"
fi

# Create CSS file
cat > "$BUILD_DIR/style.css" << 'EOF'
/* Guvnor Documentation Styles */
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    line-height: 1.6;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
    color: #333;
}

header {
    border-bottom: 1px solid #eee;
    margin-bottom: 2rem;
    padding-bottom: 1rem;
}

header h1 {
    color: #2c3e50;
    margin: 0 0 1rem 0;
}

nav a {
    margin-right: 1rem;
    color: #3498db;
    text-decoration: none;
}

nav a:hover {
    text-decoration: underline;
}

h1, h2, h3 {
    color: #2c3e50;
}

h1 {
    border-bottom: 2px solid #3498db;
    padding-bottom: 0.5rem;
}

h2 {
    border-bottom: 1px solid #bdc3c7;
    padding-bottom: 0.3rem;
}

code {
    background: #f8f9fa;
    padding: 0.2rem 0.4rem;
    border-radius: 3px;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

pre {
    background: #f8f9fa;
    padding: 1rem;
    border-radius: 5px;
    overflow-x: auto;
    border-left: 4px solid #3498db;
}

pre code {
    background: none;
    padding: 0;
}

blockquote {
    border-left: 4px solid #3498db;
    margin: 0;
    padding-left: 1rem;
    color: #666;
    font-style: italic;
}

table {
    border-collapse: collapse;
    width: 100%;
    margin: 1rem 0;
}

th, td {
    border: 1px solid #ddd;
    padding: 0.5rem;
    text-align: left;
}

th {
    background: #f8f9fa;
    font-weight: 600;
}

a {
    color: #3498db;
}

a:hover {
    color: #2980b9;
}

.highlight {
    background: #fff3cd;
    padding: 0.5rem;
    border-radius: 3px;
    border-left: 4px solid #ffc107;
}

@media (max-width: 768px) {
    body {
        padding: 10px;
    }
    
    nav a {
        display: block;
        margin: 0.5rem 0;
    }
}
EOF

echo "Documentation site built successfully!"
echo "Open $BUILD_DIR/index.html in your browser"