FROM golang:1.15 as builder

WORKDIR /go/src

# Download pacakages
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy source files and Build
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -o main .

# Use light-weight base image
FROM gcr.io/distroless/static:nonroot
WORKDIR /go/src
COPY --from=builder /go/src .

USER root
ENTRYPOINT ["/main"]