# build stage
FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/job ./cmd/job

# runtime: server
FROM gcr.io/distroless/static:nonroot AS server
COPY --from=build /out/server /server
USER 65532:65532
ENTRYPOINT ["/server"]

# runtime: job
FROM gcr.io/distroless/static:nonroot AS job
COPY --from=build /out/job /job
USER 65532:65532
ENTRYPOINT ["/job"]