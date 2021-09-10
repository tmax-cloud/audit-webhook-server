FROM golang:1.15-alpine as builder

WORKDIR /go/src

# Download pacakages
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy source files and Build
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use light-weight base image
FROM gcr.io/distroless/static:nonroot
WORKDIR /go/src
COPY --from=builder /go/src .

RUN chmod 777 main
RUN chmod 777 start.sh
ENTRYPOINT ["/go/src/start.sh"]