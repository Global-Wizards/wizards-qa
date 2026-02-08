# Stage 1: Build frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY web/frontend/package.json web/frontend/package-lock.json* ./
RUN npm ci --ignore-scripts 2>/dev/null || npm install
COPY web/frontend/ ./
RUN npm run build

# Stage 2a: Build backend
FROM golang:1.25-alpine AS backend-build
WORKDIR /app/backend
COPY web/backend/go.mod web/backend/go.sum ./
RUN go mod download
COPY web/backend/ ./
COPY VERSION /tmp/VERSION
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=$(cat /tmp/VERSION | tr -d '\n')" -trimpath -o /dashboard-server .

# Stage 2b: Build CLI
FROM golang:1.25-alpine AS cli-build
WORKDIR /app/cli
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(cat VERSION | tr -d '\n')" -trimpath -o /wizards-qa ./cmd

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates chromium nss freetype harfbuzz \
    openjdk17-jre-headless curl unzip \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

# Install Maestro CLI (pinned version) to /opt/maestro
ENV MAESTRO_DIR=/opt/maestro
ENV MAESTRO_VERSION=1.39.13
RUN curl -fsSL "https://get.maestro.mobile.dev" | bash \
    && chmod -R a+rX /opt/maestro
ENV PATH="/opt/maestro/bin:$PATH"
ENV CHROME_BIN=/usr/bin/chromium-browser
WORKDIR /app
COPY --from=backend-build /dashboard-server ./dashboard-server
COPY --from=cli-build /wizards-qa ./wizards-qa
COPY --from=frontend-build /app/frontend/dist ./web/frontend/dist/
COPY flows/ ./flows/
COPY data/ ./data/
COPY CHANGELOG.md ./CHANGELOG.md
RUN mkdir -p ./reports && chown -R appuser:appgroup /app
ENV WIZARDS_QA_CLI_PATH=/app/wizards-qa
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health || exit 1
CMD ["./dashboard-server"]
