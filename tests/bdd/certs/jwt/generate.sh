#!/bin/bash
# Generate RSA 2048 key pair for JWT BDD tests.
# Output: jwt-private.pem (test code signs tokens), jwt-public.pem (mounted in Docker)
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

# Generate RSA 2048 private key
openssl genrsa -out "$DIR/jwt-private.pem" 2048 2>/dev/null

# Extract public key
openssl rsa -in "$DIR/jwt-private.pem" -pubout -out "$DIR/jwt-public.pem" 2>/dev/null

echo "Generated: jwt-private.pem, jwt-public.pem"
