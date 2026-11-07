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

# CLI 与隧道库固定同一版本；云函数代码包始终使用 linux/amd64。
FROM golang:1.25.6-alpine AS scfproxy-builder
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
RUN apk add --no-cache zip
RUN go mod download github.com/zcw199604/scfproxy@v0.1.0 \
  && mkdir -p /out \
  && cd /go/pkg/mod/github.com/zcw199604/scfproxy@v0.1.0 \
  && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scfproxy ./cmd/scfproxy \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/scf-server ./cmd/scf-server \
  && cp deploy/scf_bootstrap /out/scf_bootstrap \
  && chmod 755 /out/scf_bootstrap \
  && cd /out \
  && zip -q scf-server.zip scf-server scf_bootstrap

FROM alpine:3.19
ENV TZ=Asia/Shanghai
RUN apk add --no-cache ca-certificates tzdata ffmpeg exiftool \
  && ln -snf "/usr/share/zoneinfo/${TZ}" /etc/localtime \
  && echo "${TZ}" > /etc/timezone \
  && mkdir -p /app/upload
WORKDIR /app
COPY --from=backend-builder /out/liao /app/
COPY --from=scfproxy-builder /out/scfproxy /usr/local/bin/scfproxy
COPY --from=scfproxy-builder /out/scf-server.zip /app/scf-server.zip
COPY --from=backend-builder /out/static /app/static/
COPY sql/ /app/sql/
EXPOSE 8080
ENV SERVER_PORT=8080
ENTRYPOINT ["/app/liao"]
