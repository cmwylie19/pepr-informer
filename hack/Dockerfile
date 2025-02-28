FROM alpine:3.18

RUN apk add --no-cache \
    curl \
    tar \
    bash && \
    curl -L https://github.com/fullstorydev/grpcurl/releases/download/v1.8.7/grpcurl_1.8.7_linux_x86_64.tar.gz -o grpcurl.tar.gz && \
    tar -xzvf grpcurl.tar.gz -C /usr/local/bin && \
    chmod +x /usr/local/bin/grpcurl && \
    rm grpcurl.tar.gz

CMD ["sleep", "infinity"]
