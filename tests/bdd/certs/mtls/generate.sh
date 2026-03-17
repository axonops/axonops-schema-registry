#!/bin/bash
# Generate ECDSA P256 certificates for mTLS BDD testing.
#
# Output:
#   ca.pem / ca-key.pem                   — Test CA
#   server.pem / server-key.pem           — Server cert (SANs: schema-registry, localhost, 127.0.0.1)
#   client-admin.pem / client-admin-key.pem       — Client cert CN=admin-mtls
#   client-readonly.pem / client-readonly-key.pem  — Client cert CN=readonly-mtls
#   client-expired.pem / client-expired-key.pem    — Expired client cert CN=expired-mtls
#   client-wrong-ca.pem / client-wrong-ca-key.pem  — Client cert signed by a different CA
#   wrong-ca.pem / wrong-ca-key.pem                — The "wrong" CA (not trusted by server)
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

# ---------- CA ----------
openssl ecparam -genkey -name prime256v1 -out "$DIR/ca-key.pem" 2>/dev/null
openssl req -new -x509 -key "$DIR/ca-key.pem" -sha256 \
  -subj "/CN=mTLS Test CA" \
  -days 3650 \
  -out "$DIR/ca.pem" 2>/dev/null

# ---------- Server cert ----------
openssl ecparam -genkey -name prime256v1 -out "$DIR/server-key.pem" 2>/dev/null
openssl req -new -key "$DIR/server-key.pem" \
  -subj "/CN=schema-registry" \
  -out "$DIR/server.csr" 2>/dev/null

cat > "$DIR/server-ext.cnf" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:schema-registry,DNS:localhost,IP:127.0.0.1
EOF

openssl x509 -req -in "$DIR/server.csr" \
  -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial \
  -extfile "$DIR/server-ext.cnf" \
  -days 3650 -sha256 \
  -out "$DIR/server.pem" 2>/dev/null

# ---------- Helper: generate client cert ----------
generate_client() {
  local name="$1"  # e.g. "client-admin"
  local cn="$2"    # e.g. "admin-mtls"
  local days="$3"  # e.g. 3650
  local ca_cert="$4"
  local ca_key="$5"

  openssl ecparam -genkey -name prime256v1 -out "$DIR/${name}-key.pem" 2>/dev/null
  openssl req -new -key "$DIR/${name}-key.pem" \
    -subj "/CN=${cn}" \
    -out "$DIR/${name}.csr" 2>/dev/null

  cat > "$DIR/${name}-ext.cnf" <<EXTEOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=clientAuth
EXTEOF

  openssl x509 -req -in "$DIR/${name}.csr" \
    -CA "$ca_cert" -CAkey "$ca_key" -CAcreateserial \
    -extfile "$DIR/${name}-ext.cnf" \
    -days "$days" -sha256 \
    -out "$DIR/${name}.pem" 2>/dev/null

  rm -f "$DIR/${name}.csr" "$DIR/${name}-ext.cnf"
}

# ---------- Client certs signed by our CA ----------
generate_client "client-admin"    "admin-mtls"    3650 "$DIR/ca.pem" "$DIR/ca-key.pem"
generate_client "client-readonly" "readonly-mtls" 3650 "$DIR/ca.pem" "$DIR/ca-key.pem"

# ---------- Expired client cert (signed by our CA, valid for 1 day in the past) ----------
# Use -days 1 and backdate the start to make it already expired.
openssl ecparam -genkey -name prime256v1 -out "$DIR/client-expired-key.pem" 2>/dev/null
openssl req -new -key "$DIR/client-expired-key.pem" \
  -subj "/CN=expired-mtls" \
  -out "$DIR/client-expired.csr" 2>/dev/null

cat > "$DIR/client-expired-ext.cnf" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=clientAuth
EOF

# Create cert that expired yesterday: valid from 2 days ago for 1 day.
openssl ca -batch -notext \
  -cert "$DIR/ca.pem" -keyfile "$DIR/ca-key.pem" \
  -startdate "$(date -u -d '-2 days' +%Y%m%d000000Z 2>/dev/null || date -u -v-2d +%Y%m%d000000Z)" \
  -enddate "$(date -u -d '-1 day' +%Y%m%d000000Z 2>/dev/null || date -u -v-1d +%Y%m%d000000Z)" \
  -extfile "$DIR/client-expired-ext.cnf" \
  -in "$DIR/client-expired.csr" \
  -out "$DIR/client-expired.pem" 2>/dev/null || {
  # Fallback: use openssl x509 with -days 0 (creates a cert valid for 0 days = already expired)
  # Some systems need faketime or specific openssl version for precise backdating.
  # Use a simpler approach: create cert with not-after in the past via a tiny validity window.
  openssl x509 -req -in "$DIR/client-expired.csr" \
    -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial \
    -extfile "$DIR/client-expired-ext.cnf" \
    -days 0 -sha256 \
    -out "$DIR/client-expired.pem" 2>/dev/null
}
rm -f "$DIR/client-expired.csr" "$DIR/client-expired-ext.cnf"

# ---------- Wrong-CA client cert ----------
# Generate a separate CA that the server will NOT trust.
openssl ecparam -genkey -name prime256v1 -out "$DIR/wrong-ca-key.pem" 2>/dev/null
openssl req -new -x509 -key "$DIR/wrong-ca-key.pem" -sha256 \
  -subj "/CN=Wrong Test CA" \
  -days 3650 \
  -out "$DIR/wrong-ca.pem" 2>/dev/null

generate_client "client-wrong-ca" "wrong-ca-mtls" 3650 "$DIR/wrong-ca.pem" "$DIR/wrong-ca-key.pem"

# ---------- Cleanup ----------
rm -f "$DIR/server.csr" "$DIR/server-ext.cnf" "$DIR"/*.srl

echo "Generated mTLS test certificates in $DIR"
echo "  CA:              ca.pem, ca-key.pem"
echo "  Server:          server.pem, server-key.pem"
echo "  Client (admin):  client-admin.pem, client-admin-key.pem"
echo "  Client (readonly): client-readonly.pem, client-readonly-key.pem"
echo "  Client (expired):  client-expired.pem, client-expired-key.pem"
echo "  Client (wrong CA): client-wrong-ca.pem, client-wrong-ca-key.pem"
echo "  Wrong CA:        wrong-ca.pem, wrong-ca-key.pem"
