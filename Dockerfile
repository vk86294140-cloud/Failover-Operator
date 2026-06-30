# Build the manager binary.
FROM golang:1.22 AS build
WORKDIR /workspace
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -o manager ./cmd

# Run on a minimal, non-root distroless base.
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
