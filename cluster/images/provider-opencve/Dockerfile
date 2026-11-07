FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

ARG TARGETOS
ARG TARGETARCH

COPY "bin/${TARGETOS}_${TARGETARCH}/provider" /usr/local/bin/provider

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/provider"]
