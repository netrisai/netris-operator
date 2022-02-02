# syntax=docker/dockerfile:experimental
# Build the manager binary
FROM golang:1.17 as builder

WORKDIR /workspace

# Config ssh private key
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
RUN git config --global --add url."git@github.com:".insteadOf "https://github.com/"

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=ssh,id=ssh_private_key_ci go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY configloader/ configloader/
COPY lbwatcher/ lbwatcher/
COPY calicowatcher/ calicowatcher/
COPY netrisstorage/ netrisstorage/

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
