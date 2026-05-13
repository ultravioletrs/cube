# Provider Rollout Checklist

This checklist is for rolling out source providers in production with minimal risk.

## Scope

- Native providers: `google_drive`, `s3`, `microsoft` (`onedrive`, `sharepoint` aliases)
- Fallback provider: `rclone` (`dropbox` alias and future backends)

## Preconditions

1. Confirm source type config validation passes in staging for each provider.
2. Confirm provider smoke tests pass:
   - Google provider list/download smoke
   - S3 provider list/download smoke
   - Microsoft provider list/download smoke
3. Confirm embedder runtime preflight passes:
   - `EMBEDDER_RCLONE_BINARY` points to a valid binary
   - `EMBEDDER_RCLONE_CONFIG_DIR` exists and is readable

## Deployment Steps

1. Deploy embedder with read-only `rclone` config mount.
2. Enable `EMBEDDER_RCLONE_PREFLIGHT=true` (default) in all environments.
3. Verify `/metrics` endpoint is reachable from observability stack.
4. Create one source of each target type in staging and run manual sync:
   - `google_drive`
   - `s3`
   - `microsoft` (or `onedrive` / `sharepoint`)
   - `dropbox` (rclone fallback)

## Metrics To Watch

1. `cube_embedder_source_sync_runs_total{result="error"}`
2. `cube_embedder_source_sync_duration_seconds`
3. `cube_embedder_source_sync_files_total{kind="deleted"}`
4. `cube_embedder_source_download_runs_total{result="error"}`
5. `cube_embedder_source_download_duration_seconds`

## Alert Rules

1. Use the bundled alert rules file:
   - `internal/embedder/monitoring/alerts/provider-alerts.yaml`
2. Add it to your Prometheus config `rule_files` list.
3. Validate with `promtool check rules internal/embedder/monitoring/alerts/provider-alerts.yaml` before deploy.

## Rollout Gates

1. Error-rate gate: source sync and download error ratio stays under agreed SLO for 24h.
2. Latency gate: P95 sync and download duration does not regress beyond baseline.
3. Data gate: discovered/queued/deleted file counts match expected provider behavior.

## Rollback Plan

1. Disable creation of newly introduced source types in API client/feature flag.
2. Route affected source types to fallback provider where possible.
3. Revert to previous embedder image if preflight or runtime compatibility fails.
4. Keep source records intact; avoid destructive cleanup during rollback.
