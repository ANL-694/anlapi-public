#!/usr/bin/env bash
set -euo pipefail

mode="${1:-prepare}"
: "${DR_DUMP_FILE:?DR_DUMP_FILE is required}"
: "${DR_EXPECTED_SHA256:?DR_EXPECTED_SHA256 is required}"

database_name="${DR_DATABASE_NAME:-anlapi}"
candidate_name="${DR_CANDIDATE_DATABASE:-anlapi_dr_candidate}"
database_owner="${DR_DATABASE_OWNER:-anlapi}"
app_binary="${DR_APP_BINARY:-/opt/anlapi/anlapi}"
app_service="${DR_APP_SERVICE:-anlapi}"
app_environment_file="${DR_ENVIRONMENT_FILE:-/opt/anlapi/.env}"

actual_hash="$(sha256sum "$DR_DUMP_FILE" | awk '{print $1}')"
if [[ "$actual_hash" != "${DR_EXPECTED_SHA256,,}" ]]; then
  echo 'Failback dump SHA-256 mismatch.' >&2
  exit 1
fi

audit_oauth_vault_with_systemd_unit() (
  set -Eeuo pipefail

  local target="$1"
  local audit_log=''
  local audit_unit_tmp=''
  local audit_unit_file=''
  local audit_unit_name=''
  local audit_override_file=''
  local systemd_configuration_changed=0
  local audit_start_attempted=0
  local audit_unit_stopped=0
  local preserve_audit_unit=0
  local cleanup_failed=0

  cleanup() {
    local original_status="$?"
    local active_state=''
    local path

    trap - EXIT
    trap '' INT TERM
    set +e

    if [[ "$audit_start_attempted" == '1' ]]; then
      if ! sudo systemctl stop "$audit_unit_name"; then
        echo "Unable to stop temporary OAuth audit unit: $audit_unit_name" >&2
        cleanup_failed=1
      fi
      active_state="$(sudo systemctl show --property=ActiveState --value "$audit_unit_name" 2>/dev/null)"
      case "$active_state" in
        inactive|failed) audit_unit_stopped=1 ;;
        *)
          echo "Temporary OAuth audit unit is not confirmed stopped: $audit_unit_name ($active_state)" >&2
          echo 'Leaving the temporary unit and environment file in place for manual inspection.' >&2
          preserve_audit_unit=1
          cleanup_failed=1
          ;;
      esac
      if [[ "$audit_unit_stopped" == '1' ]] && ! sudo systemctl reset-failed "$audit_unit_name"; then
        echo "Unable to clear temporary OAuth audit unit state: $audit_unit_name" >&2
        cleanup_failed=1
      fi
    fi

    for path in "$audit_unit_tmp"; do
      [[ -n "$path" ]] || continue
      if ! sudo rm -f -- "$path" || ! sudo test ! -e "$path"; then
        echo "Unable to remove temporary OAuth audit file: $path" >&2
        cleanup_failed=1
      fi
    done

    if [[ "$preserve_audit_unit" != '1' && -n "$audit_unit_file" ]]; then
      if ! sudo rm -f -- "$audit_unit_file" || ! sudo test ! -e "$audit_unit_file"; then
        echo "Unable to remove temporary OAuth audit unit: $audit_unit_file" >&2
        echo 'Preserving its environment file and skipping systemd reload to avoid a dangling unit reference.' >&2
        preserve_audit_unit=1
        cleanup_failed=1
      fi
    fi

    if [[ "$preserve_audit_unit" != '1' && "$systemd_configuration_changed" == '1' ]]; then
      if sudo systemctl daemon-reload; then
        systemd_configuration_changed=0
      else
        echo 'Unable to reload systemd after removing the temporary OAuth audit unit.' >&2
        echo 'Preserving the environment file in case the system manager still references it.' >&2
        preserve_audit_unit=1
        cleanup_failed=1
      fi
    fi

    if [[ "$preserve_audit_unit" != '1' && -n "$audit_override_file" ]]; then
      if ! sudo rm -f -- "$audit_override_file" || ! sudo test ! -e "$audit_override_file"; then
        echo "Unable to remove temporary OAuth audit environment file: $audit_override_file" >&2
        cleanup_failed=1
      fi
    fi

    if [[ -n "$audit_log" ]] && { ! rm -f -- "$audit_log" || [[ -e "$audit_log" ]]; }; then
      echo 'Unable to remove the temporary OAuth audit log.' >&2
      cleanup_failed=1
    fi

    if [[ "$cleanup_failed" == '1' ]]; then
      echo 'OAuth audit temporary-resource cleanup failed; inspect systemd and /run before retrying.' >&2
      [[ "$original_status" -ne 0 ]] || original_status=1
    fi
    exit "$original_status"
  }

  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  audit_log="$(mktemp /tmp/anl-oauth-vault-audit.XXXXXX)"
  audit_unit_tmp="$(mktemp /run/systemd/system/anlapi-oauth-vault-audit.XXXXXX)"
  audit_unit_file="${audit_unit_tmp}.service"
  audit_unit_name="$(basename "$audit_unit_file")"
  audit_override_file="$(mktemp /run/systemd/system/anlapi-oauth-vault-audit-env.XXXXXX)"

  mv -- "$audit_unit_tmp" "$audit_unit_file"
  audit_unit_tmp=''
  systemd_configuration_changed=1

  printf 'DATABASE_DBNAME=%s\nOAUTH_VAULT_MODE=external\nOAUTH_VAULT_ALLOW_LEGACY_FALLBACK=false\n' "$target" > "$audit_override_file"
  {
    printf '%s\n' '[Unit]' 'Description=ANLAPI OAuth Vault audit' '[Service]' 'Type=oneshot' 'TimeoutStartSec=300' 'TimeoutStopSec=10'
    printf 'EnvironmentFile=%s\n' "$app_environment_file"
    printf 'EnvironmentFile=%s\n' "$audit_override_file"
    [[ -z "$app_user" ]] || printf 'User=%s\n' "$app_user"
    [[ -z "$app_group" ]] || printf 'Group=%s\n' "$app_group"
    [[ -z "$working_directory" ]] || printf 'WorkingDirectory=%s\n' "$working_directory"
    printf '%s\n' 'StandardOutput=journal' 'StandardError=journal'
    printf 'ExecStart=%s --audit-oauth-vault\n' "$app_binary"
  } > "$audit_unit_file"

  sudo systemctl daemon-reload
  audit_start_attempted=1
  sudo systemctl start "$audit_unit_name"
  sudo journalctl --unit="$audit_unit_name" --no-pager --output=cat > "$audit_log"
  if ! grep -q 'sensitive_rows=0' "$audit_log" || ! grep -q 'missing_vault_entries=0' "$audit_log"; then
    echo 'OAuth vault audit did not confirm zero sensitive rows and zero missing vault entries.' >&2
    exit 1
  fi
)

audit_core_database() {
  local target="$1"
  local sensitive_rows
  local app_user
  local app_group
  local working_directory
  local unit_path_pattern='^/[A-Za-z0-9_./-]+$'

  if [[ ! -r "$app_environment_file" ]]; then
    echo "Application environment file is not readable: $app_environment_file" >&2
    return 1
  fi
  if [[ ! "$target" =~ ^[A-Za-z_][A-Za-z0-9_]*$ || ! "$app_environment_file" =~ $unit_path_pattern || ! "$app_binary" =~ $unit_path_pattern ]]; then
    echo 'OAuth audit target, environment file, or binary has unsupported characters.' >&2
    return 1
  fi
  app_user="$(systemctl show --property=User --value "$app_service")"
  app_group="$(systemctl show --property=Group --value "$app_service")"
  working_directory="$(systemctl show --property=WorkingDirectory --value "$app_service")"
  if [[ -n "$app_user" && ! "$app_user" =~ ^[A-Za-z0-9_.-]+$ ]]; then
    echo 'Application unit User has unsupported characters.' >&2
    return 1
  fi
  if [[ -n "$app_group" && ! "$app_group" =~ ^[A-Za-z0-9_.-]+$ ]]; then
    echo 'Application unit Group has unsupported characters.' >&2
    return 1
  fi
  if [[ -n "$working_directory" && ! "$working_directory" =~ $unit_path_pattern ]]; then
    echo 'Application unit WorkingDirectory has unsupported characters.' >&2
    return 1
  fi

  sensitive_rows="$(sudo -u postgres psql -X -At -v ON_ERROR_STOP=1 -d "$target" <<'SQL'
WITH credential_keys AS (
  SELECT DISTINCT a.id,
    replace(replace(lower(trim(k.key)), '-', '_'), '.', '_') AS normalized
  FROM accounts a
  CROSS JOIN LATERAL jsonb_object_keys(COALESCE(a.credentials, '{}'::jsonb)) AS k(key)
  WHERE a.deleted_at IS NULL AND a.type IN ('oauth', 'setup-token')
)
SELECT count(DISTINCT id)
FROM credential_keys
WHERE normalized IN ('access_token','refresh_token','id_token','oauth_token','session_token','session_key','cookie','cookies','client_secret','authorization','authorization_header')
   OR normalized LIKE '%\_secret' ESCAPE '\'
   OR (normalized LIKE '%token%' AND normalized NOT IN ('token_type','oauth_type','token_version','_token_version','oauth_token_version'));
SQL
)"
  printf 'candidate_oauth_sensitive_rows=%s\n' "$sensitive_rows"
  [[ "$sensitive_rows" == '0' ]]

  audit_oauth_vault_with_systemd_unit "$target"
}

prepare_candidate() {
  sudo -u postgres psql -X -v ON_ERROR_STOP=1 -d postgres \
    -v candidate="$candidate_name" \
    -v owner="$database_owner" <<'SQL'
SELECT format('DROP DATABASE IF EXISTS %I WITH (FORCE)', :'candidate') \gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'candidate', :'owner') \gexec
SQL
  sudo -u postgres pg_restore --no-owner --no-privileges --role="$database_owner" --dbname="$candidate_name" "$DR_DUMP_FILE"
  audit_core_database "$candidate_name"
  sudo -u postgres psql -X -At -v ON_ERROR_STOP=1 -d "$candidate_name" <<'SQL'
SELECT 'users=' || count(*) FROM users WHERE deleted_at IS NULL;
SELECT 'api_keys=' || count(*) FROM api_keys WHERE deleted_at IS NULL;
SELECT 'accounts=' || count(*) FROM accounts WHERE deleted_at IS NULL;
SELECT 'pending_payments=' || count(*) FROM payment_orders WHERE status = 'pending';
SQL
  echo 'candidate_ready=true'
}

activate_candidate() {
  if [[ "${DR_ACTIVATION_CONFIRMATION:-}" != 'DOMESTIC_STOPPED_AND_CANDIDATE_VERIFIED' ]]; then
    echo 'Set DR_ACTIVATION_CONFIRMATION=DOMESTIC_STOPPED_AND_CANDIDATE_VERIFIED before activation.' >&2
    exit 2
  fi
  audit_core_database "$candidate_name"
  local backup_name
  backup_name="${database_name}_before_failback_$(date -u +%Y%m%dT%H%M%SZ)"
  systemctl stop "$app_service"
  sudo -u postgres psql -X -v ON_ERROR_STOP=1 -d postgres \
    -v current="$database_name" \
    -v candidate="$candidate_name" \
    -v backup="$backup_name" <<'SQL'
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname IN (:'current', :'candidate') AND pid <> pg_backend_pid();
SELECT format('ALTER DATABASE %I RENAME TO %I', :'current', :'backup') \gexec
SELECT format('ALTER DATABASE %I RENAME TO %I', :'candidate', :'current') \gexec
SQL
  if systemctl start "$app_service" && curl --fail --silent --show-error --max-time 30 http://127.0.0.1:8080/health >/dev/null; then
    printf 'old_core_database=%s\n' "$backup_name"
    echo 'failback_activation=true'
    return
  fi

  systemctl stop "$app_service" || true
  sudo -u postgres psql -X -v ON_ERROR_STOP=1 -d postgres \
    -v current="$database_name" \
    -v backup="$backup_name" \
    -v failed="${candidate_name}_failed" <<'SQL'
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = :'current' AND pid <> pg_backend_pid();
SELECT format('ALTER DATABASE %I RENAME TO %I', :'current', :'failed') \gexec
SELECT format('ALTER DATABASE %I RENAME TO %I', :'backup', :'current') \gexec
SQL
  systemctl start "$app_service"
  echo 'Candidate activation failed and the previous US core database was restored.' >&2
  exit 1
}

case "$mode" in
  prepare) prepare_candidate ;;
  activate) activate_candidate ;;
  *) echo 'Usage: Restore-AnlDrCore.sh [prepare|activate]' >&2; exit 2 ;;
esac
