FROM ghcr.io/bryk-io/shell:0.2.0

# Expose required ports
EXPOSE 9090

# Expose required volumes
VOLUME /etc/serve

# Add application binary and use it as default entrypoint
COPY serve /bin/serve
ENTRYPOINT ["/bin/serve"]
