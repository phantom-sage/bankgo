#!/bin/bash

# Bank REST API Documentation PDF Generator
# This script combines all markdown documentation files into a single PDF

set -e

# Configuration
DOCS_DIR="docs"
OUTPUT_DIR="docs/pdf"
OUTPUT_FILE="bank-rest-api-documentation.pdf"
TEMP_DIR="/tmp/bankapi-docs-$$"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if pandoc is installed
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v pandoc &> /dev/null; then
        log_error "pandoc is not installed. Please install it first:"
        echo "  macOS: brew install pandoc"
        echo "  Ubuntu/Debian: sudo apt-get install pandoc"
        echo "  Or visit: https://pandoc.org/installing.html"
        exit 1
    fi
    
    # Check for LaTeX (required for PDF generation)
    if ! command -v pdflatex &> /dev/null; then
        log_warning "pdflatex not found. Installing BasicTeX (recommended for PDF generation):"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            echo "  macOS: brew install --cask basictex"
        else
            echo "  Ubuntu/Debian: sudo apt-get install texlive-latex-base texlive-fonts-recommended"
        fi
        log_info "Continuing with HTML output instead..."
        USE_HTML=true
    fi
    
    log_success "Dependencies check completed"
}

# Create temporary directory
setup_temp_dir() {
    log_info "Setting up temporary directory: $TEMP_DIR"
    mkdir -p "$TEMP_DIR"
    mkdir -p "$OUTPUT_DIR"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "$TEMP_DIR"
}

# Set trap for cleanup
trap cleanup EXIT

# Generate table of contents
generate_toc() {
    log_info "Generating table of contents..."
    
    cat > "$TEMP_DIR/00-toc.md" << 'EOF'
# BankGo Complete Documentation

## Table of Contents

1. [Bank REST API Overview](#bank-rest-api-overview)
2. [Admin Dashboard Frontend](#admin-dashboard-frontend)
3. [API Reference](#api-reference)  
4. [Deployment Guide](#deployment-guide)
5. [Logging Guide](#logging-guide)
6. [Troubleshooting Guide](#troubleshooting-guide)

---

EOF
}

# Clean emojis and special characters for LaTeX compatibility
clean_for_latex() {
    local input_file="$1"
    local output_file="$2"
    
    # Remove emojis and replace with text equivalents - comprehensive pattern
    sed -E 's/🚀/[ROCKET]/g; s/📚/[BOOKS]/g; s/⚡/[LIGHTNING]/g; s/🔧/[WRENCH]/g; s/🛡️/[SHIELD]/g; s/📊/[CHART]/g; s/🎯/[TARGET]/g; s/✅/[CHECK]/g; s/❌/[X]/g; s/⚠️/[WARNING]/g; s/💡/[BULB]/g; s/🔍/[SEARCH]/g; s/📝/[MEMO]/g; s/🌟/[STAR]/g; s/🔒/[LOCK]/g; s/📋/[CLIPBOARD]/g; s/🐳/[DOCKER]/g; s/🔑/[KEY]/g; s/📦/[PACKAGE]/g; s/🌐/[GLOBE]/g; s/💻/[COMPUTER]/g; s/🔄/[REFRESH]/g; s/📈/[TRENDING_UP]/g; s/🏗️/[CONSTRUCTION]/g; s/🎨/[ART]/g; s/🧪/[TEST_TUBE]/g; s/📱/[MOBILE]/g; s/🖥️/[DESKTOP]/g; s/⭐/[STAR]/g; s/🔥/[FIRE]/g; s/💰/[MONEY]/g; s/🏦/[BANK]/g; s/💳/[CREDIT_CARD]/g; s/📄/[PAGE]/g; s/🗂️/[FOLDER]/g; s/🔐/[SECURE]/g; s/🎉/[PARTY]/g; s/👤/[USER]/g; s/👥/[USERS]/g; s/🌍/[EARTH]/g; s/📞/[PHONE]/g; s/📧/[EMAIL]/g; s/🔔/[BELL]/g; s/⏰/[ALARM]/g; s/📅/[CALENDAR]/g; s/🕒/[CLOCK]/g; s/🎪/[CIRCUS]/g; s/🎭/[MASKS]/g; s/🎬/[MOVIE]/g; s/🎮/[GAME]/g; s/🎲/[DICE]/g; s/🎯/[DART]/g; s/🎪/[TENT]/g; s/🎨/[PALETTE]/g; s/🎵/[MUSIC]/g; s/🎶/[NOTES]/g; s/🎤/[MIC]/g; s/🎧/[HEADPHONES]/g; s/🎸/[GUITAR]/g; s/🎹/[PIANO]/g; s/🎺/[TRUMPET]/g; s/🎻/[VIOLIN]/g; s/🥁/[DRUM]/g; s/🎪/[CIRCUS_TENT]/g' "$input_file" > "$output_file"
}

# Process and combine markdown files
combine_docs() {
    log_info "Combining documentation files..."
    
    # Generate table of contents
    generate_toc
    
    # Add main README (overview)
    log_info "Processing README.md..."
    echo -e "\n\\newpage\n" >> "$TEMP_DIR/00-toc.md"
    echo "# Bank REST API Overview" >> "$TEMP_DIR/00-toc.md"
    echo "" >> "$TEMP_DIR/00-toc.md"
    
    # Process README.md and remove the first title (we'll add our own)
    if [[ "$USE_HTML" != "true" ]]; then
        # Clean emojis for PDF generation - remove all emojis using python
        tail -n +2 README.md | python3 -c "import sys, re; print(re.sub(r'[^\x00-\x7F]+', '', sys.stdin.read()))" >> "$TEMP_DIR/00-toc.md"
    else
        # Keep emojis for HTML generation
        tail -n +2 README.md >> "$TEMP_DIR/00-toc.md"
    fi
    
    # Add Admin Frontend Documentation
    if [[ -f "admin-frontend/README.md" ]]; then
        log_info "Processing admin-frontend/README.md..."
        echo -e "\n\\newpage\n" >> "$TEMP_DIR/00-toc.md"
        echo "# Admin Dashboard Frontend" >> "$TEMP_DIR/00-toc.md"
        echo "" >> "$TEMP_DIR/00-toc.md"
        
        # Process admin frontend README and remove the first title
        if [[ "$USE_HTML" != "true" ]]; then
            # Clean emojis for PDF generation
            tail -n +2 admin-frontend/README.md | python3 -c "import sys, re; print(re.sub(r'[^\x00-\x7F]+', '', sys.stdin.read()))" >> "$TEMP_DIR/00-toc.md"
        else
            # Keep emojis for HTML generation
            tail -n +2 admin-frontend/README.md >> "$TEMP_DIR/00-toc.md"
        fi
    fi
    
    # Process each documentation file
    local counter=1
    for doc_file in "$DOCS_DIR"/*.md; do
        if [[ -f "$doc_file" ]]; then
            filename=$(basename "$doc_file" .md)
            log_info "Processing $filename.md..."
            
            # Add page break and process file
            echo -e "\n\\newpage\n" >> "$TEMP_DIR/$(printf "%02d" $counter)-$filename.md"
            
            if [[ "$USE_HTML" != "true" ]]; then
                # Clean emojis for PDF generation - remove all emojis using python
                python3 -c "import sys, re; print(re.sub(r'[^\x00-\x7F]+', '', open('$doc_file').read()))" >> "$TEMP_DIR/$(printf "%02d" $counter)-$filename.md"
            else
                # Keep emojis for HTML generation
                cat "$doc_file" >> "$TEMP_DIR/$(printf "%02d" $counter)-$filename.md"
            fi
            
            ((counter++))
        fi
    done
    
    # Combine all files
    log_info "Merging all documentation files..."
    cat "$TEMP_DIR"/*.md > "$TEMP_DIR/combined-docs.md"
}

# Generate PDF using pandoc
generate_pdf() {
    local input_file="$TEMP_DIR/combined-docs.md"
    local output_path="$OUTPUT_DIR/$OUTPUT_FILE"
    
    if [[ "$USE_HTML" == "true" ]]; then
        # Generate HTML instead of PDF
        local html_file="${OUTPUT_FILE%.pdf}.html"
        output_path="$OUTPUT_DIR/$html_file"
        
        log_info "Generating HTML documentation: $html_file"
        pandoc "$input_file" \
            --from markdown \
            --to html5 \
            --standalone \
            --toc \
            --toc-depth=3 \
            --css="https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.2.0/github-markdown-light.min.css" \
            --metadata title="Bank REST API Documentation" \
            --metadata author="Bank API Team" \
            --metadata date="$(date '+%Y-%m-%d')" \
            --output "$output_path"
    else
        # Generate PDF
        log_info "Generating PDF documentation: $OUTPUT_FILE"
        pandoc "$input_file" \
            --from markdown \
            --to pdf \
            --pdf-engine=pdflatex \
            --toc \
            --toc-depth=3 \
            --number-sections \
            --highlight-style=pygments \
            --variable="margin-left=1in" \
            --variable="margin-right=1in" \
            --variable="margin-top=1in" \
            --variable="margin-bottom=1in" \
            --variable="fontsize=11pt" \
            --variable="documentclass=article" \
            --variable="papersize=letter" \
            --metadata title="Bank REST API Documentation" \
            --metadata author="Bank API Team" \
            --metadata date="$(date '+%Y-%m-%d')" \
            --output "$output_path"
    fi
    
    if [[ $? -eq 0 ]]; then
        log_success "Documentation generated successfully: $output_path"
        
        # Display file info
        if [[ -f "$output_path" ]]; then
            local file_size=$(du -h "$output_path" | cut -f1)
            log_info "File size: $file_size"
            log_info "Location: $(realpath "$output_path")"
        fi
    else
        log_error "Failed to generate documentation"
        exit 1
    fi
}

# Main function
main() {
    log_info "Starting Bank REST API documentation PDF generation..."
    log_info "Timestamp: $(date)"
    
    # Check if we're in the right directory
    if [[ ! -f "README.md" ]] || [[ ! -d "$DOCS_DIR" ]]; then
        log_error "Please run this script from the project root directory"
        log_error "Expected files: README.md, docs/ directory"
        exit 1
    fi
    
    check_dependencies
    setup_temp_dir
    combine_docs
    generate_pdf
    
    log_success "Documentation generation completed!"
    
    # Open the generated file (optional)
    if [[ "$1" == "--open" ]]; then
        local output_path="$OUTPUT_DIR/$OUTPUT_FILE"
        if [[ "$USE_HTML" == "true" ]]; then
            output_path="$OUTPUT_DIR/${OUTPUT_FILE%.pdf}.html"
        fi
        
        if [[ "$OSTYPE" == "darwin"* ]]; then
            open "$output_path"
        elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
            xdg-open "$output_path"
        fi
    fi
}

# Show help
show_help() {
    echo "Bank REST API Documentation PDF Generator"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --open    Open the generated file after creation"
    echo "  --help    Show this help message"
    echo ""
    echo "Dependencies:"
    echo "  - pandoc (required)"
    echo "  - pdflatex (optional, for PDF generation)"
    echo ""
    echo "If pdflatex is not available, HTML output will be generated instead."
}

# Parse command line arguments
case "${1:-}" in
    --help|-h)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac