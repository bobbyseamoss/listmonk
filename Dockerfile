FROM alpine:latest

# Install dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec jq postgresql-client

# Set the working directory
WORKDIR /listmonk

# Copy only the necessary files
COPY listmonk .
COPY config.toml.sample .
COPY config.toml.sample config.toml
COPY queries.sql .
COPY schema.sql .
COPY permissions.json .
COPY migrations ./migrations
COPY static ./static
COPY i18n ./i18n
COPY frontend/dist ./frontend/dist

# Copy deployment scripts and configuration templates
COPY deployment ./deployment

# Copy the entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/

# Make the binary, entrypoint script and deployment scripts executable
RUN chmod +x /listmonk/listmonk && \
    chmod +x /usr/local/bin/docker-entrypoint.sh && \
    chmod +x /listmonk/deployment/scripts/*.sh

# Expose the application port
EXPOSE 9000

# Set the entrypoint
ENTRYPOINT ["docker-entrypoint.sh"]

# Define the command to run the application
CMD ["/listmonk/listmonk"]
