FROM alpine:3.20
RUN apk add --no-cache bash jq coreutils ca-certificates curl

ENV VEGETA_VER=v12.12.0
RUN curl -sSLf https://github.com/tsenart/vegeta/releases/download/${VEGETA_VER}/vegeta_${VEGETA_VER#v}_linux_amd64.tar.gz | tar -xz -C /usr/local/bin vegeta
COPY mixed.sh /scripts/mixed.sh
RUN chmod +x /scripts/mixed.sh
ENTRYPOINT ["/scripts/mixed.sh"]