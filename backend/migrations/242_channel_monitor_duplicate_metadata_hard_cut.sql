-- Rename persisted operation markers into the ANLAPI-owned namespace without
-- retaining a dual-read compatibility path. Any previous namespace is only a
-- one-time migration source; the application writes the ANLAPI key afterward.
DO $$
DECLARE
    old_key TEXT;
BEGIN
    FOR old_key IN
        SELECT DISTINCT key_name
        FROM channel_monitors
        CROSS JOIN LATERAL jsonb_object_keys(COALESCE(extra_headers, '{}'::jsonb)) AS key_name
        WHERE key_name <> 'anlapi:duplicate_operation_id'
          AND key_name LIKE '%:duplicate_operation_id'
    LOOP
        IF EXISTS (
            SELECT 1
            FROM channel_monitors
            WHERE extra_headers ? old_key
              AND extra_headers ? 'anlapi:duplicate_operation_id'
              AND extra_headers -> old_key
                  IS DISTINCT FROM extra_headers -> 'anlapi:duplicate_operation_id'
        ) THEN
            RAISE EXCEPTION 'channel monitor duplicate operation metadata contains conflicting old and new values';
        END IF;

        UPDATE channel_monitors
        SET extra_headers = jsonb_set(
            extra_headers - old_key,
            '{anlapi:duplicate_operation_id}',
            extra_headers -> old_key,
            true
        )
        WHERE extra_headers ? old_key;
    END LOOP;
END;
$$;
