################
# Build binary #
################
FROM --platform=${BUILDPLATFORM:-linux/arm64} golang:1.20 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

WORKDIR /build

COPY . .

RUN go mod download
RUN go vet -v
RUN go test -v
RUN go build -ldflags="-w -s" -o app .

#####################
# Build final image #
#####################
FROM --platform=${TARGETPLATFORM:-linux/arm64} gcr.io/distroless/static-debian11:nonroot

COPY --from=builder /build/app /
ADD --chmod=755 https://github.com/mvdkleijn/healthchecker/releases/download/v1.0.2/healthchecker-${TARGETOS}-${TARGETARCH} /healthchecker

EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=5s --retries=3 \
    CMD ["/healthchecker", "http://127.0.0.1:8080/api/v1/status"]

ENTRYPOINT ["/app"]