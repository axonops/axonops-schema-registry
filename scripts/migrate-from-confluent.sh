#!/bin/bash
#
# Migrate schemas from Confluent Schema Registry to AxonOps Schema Registry
#
# Usage:
#   ./migrate-from-confluent.sh [options]
#
# Options:
#   --source URL        Confluent Schema Registry URL (default: http://localhost:8081)
#   --target URL        AxonOps Schema Registry URL (default: http://localhost:8082)
#   --source-user USER  Basic auth user for source (optional)
#   --source-pass PASS  Basic auth password for source (optional)
#   --target-user USER  Basic auth user for target (optional)
#   --target-pass PASS  Basic auth password for target (optional)
#   --target-apikey KEY API key for target (optional)
#   --dry-run           Export only, don't import
#   --verify            Verify after import
#   --output FILE       Save exported schemas to file (default: schemas-export.json)
#   --help              Show this help message
#

set -euo pipefail

# Default values
SOURCE_URL="http://localhost:8081"
TARGET_URL="http://localhost:8082"
SOURCE_USER=""
SOURCE_PASS=""
TARGET_USER=""
TARGET_PASS=""
TARGET_APIKEY=""
DRY_RUN=false
VERIFY=false
OUTPUT_FILE="schemas-export.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    head -25 "$0" | tail -20
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --source)
            SOURCE_URL="$2"
            shift 2
            ;;
        --target)
            TARGET_URL="$2"
            shift 2
            ;;
        --source-user)
            SOURCE_USER="$2"
            shift 2
            ;;
        --source-pass)
            SOURCE_PASS="$2"
            shift 2
            ;;
        --target-user)
            TARGET_USER="$2"
            shift 2
            ;;
        --target-pass)
            TARGET_PASS="$2"
            shift 2
            ;;
        --target-apikey)
            TARGET_APIKEY="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verify)
            VERIFY=true
            shift
            ;;
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --help)
            show_help
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Build curl auth options as arrays for proper quoting
SOURCE_AUTH=()
if [[ -n "$SOURCE_USER" && -n "$SOURCE_PASS" ]]; then
    SOURCE_AUTH=(-u "${SOURCE_USER}:${SOURCE_PASS}")
fi

TARGET_AUTH=()
if [[ -n "$TARGET_APIKEY" ]]; then
    TARGET_AUTH=(-H "Authorization: Bearer ${TARGET_APIKEY}")
elif [[ -n "$TARGET_USER" && -n "$TARGET_PASS" ]]; then
    TARGET_AUTH=(-u "${TARGET_USER}:${TARGET_PASS}")
fi

# Check dependencies
check_dependencies() {
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Install with: apt-get install jq"
        exit 1
    fi
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed."
        exit 1
    fi
}

# Test connectivity
test_connectivity() {
    log_info "Testing connectivity to source: $SOURCE_URL"
    if ! curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/" > /dev/null 2>&1; then
        log_error "Cannot connect to source Schema Registry at $SOURCE_URL"
        exit 1
    fi

    if [[ "$DRY_RUN" == "false" ]]; then
        log_info "Testing connectivity to target: $TARGET_URL"
        if ! curl -sf "${TARGET_AUTH[@]}" "$TARGET_URL/" > /dev/null 2>&1; then
            log_error "Cannot connect to target Schema Registry at $TARGET_URL"
            exit 1
        fi
    fi
}

# Export schemas from Confluent
export_schemas() {
    log_info "Exporting schemas from Confluent Schema Registry..."

    # Get all subjects
    local subjects
    subjects=$(curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/subjects" | jq -r '.[]')

    if [[ -z "$subjects" ]]; then
        log_warn "No subjects found in source registry"
        echo '{"schemas":[]}' > "$OUTPUT_FILE"
        return
    fi

    local total_schemas=0
    local schemas_json="[]"

    # For each subject, get all versions
    while IFS= read -r subject; do
        [[ -z "$subject" ]] && continue
        log_info "  Exporting subject: $subject"

        # Get all versions for this subject
        local versions
        versions=$(curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/subjects/$subject/versions" | jq -r '.[]')

        while IFS= read -r version; do
            [[ -z "$version" ]] && continue

            # Get schema details for this version
            local schema_info
            schema_info=$(curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/subjects/$subject/versions/$version")

            local schema_id
            schema_id=$(echo "$schema_info" | jq -r '.id')

            local schema_type
            schema_type=$(echo "$schema_info" | jq -r '.schemaType // "AVRO"')

            local schema_content
            schema_content=$(echo "$schema_info" | jq -r '.schema')

            local references
            references=$(echo "$schema_info" | jq -c '.references // []')

            # Build the import schema object
            local import_obj
            import_obj=$(jq -n \
                --argjson id "$schema_id" \
                --arg subject "$subject" \
                --argjson version "$version" \
                --arg schemaType "$schema_type" \
                --arg schema "$schema_content" \
                --argjson references "$references" \
                '{id: $id, subject: $subject, version: $version, schemaType: $schemaType, schema: $schema, references: $references}')

            schemas_json=$(echo "$schemas_json" | jq --argjson obj "$import_obj" '. += [$obj]')
            total_schemas=$((total_schemas + 1))

        done <<< "$versions"
    done <<< "$subjects"

    # Sort by ID to ensure dependencies are imported first
    schemas_json=$(echo "$schemas_json" | jq 'sort_by(.id)')

    # Write to output file
    echo "{\"schemas\": $schemas_json}" | jq '.' > "$OUTPUT_FILE"

    log_info "Exported $total_schemas schemas to $OUTPUT_FILE"
}

# Import schemas to AxonOps
import_schemas() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Dry run mode - skipping import"
        return
    fi

    log_info "Importing schemas to AxonOps Schema Registry..."

    local import_data
    import_data=$(cat "$OUTPUT_FILE")

    local schema_count
    schema_count=$(echo "$import_data" | jq '.schemas | length')

    if [[ "$schema_count" == "0" ]]; then
        log_warn "No schemas to import"
        return
    fi

    log_info "Importing $schema_count schemas..."

    # Import all schemas in one request
    local response
    response=$(curl -sf "${TARGET_AUTH[@]}" \
        -X POST "$TARGET_URL/import/schemas" \
        -H "Content-Type: application/json" \
        -d "$import_data")

    local imported
    imported=$(echo "$response" | jq -r '.imported')

    local errors
    errors=$(echo "$response" | jq -r '.errors')

    log_info "Import complete: $imported successful, $errors errors"

    if [[ "$errors" != "0" ]]; then
        log_warn "Some schemas failed to import:"
        echo "$response" | jq -r '.results[] | select(.success == false) | "  ID \(.id) (\(.subject) v\(.version)): \(.error)"'
    fi
}

# Verify migration
verify_migration() {
    if [[ "$VERIFY" != "true" ]]; then
        return
    fi

    log_info "Verifying migration..."

    local source_subjects
    source_subjects=$(curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/subjects" | jq -r '.[]' | sort)

    local target_subjects
    target_subjects=$(curl -sf "${TARGET_AUTH[@]}" "$TARGET_URL/subjects" | jq -r '.[]' | sort)

    # Compare subjects
    local missing_subjects
    missing_subjects=$(comm -23 <(echo "$source_subjects") <(echo "$target_subjects"))

    if [[ -n "$missing_subjects" ]]; then
        log_error "Missing subjects in target:"
        echo "$missing_subjects" | while read -r subj; do
            echo "  - $subj"
        done
    fi

    # Verify each schema
    local verification_errors=0

    while IFS= read -r subject; do
        [[ -z "$subject" ]] && continue

        local source_versions
        source_versions=$(curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/subjects/$subject/versions" | jq -r '.[]')

        while IFS= read -r version; do
            [[ -z "$version" ]] && continue

            local source_schema
            source_schema=$(curl -sf "${SOURCE_AUTH[@]}" "$SOURCE_URL/subjects/$subject/versions/$version")

            local source_id
            source_id=$(echo "$source_schema" | jq -r '.id')

            local source_content
            source_content=$(echo "$source_schema" | jq -r '.schema')

            # Get from target
            local target_schema
            target_schema=$(curl -sf "${TARGET_AUTH[@]}" "$TARGET_URL/subjects/$subject/versions/$version" 2>/dev/null || echo "null")

            if [[ "$target_schema" == "null" ]]; then
                log_error "Missing: $subject v$version (ID: $source_id)"
                verification_errors=$((verification_errors + 1))
                continue
            fi

            local target_id
            target_id=$(echo "$target_schema" | jq -r '.id')

            local target_content
            target_content=$(echo "$target_schema" | jq -r '.schema')

            # Verify ID matches
            if [[ "$source_id" != "$target_id" ]]; then
                log_error "ID mismatch for $subject v$version: source=$source_id, target=$target_id"
                verification_errors=$((verification_errors + 1))
            fi

            # Verify schema content (normalize JSON for comparison)
            local source_normalized
            source_normalized=$(echo "$source_content" | jq -cS '.' 2>/dev/null || echo "$source_content")

            local target_normalized
            target_normalized=$(echo "$target_content" | jq -cS '.' 2>/dev/null || echo "$target_content")

            if [[ "$source_normalized" != "$target_normalized" ]]; then
                log_error "Schema content mismatch for $subject v$version (ID: $source_id)"
                verification_errors=$((verification_errors + 1))
            fi

        done <<< "$source_versions"
    done <<< "$source_subjects"

    if [[ "$verification_errors" == "0" ]]; then
        log_info "Verification successful - all schemas match!"
    else
        log_error "Verification found $verification_errors errors"
        exit 1
    fi
}

# Print summary
print_summary() {
    echo ""
    echo "=========================================="
    echo "Migration Summary"
    echo "=========================================="
    echo "Source:      $SOURCE_URL"
    echo "Target:      $TARGET_URL"
    echo "Export file: $OUTPUT_FILE"
    echo ""

    if [[ -f "$OUTPUT_FILE" ]]; then
        local schema_count
        schema_count=$(jq '.schemas | length' "$OUTPUT_FILE")

        local subject_count
        subject_count=$(jq '[.schemas[].subject] | unique | length' "$OUTPUT_FILE")

        local max_id
        max_id=$(jq '[.schemas[].id] | max // 0' "$OUTPUT_FILE")

        echo "Schemas exported: $schema_count"
        echo "Subjects:         $subject_count"
        echo "Highest ID:       $max_id"
    fi
    echo "=========================================="
}

# Main
main() {
    echo "=========================================="
    echo "Confluent to AxonOps Schema Migration"
    echo "=========================================="
    echo ""

    check_dependencies
    test_connectivity
    export_schemas
    import_schemas
    verify_migration
    print_summary

    log_info "Migration complete!"
}

main
