FROM alpine:3.18
LABEL maintainer="Marvin Menzerath <dev@marvin.pro>"

ENV MODE=production

RUN apk add -U --no-cache ca-certificates

WORKDIR /app
COPY --chmod=0755 build/mcgen_linux_amd64 mcgen

ENTRYPOINT ["./mcgen"]
