FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /manager ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /manager /manager
ENTRYPOINT ["/manager"]
