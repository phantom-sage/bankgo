-- name: CreateAlert :one
INSERT INTO alerts (
    severity,
    title,
    message,
    source,
    timestamp,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetAlert :one
SELECT * FROM alerts WHERE id = $1;

-- name: ListAlerts :many
SELECT * FROM alerts
WHERE 
    ($1::text IS NULL OR severity = $1) AND
    ($2::boolean IS NULL OR acknowledged = $2) AND
    ($3::boolean IS NULL OR resolved = $3) AND
    ($4::text IS NULL OR source = $4) AND
    ($5::timestamptz IS NULL OR timestamp >= $5) AND
    ($6::timestamptz IS NULL OR timestamp <= $6)
ORDER BY timestamp DESC
LIMIT $7 OFFSET $8;

-- name: CountAlerts :one
SELECT COUNT(*) FROM alerts
WHERE 
    ($1::text IS NULL OR severity = $1) AND
    ($2::boolean IS NULL OR acknowledged = $2) AND
    ($3::boolean IS NULL OR resolved = $3) AND
    ($4::text IS NULL OR source = $4) AND
    ($5::timestamptz IS NULL OR timestamp >= $5) AND
    ($6::timestamptz IS NULL OR timestamp <= $6);

-- name: AcknowledgeAlert :one
UPDATE alerts 
SET 
    acknowledged = TRUE,
    acknowledged_by = $2,
    acknowledged_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ResolveAlert :one
UPDATE alerts 
SET 
    resolved = TRUE,
    resolved_by = $2,
    resolved_at = NOW(),
    resolved_notes = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUnresolvedAlertsCount :one
SELECT COUNT(*) FROM alerts WHERE resolved = FALSE;

-- name: GetAlertsBySource :many
SELECT * FROM alerts 
WHERE source = $1 AND resolved = FALSE
ORDER BY timestamp DESC
LIMIT $2;

-- name: DeleteOldResolvedAlerts :exec
DELETE FROM alerts 
WHERE resolved = TRUE 
AND resolved_at < $1;

-- name: GetAlertStatistics :one
SELECT 
    COUNT(*) as total_alerts,
    COUNT(*) FILTER (WHERE severity = 'critical') as critical_count,
    COUNT(*) FILTER (WHERE severity = 'warning') as warning_count,
    COUNT(*) FILTER (WHERE severity = 'info') as info_count,
    COUNT(*) FILTER (WHERE acknowledged = TRUE) as acknowledged_count,
    COUNT(*) FILTER (WHERE resolved = TRUE) as resolved_count,
    COUNT(*) FILTER (WHERE resolved = FALSE) as unresolved_count
FROM alerts
WHERE 
    ($1::timestamptz IS NULL OR timestamp >= $1) AND
    ($2::timestamptz IS NULL OR timestamp <= $2);

-- name: SearchAlerts :many
SELECT * FROM alerts
WHERE 
    ($1::text IS NULL OR (
        title ILIKE '%' || $1 || '%' OR 
        message ILIKE '%' || $1 || '%' OR
        source ILIKE '%' || $1 || '%'
    )) AND
    ($2::text IS NULL OR severity = $2) AND
    ($3::boolean IS NULL OR acknowledged = $3) AND
    ($4::boolean IS NULL OR resolved = $4) AND
    ($5::timestamptz IS NULL OR timestamp >= $5) AND
    ($6::timestamptz IS NULL OR timestamp <= $6)
ORDER BY 
    CASE WHEN $7 = 'timestamp' AND $8 = true THEN timestamp END DESC,
    CASE WHEN $7 = 'timestamp' AND $8 = false THEN timestamp END ASC,
    CASE WHEN $7 = 'severity' AND $8 = true THEN 
        CASE severity 
            WHEN 'critical' THEN 1 
            WHEN 'warning' THEN 2 
            WHEN 'info' THEN 3 
        END 
    END DESC,
    CASE WHEN $7 = 'severity' AND $8 = false THEN 
        CASE severity 
            WHEN 'info' THEN 1 
            WHEN 'warning' THEN 2 
            WHEN 'critical' THEN 3 
        END 
    END DESC,
    timestamp DESC
LIMIT $9 OFFSET $10;