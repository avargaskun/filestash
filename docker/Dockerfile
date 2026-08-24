# STEP1: BUILD BACKEND
FROM golang:1.26-trixie AS builder_backend
WORKDIR /home/filestash/
COPY . .
RUN apt-get update > /dev/null && \
    apt-get install -y curl make > /dev/null 2>&1 && \
    apt-get install -y libjpeg-dev libtiff-dev libpng-dev libwebp-dev libraw-dev libheif-dev libgif-dev libvips-dev > /dev/null 2>&1 && \
    apt-get install -y libavcodec-dev libavdevice-dev libavfilter-dev libavformat-dev libswresample-dev libswscale-dev libavutil-dev > /dev/null 2>&1 && \
    make init && \
    make build

# STEP2: BUILD PROD IMAGE
FROM debian:stable-slim
LABEL org.opencontainers.image.source="https://github.com/avargaskun/filestash"
WORKDIR /app/
COPY --from=builder_backend /home/filestash/dist/ .
RUN sed -i 's/^Components: main$/Components: main contrib non-free non-free-firmware/' /etc/apt/sources.list.d/debian.sources && \
    apt-get update > /dev/null && \
    apt-get -y upgrade > /dev/null && \
    apt-get install -y --no-install-recommends ca-certificates curl ffmpeg libbrotli1 poppler-utils vainfo && \
    if [ "$(dpkg --print-architecture)" = "amd64" ]; then \
        apt-get install -y --no-install-recommends intel-media-va-driver-non-free; \
    fi && \
    useradd filestash && \
    mkdir -p /app/data/state/ && \
    chown -R filestash:filestash /app/ && \
    chmod 730 /app/filestash && \
    rm -rf /var/lib/apt/lists/* && \
    rm -rf /tmp/*

USER filestash
CMD ["/app/filestash"]
EXPOSE 8334
