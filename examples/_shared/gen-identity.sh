#!/usr/bin/env bash
# gen-identity.sh <name> <outdir> — generate one operator identity.
#
# Produces two files under <outdir>:
#   <name>.key   Ed25519 private key, OpenSSH format (what sbctl --key wants)
#   <name>.pem   SPKI "PUBLIC KEY" PEM (what authorized_operator_keys wants)
#
# Two formats are required because of a format split in the alpha: the
# daemon parses operator public keys as PKIX/SPKI PEM (E-CFG-009), while
# sbctl loads the private key via golang.org/x/crypto/ssh and only accepts
# the OPENSSH PRIVATE KEY form (a PKCS#8 key fails with "E-CFG-010 ...
# not an Ed25519 private key (got ed25519.PrivateKey)" — pointer vs value
# type assertion in the key loader).
#
# The keypair is generated with ssh-keygen (portable across OpenSSH
# versions); the SPKI PEM is derived from the OpenSSH public key by
# prepending the fixed 12-byte DER prefix for an ed25519
# SubjectPublicKeyInfo to the raw 32-byte key. The result is verified
# with openssl before it is trusted.
set -euo pipefail

name="$1"
outdir="$2"
mkdir -p "${outdir}"

if [[ -f "${outdir}/${name}.key" && -f "${outdir}/${name}.pem" ]]; then
  echo "gen-identity: ${name} already exists in ${outdir}, keeping it"
  exit 0
fi

ssh-keygen -t ed25519 -N '' -q -f "${outdir}/${name}.key"

# OpenSSH pubkey blob: string "ssh-ed25519" + length-prefixed 32-byte key;
# the raw key is the final 32 bytes of the base64-decoded blob.
raw_pub="$(awk '{print $2}' "${outdir}/${name}.key.pub" | base64 -d | tail -c 32 | base64)"
spki_b64="$(
  {
    printf '\x30\x2a\x30\x05\x06\x03\x2b\x65\x70\x03\x21\x00'
    printf '%s' "${raw_pub}" | base64 -d
  } | base64
)"
{
  echo "-----BEGIN PUBLIC KEY-----"
  echo "${spki_b64}"
  echo "-----END PUBLIC KEY-----"
} > "${outdir}/${name}.pem"

openssl pkey -pubin -in "${outdir}/${name}.pem" -noout
chmod 0600 "${outdir}/${name}.key"

echo "gen-identity: generated ${name} (OpenSSH private + SPKI PEM public)"
