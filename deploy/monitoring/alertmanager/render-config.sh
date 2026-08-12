#!/bin/sh
# Build Alertmanager's config from the static base plus whatever channels this
# deployment has actually configured.
#
# Alertmanager does not expand environment variables in its config file and
# validates the whole thing at startup, so an `email_configs` with an empty
# `to:` or a `telegram_configs` with an empty `bot_token` is not an inactive
# channel — it is a container that refuses to start. Rendering the receivers
# here is what makes "leave it empty and that channel is off" true.
#
# Run as the container's command, before alertmanager itself:
#   /etc/alertmanager/render-config.sh && exec /bin/alertmanager --config.file=...
#
# It writes to /tmp because /etc/alertmanager is mounted read-only: config that
# carries an SMTP password should not be writable by the process reading it,
# and a read-only mount is what stops a compromised container rewriting its own
# alerting rules to deliver nowhere.

set -eu

BASE=/etc/alertmanager/alertmanager.base.yml
OUT=/tmp/alertmanager.yml

cat "$BASE" > "$OUT"

{
    echo ""
    echo "# Appended by render-config.sh at container start."
    echo "receivers:"

    # Always present, so route.receiver always resolves. Delivery is added
    # below only where it is configured.
    echo "  - name: default"
    if [ -n "${ALERT_EMAIL_TO:-}" ] && [ -n "${ALERT_SMTP_HOST:-}" ]; then
        echo "    email_configs:"
        echo "      - to: \"${ALERT_EMAIL_TO}\""
        echo "        from: \"${ALERT_EMAIL_FROM:-alerts@localhost}\""
        echo "        smarthost: \"${ALERT_SMTP_HOST}\""
        echo "        auth_username: \"${ALERT_SMTP_USER:-}\""
        echo "        auth_password: \"${ALERT_SMTP_PASSWORD:-}\""
        echo "        require_tls: ${ALERT_SMTP_REQUIRE_TLS:-true}"
        echo "        send_resolved: true"
        echo "        headers:"
        echo "          Subject: '[Nexus/{{ .CommonLabels.severity }}] {{ .CommonAnnotations.summary }}'"
    fi

    echo "  - name: urgent"
    if [ -n "${ALERT_EMAIL_TO:-}" ] && [ -n "${ALERT_SMTP_HOST:-}" ]; then
        echo "    email_configs:"
        echo "      - to: \"${ALERT_EMAIL_TO}\""
        echo "        from: \"${ALERT_EMAIL_FROM:-alerts@localhost}\""
        echo "        smarthost: \"${ALERT_SMTP_HOST}\""
        echo "        auth_username: \"${ALERT_SMTP_USER:-}\""
        echo "        auth_password: \"${ALERT_SMTP_PASSWORD:-}\""
        echo "        require_tls: ${ALERT_SMTP_REQUIRE_TLS:-true}"
        echo "        send_resolved: true"
        echo "        headers:"
        echo "          Subject: '[Nexus/PAGE] {{ .CommonAnnotations.summary }}'"
    fi
    if [ -n "${ALERT_TELEGRAM_TOKEN:-}" ] && [ -n "${ALERT_TELEGRAM_CHAT_ID:-}" ]; then
        echo "    telegram_configs:"
        echo "      - bot_token: \"${ALERT_TELEGRAM_TOKEN}\""
        echo "        chat_id: ${ALERT_TELEGRAM_CHAT_ID}"
        echo "        parse_mode: HTML"
        echo "        send_resolved: true"
        echo "        message: '{{ template \"telegram.nexus\" . }}'"
    fi
} >> "$OUT"

# Say which channels came out, once, at startup. "No alerts arrived" is a
# question asked weeks later, and this line is the answer to it.
channels=""
[ -n "${ALERT_EMAIL_TO:-}" ] && [ -n "${ALERT_SMTP_HOST:-}" ] && channels="email"
[ -n "${ALERT_TELEGRAM_TOKEN:-}" ] && [ -n "${ALERT_TELEGRAM_CHAT_ID:-}" ] && channels="${channels:+$channels,}telegram"
echo "alertmanager: notification channels configured: ${channels:-none (alerts are visible in the UI only)}"
