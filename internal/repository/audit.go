package repository

import (
	"context"
	"database/sql"
	"time"

	"supermarket/internal/models"
)

type AuditRepository struct{ db *sql.DB }

func NewAuditRepository(db *sql.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Insert(ctx context.Context, entry models.AuditEntry) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO audit_log(user_id,action,module,record_id,description) VALUES(?,?,?,?,?)`,
		entry.UserID, entry.Action, entry.Module, entry.RecordID, entry.Description)
	return err
}

func (r *AuditRepository) InsertTx(ctx context.Context, tx *sql.Tx, entry models.AuditEntry) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_log(user_id,action,module,record_id,description) VALUES(?,?,?,?,?)`,
		entry.UserID, entry.Action, entry.Module, entry.RecordID, entry.Description)
	return err
}

func (r *AuditRepository) List(ctx context.Context, from, to, module string, userID int64) ([]models.AuditEntry, error) {
	q := `SELECT id,user_id,action,module,record_id,COALESCE(description,''),created_at FROM audit_log WHERE 1=1`
	args := []any{}
	if from != "" {
		q += ` AND date(created_at)>=?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND date(created_at)<=?`
		args = append(args, to)
	}
	if userID > 0 {
		q += ` AND user_id=?`
		args = append(args, userID)
	}
	if module != "" {
		q += ` AND module=?`
		args = append(args, module)
	}
	q += ` ORDER BY created_at DESC,id DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		var record sql.NullInt64
		var created string
		if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Module, &record, &e.Description, &created); err != nil {
			return nil, err
		}
		if record.Valid {
			e.RecordID = &record.Int64
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, e)
	}
	return out, rows.Err()
}
