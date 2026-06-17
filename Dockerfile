FROM node:latest AS frontend-build
WORKDIR /src/frontend
RUN npm install -g pnpm@10
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

FROM golang:latest AS backend-build
WORKDIR /src/backend
COPY backend/go.mod ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:latest
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
ENV PORT=8080
ENV CONFIG_DIR=/config
ENV WEB_DIR=/web
COPY --from=backend-build /server /server
COPY --from=frontend-build /src/frontend/build /web
USER app
EXPOSE 8080
ENTRYPOINT ["/server"]
