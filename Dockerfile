FROM ghcr.io/bryk-io/shell:0.2.0@sha256:482a248b09682a878183322a56b8a36dc7e6585d8b8207872fadd475b3f64a48

# Expose required ports
EXPOSE 9090

# Expose required volumes
VOLUME /etc/serve

# Add application binary and use it as default entrypoint
COPY serve /bin/serve
ENTRYPOINT ["/bin/serve"]
