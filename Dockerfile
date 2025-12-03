FROM alpine:3.23
LABEL maintainer="Marvin Menzerath <dev@marvin.pro>"

ENV MODE=production

RUN apk add -U --no-cache ca-certificates

WORKDIR /app
COPY --chmod=0755 build/linux_amd64/mcgen_linux_amd64 mcgen

EXPOSE 8080
EXPOSE 9100

ENTRYPOINT ["./mcgen"]
