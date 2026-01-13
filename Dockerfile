# 多阶段构建：前端（Vite）+ 后端（Go）

FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build -- --outDir=dist

FROM golang:1.22-alpine AS backend-builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/liao ./cmd/liao
COPY --from=frontend-builder /app/frontend/dist /out/static

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && mkdir -p /app/upload
WORKDIR /app
COPY --from=backend-builder /out/liao /out/static /app/
EXPOSE 8080
ENV SERVER_PORT=8080
ENTRYPOINT ["/app/liao"]
