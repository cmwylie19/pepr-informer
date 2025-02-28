FROM golang:1.24.0-alpine3.21 AS builder

LABEL description="pepr-informer" \
      maintainer="Casey Wylie casewylie@gmail.com"

WORKDIR /app
COPY . .
RUN go mod download && go mod verify
RUN go build 

FROM scratch
WORKDIR /app
COPY --from=builder /app/pepr-informer ./

ENTRYPOINT ["./pepr-informer"]
