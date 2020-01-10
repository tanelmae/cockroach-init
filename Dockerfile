FROM golang:1.13.5-alpine as builder
RUN apk add --update upx ca-certificates

ARG VERSION=local
ENV CGO_ENABLED 0
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -mod=readonly -ldflags "-s -w -X main.Version=${VERSION}" -o crdb-runner cmd/main.go
RUN upx crdb-runner

FROM scratch
COPY --from=builder /workspace/crdb-runner /bin/crdb-runner
ENTRYPOINT ["/bin/crdb-runner"]
