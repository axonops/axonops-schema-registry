#!/bin/bash
# Generate self-signed CA + LDAP server certificate + client certificate
# for BDD LDAP TLS testing (LDAPS and mTLS).
#
# Output:
#   ca.pem, ca-key.pem          — Certificate Authority
#   server.pem, server-key.pem  — LDAP server cert (SAN: DNS:ldap, DNS:localhost, IP:127.0.0.1)
#   client.pem, client-key.pem  — Client cert for mTLS testing
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

# ---------- CA ----------
openssl ecparam -genkey -name prime256v1 -out "$DIR/ca-key.pem" 2>/dev/null

openssl req -new -x509 -key "$DIR/ca-key.pem" -sha256 \
  -subj "/CN=LDAP BDD Test CA" \
  -days 3650 \
  -out "$DIR/ca.pem" 2>/dev/null

# ---------- LDAP server cert ----------
openssl ecparam -genkey -name prime256v1 -out "$DIR/server-key.pem" 2>/dev/null

openssl req -new -key "$DIR/server-key.pem" \
  -subj "/CN=ldap" \
  -out "$DIR/server.csr" 2>/dev/null

cat > "$DIR/server-ext.cnf" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:ldap,DNS:localhost,IP:127.0.0.1
EOF

openssl x509 -req -in "$DIR/server.csr" \
  -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial \
  -extfile "$DIR/server-ext.cnf" \
  -days 3650 -sha256 \
  -out "$DIR/server.pem" 2>/dev/null

# ---------- Client cert (for mTLS) ----------
openssl ecparam -genkey -name prime256v1 -out "$DIR/client-key.pem" 2>/dev/null

openssl req -new -key "$DIR/client-key.pem" \
  -subj "/CN=schema-registry-client" \
  -out "$DIR/client.csr" 2>/dev/null

cat > "$DIR/client-ext.cnf" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=clientAuth
EOF

openssl x509 -req -in "$DIR/client.csr" \
  -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial \
  -extfile "$DIR/client-ext.cnf" \
  -days 3650 -sha256 \
  -out "$DIR/client.pem" 2>/dev/null

# ---------- Cleanup temp files ----------
rm -f "$DIR/server.csr" "$DIR/server-ext.cnf" \
      "$DIR/client.csr" "$DIR/client-ext.cnf" \
      "$DIR/ca.srl"

echo "Generated: ca.pem, ca-key.pem, server.pem, server-key.pem, client.pem, client-key.pem"
