FROM gcr.io/distroless/static
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/gatekeeper /usr/local/bin/gatekeeper
ENTRYPOINT [ "/usr/local/bin/gatekeeper" ]
