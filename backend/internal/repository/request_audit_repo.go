package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type requestAuditRepository struct {
	db *sql.DB
}

func NewRequestAuditRepository(db *sql.DB) service.RequestAuditRepository {
	return &requestAuditRepository{db: db}
}

const requestAuditColumns = `created_at, request_id, client_request_id, user_id, api_key_id, account_id,
method, path, model, stream, status_code, duration_ms, client_ip, user_agent, request_headers,
request_content_type, request_body, request_body_size, request_truncated,
response_content_type, response_body, response_body_size, response_truncated`

const requestAuditColumnCount = 23

func (r *requestAuditRepository) BatchInsert(ctx context.Context, logs []*service.RequestAuditLog) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil request audit repository")
	}
	if len(logs) == 0 {
		return 0, nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO request_audit_logs (")
	sb.WriteString(requestAuditColumns)
	sb.WriteString(") VALUES ")
	args := make([]any, 0, len(logs)*requestAuditColumnCount)
	n := 0
	for _, l := range logs {
		if l == nil {
			continue
		}
		if n > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(")
		for i := 0; i < requestAuditColumnCount; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, "$%d", len(args)+i+1)
		}
		sb.WriteString(")")
		args = append(args, requestAuditInsertValues(l)...)
		n++
	}
	if n == 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, sb.String(), args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func requestAuditInsertValues(l *service.RequestAuditLog) []any {
	createdAt := l.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	headers := "{}"
	if len(l.RequestHeaders) > 0 {
		if encoded, err := json.Marshal(l.RequestHeaders); err == nil {
			headers = string(encoded)
		}
	}
	return []any{
		createdAt.UTC(),
		truncateString(l.RequestID, 128),
		truncateString(l.ClientRequestID, 128),
		nullInt64Ptr(l.UserID),
		nullInt64Ptr(l.APIKeyID),
		nullInt64Ptr(l.AccountID),
		truncateString(l.Method, 16),
		truncateString(l.Path, 512),
		truncateString(l.Model, 255),
		l.Stream,
		l.StatusCode,
		l.DurationMs,
		truncateString(l.ClientIP, 64),
		truncateString(l.UserAgent, 512),
		headers,
		truncateString(l.RequestContentType, 255),
		l.RequestBody,
		l.RequestBodySize,
		l.RequestTruncated,
		truncateString(l.ResponseContentType, 255),
		l.ResponseBody,
		l.ResponseBodySize,
		l.ResponseTruncated,
	}
}

const requestAuditSelect = `SELECT id, ` + requestAuditColumns + ` FROM request_audit_logs `

// FindForUsage 先按 client_request_id / request_id 精确关联，取时间最接近的一条；
// 找不到时按 api_key_id + 时间窗口兜底（例如 usage 使用了上游 request id）。
func (r *requestAuditRepository) FindForUsage(ctx context.Context, q service.RequestAuditLookup) (*service.RequestAuditLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil request audit repository")
	}
	target := q.Target.UTC()
	if q.ClientRequestID != "" || q.RequestID != "" {
		log, err := r.scanOne(ctx, requestAuditSelect+`
WHERE ($1 <> '' AND client_request_id = $1) OR ($2 <> '' AND request_id = $2)
ORDER BY ABS(EXTRACT(EPOCH FROM (created_at - $3::timestamptz))) LIMIT 1`, q.ClientRequestID, q.RequestID, target)
		if err != nil || log != nil {
			return log, err
		}
	}
	if q.APIKeyID <= 0 || q.From.IsZero() || q.To.IsZero() {
		return nil, nil
	}
	return r.scanOne(ctx, requestAuditSelect+`
WHERE api_key_id = $1 AND created_at BETWEEN $2 AND $3
ORDER BY ABS(EXTRACT(EPOCH FROM (created_at - $4::timestamptz))) LIMIT 1`, q.APIKeyID, q.From.UTC(), q.To.UTC(), target)
}

func (r *requestAuditRepository) scanOne(ctx context.Context, query string, args ...any) (*service.RequestAuditLog, error) {
	var (
		l                           service.RequestAuditLog
		userID, apiKeyID, accountID sql.NullInt64
		headersRaw                  []byte
	)
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&l.ID, &l.CreatedAt, &l.RequestID, &l.ClientRequestID, &userID, &apiKeyID, &accountID,
		&l.Method, &l.Path, &l.Model, &l.Stream, &l.StatusCode, &l.DurationMs, &l.ClientIP, &l.UserAgent, &headersRaw,
		&l.RequestContentType, &l.RequestBody, &l.RequestBodySize, &l.RequestTruncated,
		&l.ResponseContentType, &l.ResponseBody, &l.ResponseBodySize, &l.ResponseTruncated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		l.UserID = &userID.Int64
	}
	if apiKeyID.Valid {
		l.APIKeyID = &apiKeyID.Int64
	}
	if accountID.Valid {
		l.AccountID = &accountID.Int64
	}
	if len(headersRaw) > 0 {
		_ = json.Unmarshal(headersRaw, &l.RequestHeaders)
	}
	return &l, nil
}

func (r *requestAuditRepository) DeleteBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil request audit repository")
	}
	if batchSize <= 0 {
		batchSize = 2000
	}
	res, err := r.db.ExecContext(ctx, `
WITH batch AS (
  SELECT id FROM request_audit_logs WHERE created_at < $1 ORDER BY id LIMIT $2
)
DELETE FROM request_audit_logs WHERE id IN (SELECT id FROM batch)`, cutoff.UTC(), batchSize)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
