FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
ARG VERSION=
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /nudge .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /nudge /usr/local/bin/nudge
# The agent socket's folder is made on start; the root file system may be read-only.
ENV NUDGE_SOCKET=/tmp/nudge/agent.sock
ENTRYPOINT ["/usr/local/bin/nudge"]
CMD ["serve"]
