package history

import (
	"encoding/json"
	"fmt"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"s2k/objects"
)

// stepInfo holds metadata about a single sync step.
type stepInfo struct {
	StepID      int64
	Source      string
	Destination string
	Created     time.Time
	ObjectCount int
}

// allSteps returns all steps in the database ordered by step_id.
func allSteps(conn *sqlite.Conn) ([]stepInfo, error) {
	var steps []stepInfo
	if err := sqlitex.Execute(conn, `
		SELECT s.step_id, s.source, s.destination, s.created, COUNT(o.path)
		FROM steps s
		LEFT JOIN objects o ON o.step_id = s.step_id
		GROUP BY s.step_id
		ORDER BY s.step_id;`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				steps = append(steps, stepInfo{
					StepID:      stmt.ColumnInt64(0),
					Source:      stmt.ColumnText(1),
					Destination: stmt.ColumnText(2),
					Created:     time.Unix(stmt.ColumnInt64(3), 0).UTC(),
					ObjectCount: int(stmt.ColumnInt64(4)),
				})
				return nil
			},
		},
	); err != nil {
		return nil, fmt.Errorf("unable to read steps: %w", err)
	}
	return steps, nil
}

// identifiers returns all identifier values from the database.
func identifiers(conn *sqlite.Conn) ([]string, error) {
	var values []string
	if err := sqlitex.Execute(conn, `SELECT value FROM identifiers;`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			values = append(values, stmt.ColumnText(0))
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("unable to read identifiers: %w", err)
	}
	return values, nil
}

// stepObjects returns the object info set for a given step.
func stepObjects(conn *sqlite.Conn, stepID int64) (objects.ObjectInfoSet, error) {
	if stepID == 0 {
		return objects.New(), nil
	}

	ois := objects.New()
	if err := sqlitex.Execute(conn, `SELECT path, data FROM objects WHERE step_id=?;`, &sqlitex.ExecOptions{
		Args: []any{stepID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			var oi objects.ObjectInfo
			path := stmt.ColumnText(0)
			data := stmt.ColumnText(1)
			if err := json.Unmarshal([]byte(data), &oi); err != nil {
				return fmt.Errorf("unable to unmarshal object info for %q: %w", path, err)
			}
			ois[path] = &oi
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("unable to retrieve step %d objects: %w", stepID, err)
	}
	return ois, nil
}

// openReadOnly opens a database in read-only mode with foreign keys enabled.
func openReadOnly(dbpath string) (*sqlite.Conn, error) {
	conn, err := sqlite.OpenConn(dbpath, sqlite.OpenReadOnly)
	if err != nil {
		return nil, err
	}
	if err := sqlitex.ExecuteTransient(conn, `PRAGMA foreign_keys = ON;`, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
