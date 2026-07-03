# Minimal runtime image
# Binary is pre-built by GoReleaser and placed in build context
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget

# Copy pre-built binary
COPY golem /usr/local/bin/golem

# Create non-root user
RUN adduser -D -g '' appuser

# Switch to non-root user
USER appuser

EXPOSE 18790

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -qO- http://localhost:18790/health || exit 1

ENTRYPOINT ["/usr/local/bin/golem"]
CMD ["gateway"]
