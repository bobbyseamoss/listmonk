#!/bin/sh

set -e

export PUID=${PUID:-0}
export PGID=${PGID:-0}
export GROUP_NAME="app"
export USER_NAME="app"

# This function evaluates if the supplied PGID is already in use
# if it is not in use, it creates the group with the PGID
# if it is in use, it sets the GROUP_NAME to the existing group
create_group() {
  if ! getent group ${PGID} > /dev/null 2>&1; then
    addgroup -g ${PGID} ${GROUP_NAME}
  else
    existing_group=$(getent group ${PGID} | cut -d: -f1)
    export GROUP_NAME=${existing_group}
  fi
}

# This function evaluates if the supplied PUID is already in use
# if it is not in use, it creates the user with the PUID and PGID
create_user() {
  if ! getent passwd ${PUID} > /dev/null 2>&1; then
    adduser -u ${PUID} -G ${GROUP_NAME} -s /bin/sh -D ${USER_NAME}
  else
    existing_user=$(getent passwd ${PUID} | cut -d: -f1)
    export USER_NAME=${existing_user}
  fi
}

# Run the needed functions to create the user and group
create_group
create_user

load_secret_files() {
  # Save and restore IFS
  old_ifs="$IFS"
  IFS='
'
  # Capture all env variables starting with LISTMONK_ and ending with _FILE.
  # It's value is assumed to be a file path with its actual value.
  for line in $(env | grep '^LISTMONK_.*_FILE='); do
    var="${line%%=*}"
    fpath="${line#*=}"

    # If it's a valid file, read its contents and assign it to the var
    # without the _FILE suffix.
    # Eg: LISTMONK_DB_USER_FILE=/run/secrets/user -> LISTMONK_DB_USER=$(contents of /run/secrets/user)
    if [ -f "$fpath" ]; then
      new_var="${var%_FILE}"
      export "$new_var"="$(cat "$fpath")"
    fi
  done
  IFS="$old_ifs"
}

# Load env variables from files if LISTMONK_*_FILE variables are set.
load_secret_files

# Try to set the ownership of the app directory to the app user.
if ! chown -R ${PUID}:${PGID} /listmonk 2>/dev/null; then
  echo "Warning: Failed to change ownership of /listmonk. Readonly volume?"
fi

echo "Launching listmonk with user=[${USER_NAME}] group=[${GROUP_NAME}] PUID=[${PUID}] PGID=[${PGID}]"

# Override database configuration with environment variables if set
echo "Applying database configuration from environment variables..."
if [ -n "${LISTMONK_DB_HOST}" ]; then
  sed -i "s|^host = .*|host = \"${LISTMONK_DB_HOST}\"|" /listmonk/config.toml
  echo "  ✓ DB host: ${LISTMONK_DB_HOST}"
fi
if [ -n "${LISTMONK_DB_PORT}" ]; then
  sed -i "s|^port = .*|port = ${LISTMONK_DB_PORT}|" /listmonk/config.toml
  echo "  ✓ DB port: ${LISTMONK_DB_PORT}"
fi
if [ -n "${LISTMONK_DB_USER}" ]; then
  sed -i "s|^user = .*|user = \"${LISTMONK_DB_USER}\"|" /listmonk/config.toml
  echo "  ✓ DB user: ${LISTMONK_DB_USER}"
fi
if [ -n "${LISTMONK_DB_PASSWORD}" ]; then
  sed -i "s|^password = .*|password = \"${LISTMONK_DB_PASSWORD}\"|" /listmonk/config.toml
  echo "  ✓ DB password: ***"
fi
if [ -n "${LISTMONK_DB_DATABASE}" ]; then
  sed -i "s|^database = .*|database = \"${LISTMONK_DB_DATABASE}\"|" /listmonk/config.toml
  echo "  ✓ DB database: ${LISTMONK_DB_DATABASE}"
fi
if [ -n "${LISTMONK_DB_SSL_MODE}" ]; then
  sed -i "s|^ssl_mode = .*|ssl_mode = \"${LISTMONK_DB_SSL_MODE}\"|" /listmonk/config.toml
  echo "  ✓ DB SSL mode: ${LISTMONK_DB_SSL_MODE}"
fi

# Auto-installation and configuration initialization
if [ "${AUTO_INSTALL}" = "true" ] || [ "${AUTO_INSTALL}" = "1" ]; then
  echo "AUTO_INSTALL enabled - checking database initialization..."

  # Check if database is already initialized by trying to query settings table
  if ! /listmonk/listmonk --check-db 2>/dev/null; then
    echo "Database not initialized. Running installation..."

    # Run install command
    /listmonk/listmonk --install --yes --idempotent

    echo "✓ Database initialized"

    # Run configuration initialization if script exists and has templates
    if [ -f /listmonk/deployment/scripts/init_config.sh ] && [ -d /listmonk/deployment/configs ]; then
      echo "Running configuration initialization..."
      if /bin/sh /listmonk/deployment/scripts/init_config.sh; then
        echo "✓ Configuration initialized"
      else
        echo "⚠️  Warning: Configuration initialization failed, continuing anyway"
      fi
    else
      echo "⚠️  Skipping configuration initialization (templates not found)"
    fi
  else
    echo "✓ Database already initialized"
  fi

  # Always run upgrade (idempotent)
  echo "Running database upgrade..."
  /listmonk/listmonk --upgrade --yes
  echo "✓ Database up to date"
fi

# Enable bounce webhooks in config.toml if not already present
echo "Configuring bounce webhooks..."
if ! grep -q "^\[bounce\]" /listmonk/config.toml 2>/dev/null; then
  echo "" >> /listmonk/config.toml
  echo "[bounce]" >> /listmonk/config.toml
  echo "webhooks_enabled = true" >> /listmonk/config.toml
  echo "" >> /listmonk/config.toml
  echo "[bounce.azure]" >> /listmonk/config.toml
  echo "enabled = true" >> /listmonk/config.toml
  echo "✓ Bounce webhooks enabled in config"
else
  # Update existing bounce section
  if ! grep -q "webhooks_enabled" /listmonk/config.toml 2>/dev/null; then
    sed -i '/^\[bounce\]/a webhooks_enabled = true' /listmonk/config.toml
    echo "✓ Added webhooks_enabled to existing [bounce] section"
  fi
  if ! grep -q "^\[bounce.azure\]" /listmonk/config.toml 2>/dev/null; then
    echo "" >> /listmonk/config.toml
    echo "[bounce.azure]" >> /listmonk/config.toml
    echo "enabled = true" >> /listmonk/config.toml
    echo "✓ Added [bounce.azure] section"
  fi
fi

# If running as root and PUID is not 0, then execute command as PUID
# this allows us to run the container as a non-root user
if [ "$(id -u)" = "0" ] && [ "${PUID}" != "0" ]; then
  su-exec ${PUID}:${PGID} "$@"
else
  exec "$@"
fi
