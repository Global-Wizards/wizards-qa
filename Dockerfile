# Stage 1: Build frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY web/frontend/package.json ./
RUN npm install
COPY web/frontend/ ./
RUN npm run build

# Stage 2a: Build backend
FROM golang:1.22-alpine AS backend-build
WORKDIR /app/backend
COPY web/backend/go.mod web/backend/go.sum ./
RUN go mod download
COPY web/backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /dashboard-server .

# Stage 2b: Build CLI
FROM golang:1.22-alpine AS cli-build
WORKDIR /app/cli
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /wizards-qa .

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-build /dashboard-server ./dashboard-server
COPY --from=cli-build /wizards-qa ./wizards-qa
COPY --from=frontend-build /app/frontend/dist ./web/frontend/dist/
COPY flows/ ./flows/
COPY data/ ./data/
RUN mkdir -p ./reports
ENV WIZARDS_QA_CLI_PATH=/app/wizards-qa
EXPOSE 8080
CMD ["./dashboard-server"]
