################
# Build binary #
################
FROM --platform=${BUILDPLATFORM:-linux/arm64} golang:1.24 as builder

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

RUN \ 
  case ${TARGETPLATFORM} in \
    "linux/amd64") DOWNLOAD_ARCH="linux-amd64"  ;; \
    "linux/arm64") DOWNLOAD_ARCH="linux-arm64"  ;; \
  esac && \
  wget https://github.com/mvdkleijn/healthchecker/releases/download/v1.1.0/healthchecker-${DOWNLOAD_ARCH} && \
  mv /build/healthchecker-${DOWNLOAD_ARCH} /build/healthchecker && \
  chmod 755 /build/healthchecker && \
  mkdir /homedash && chmod 755 /homedash

COPY . .

RUN go mod download
RUN go build -ldflags="-w -s" -o app .

#####################
# Build final image #
#####################
FROM --platform=${TARGETPLATFORM:-linux/arm64} gcr.io/distroless/static-debian11:nonroot

COPY --from=builder /build/app /
COPY --from=builder /build/healthchecker /healthchecker
COPY --from=builder /homedash /homedash

EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=5s --retries=3 \
    CMD ["/healthchecker", "http://127.0.0.1:8080/api/v1/status"]

ENTRYPOINT ["/app"]