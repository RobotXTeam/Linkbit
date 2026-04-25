FROM node:20-bookworm AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM golang:1.26-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/linkbit-controller ./cmd/linkbit-controller && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/linkbit-relay ./cmd/linkbit-relay && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/linkbit-agent ./cmd/linkbit-agent

FROM gcr.io/distroless/static-debian12 AS controller
WORKDIR /opt/linkbit
COPY --from=go-build /out/linkbit-controller /opt/linkbit/linkbit-controller
COPY --from=web /src/web/dist /opt/linkbit/web
ENV LINKBIT_LISTEN_ADDR=:8080
ENV LINKBIT_WEB_DIR=/opt/linkbit/web
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/opt/linkbit/linkbit-controller"]

FROM gcr.io/distroless/static-debian12 AS relay
WORKDIR /opt/linkbit
COPY --from=go-build /out/linkbit-relay /opt/linkbit/linkbit-relay
ENV LINKBIT_LISTEN_ADDR=:8443
EXPOSE 8443
USER 65532:65532
ENTRYPOINT ["/opt/linkbit/linkbit-relay"]

