FROM --platform=$BUILDPLATFORM node:20-bookworm AS frontend-builder

COPY web/package*.json /web/
WORKDIR /web
RUN npm ci

COPY web/ /web/
RUN npm run build && \
     echo "Frontend build completed:" && \
     ls -lah dist/

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS builder

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG MWX_ISP_VERSION=develop
ARG MWX_ISP_COMMIT=unknown
ARG MWX_ISP_BUILD_TIME=unknown

COPY . /src
WORKDIR /src

# Copy built frontend from frontend-builder stage
COPY --from=frontend-builder /web/dist /src/web/dist

# Verify frontend is present (built to dist/admin/)
RUN test -f /src/web/dist/admin/index.html || (echo "ERROR: Frontend not found!" && exit 1)

# Build for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -ldflags \
     "-s -w -X main.Version=${MWX_ISP_VERSION} -X main.BuildTime=${MWX_ISP_BUILD_TIME} -X main.GitCommit=${MWX_ISP_COMMIT} -extldflags \"-static\"" -o /mwx-isp main.go

FROM alpine:3.24

RUN apk add --no-cache curl ca-certificates tzdata && \
    addgroup -S -g 10001 mwx && \
    adduser -S -D -H -u 10001 -G mwx mwx && \
    mkdir -p /var/toughradius && \
    chown 10001:10001 /var/toughradius
ARG MWX_ISP_VERSION=develop
ARG MWX_ISP_COMMIT=unknown
ARG MWX_ISP_BUILD_TIME=unknown
ENV TZ=Asia/Jakarta
LABEL org.opencontainers.image.title="MWX-ISP" \
      org.opencontainers.image.description="ISP Management, RADIUS, and Billing" \
      org.opencontainers.image.source="https://github.com/bjo163/mwx-isp" \
      org.opencontainers.image.version="${MWX_ISP_VERSION}" \
      org.opencontainers.image.revision="${MWX_ISP_COMMIT}" \
      org.opencontainers.image.created="${MWX_ISP_BUILD_TIME}"

COPY --from=builder /mwx-isp /usr/local/bin/mwx-isp

RUN chmod 0755 /usr/local/bin/mwx-isp

# Expose required ports:
# 1816 - Web/Admin API (HTTP)
# 1812 - RADIUS Authentication (UDP)
# 1813 - RADIUS Accounting (UDP)
# 2083 - RadSec (RADIUS over TLS)
EXPOSE 1816/tcp 1812/udp 1813/udp 2083/tcp

USER 10001:10001
ENTRYPOINT ["/usr/local/bin/mwx-isp"]
