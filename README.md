# External logs with Grafana Cloud

The optional Alloy service forwards new logs from the `app` container to Grafana Cloud Logs. It starts at the end of the existing Docker log file, so old entries are not uploaded on first setup. Alloy keeps its read positions in a named volume so it can resume after a restart.

1. Create a Grafana Cloud stack and a token with permission to write logs.
2. Add these values to the server's `.env` file (see `.env.example`):
   - `GRAFANA_CLOUD_LOKI_URL`: the full Loki push URL ending in `/loki/api/v1/push`.
   - `GRAFANA_CLOUD_LOKI_USER`: the Loki instance ID / Basic Auth username.
   - `GRAFANA_CLOUD_LOKI_TOKEN`: the token used as the Basic Auth password.
3. Start Alloy from the project directory:

   ```sh
   docker compose --profile observability up -d alloy
   ```

The Alloy service reads Docker metadata through `/var/run/docker.sock` and tails the Snap Docker log directory mounted read-only. Treat access to the Docker socket as privileged. Logs are sent to Grafana Cloud, so avoid logging credentials, webhook request bodies, or full payment objects.
