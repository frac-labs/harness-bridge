# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/bridge ./cmd/bridge

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bridge /usr/local/bin/bridge
USER 65532:65532
EXPOSE 9443
ENTRYPOINT ["/usr/local/bin/bridge"]
