# Use the official Loki image as the base
FROM grafana/loki:latest

# Copy the configuration file into the container
COPY ./loki/loki-config.yaml /etc/loki/loki-config.yaml

# Verify the config file is copied (for debugging)
RUN ls -l /etc/loki/loki-config.yaml || echo "Config file not found"

# Copy SSL certificates into the container
COPY ./loki/sslkeys/server.pem /etc/loki/certs/server.pem
COPY ./loki/sslkeys/server.key /etc/loki/certs/server.key
COPY ./rootca/ca.crt /etc/loki/certs/ca.crt

# Expose the Loki port
EXPOSE 3100

# Set the entrypoint to the Loki binary
ENTRYPOINT ["/usr/bin/loki"]

# Specify the config file argument
CMD ["-config.file=/etc/loki/loki-config.yaml"]