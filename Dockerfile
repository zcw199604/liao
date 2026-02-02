# 多阶段构建：前端（Vite）+ 后端（Go）

FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build -- --outDir=dist

FROM golang:1.25.6-alpine AS backend-builder
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
ENV TZ=Asia/Shanghai
RUN apk add --no-cache ca-certificates tzdata ffmpeg exiftool \
  && ln -snf "/usr/share/zoneinfo/${TZ}" /etc/localtime \
  && echo "${TZ}" > /etc/timezone \
  && mkdir -p /app/upload
WORKDIR /app
COPY --from=backend-builder /out/liao /app/
COPY --from=backend-builder /out/static /app/static/
COPY sql/ /app/sql/
EXPOSE 8080
ENV SERVER_PORT=8080
ENTRYPOINT ["/app/liao"]
