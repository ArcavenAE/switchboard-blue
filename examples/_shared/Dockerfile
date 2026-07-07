# examples/_shared/Dockerfile — runtime image for the examples ladder.
#
# Installs the published alpha binaries from GitHub Releases instead of
# building from source: the point of the examples is to prove the SHIPPED
# artifacts, not the working tree. Pin the release with --build-arg
# SWITCHBOARD_RELEASE=<tag>; the default tracks the tag these examples
# were last verified against.
#
# The binaries are installed as `switchboard` and `sbctl` (dropping the
# `-a` alpha suffix) so the example commands match docs/getting-started.md.

FROM debian:bookworm-slim

ARG SWITCHBOARD_RELEASE=alpha-20260706-203527-62e38d3
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl jq tmux procps htop openssl openssh-client \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    base="https://github.com/ArcavenAE/switchboard-blue/releases/download/${SWITCHBOARD_RELEASE}"; \
    curl -fsSL -o /usr/local/bin/switchboard "${base}/switchboard-a-linux-${TARGETARCH}"; \
    curl -fsSL -o /usr/local/bin/sbctl "${base}/sbctl-a-linux-${TARGETARCH}"; \
    chmod 0755 /usr/local/bin/switchboard /usr/local/bin/sbctl; \
    /usr/local/bin/switchboard --version; \
    /usr/local/bin/sbctl --version

COPY gen-identity.sh harness.sh /usr/local/lib/switchboard-examples/
RUN chmod 0755 /usr/local/lib/switchboard-examples/gen-identity.sh
