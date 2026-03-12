#!/bin/bash
# Generate self-signed CA + server certificate for BDD TLS syslog testing.
# Output: ca.pem, server.pem, server-key.pem
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

# Generate CA private key
openssl ecparam -genkey -name prime256v1 -out "$DIR/ca-key.pem" 2>/dev/null

# Generate CA certificate
openssl req -new -x509 -key "$DIR/ca-key.pem" -sha256 \
  -subj "/CN=BDD Test CA" \
  -days 3650 \
  -out "$DIR/ca.pem" 2>/dev/null

# Generate server private key
openssl ecparam -genkey -name prime256v1 -out "$DIR/server-key.pem" 2>/dev/null

# Generate server CSR
openssl req -new -key "$DIR/server-key.pem" \
  -subj "/CN=syslog-ng" \
  -out "$DIR/server.csr" 2>/dev/null

# Create extensions file for SANs
cat > "$DIR/ext.cnf" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:syslog-ng,DNS:localhost,IP:127.0.0.1
EOF

# Sign server cert with CA
openssl x509 -req -in "$DIR/server.csr" \
  -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial \
  -extfile "$DIR/ext.cnf" \
  -days 3650 \
  -sha256 \
  -out "$DIR/server.pem" 2>/dev/null

# Cleanup temp files
rm -f "$DIR/server.csr" "$DIR/ext.cnf" "$DIR/ca.srl"

echo "Generated: ca.pem, ca-key.pem, server.pem, server-key.pem"
