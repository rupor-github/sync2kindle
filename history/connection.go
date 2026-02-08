package history

import (
	"encoding/json"
	"fmt"
	"time"

	ole "github.com/go-ole/go-ole"
	"go.uber.org/zap"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"s2k/objects"
)

// should be usable in the zap log.Named()
const driverName = "history"

type Connection struct {
	log    *zap.Logger
	conn   *sqlite.Conn
	stepID int64
}

func Connect(path string, log *zap.Logger) (*Connection, error) {
	conn, err := sqlite.OpenConn(path, sqlite.OpenReadWrite)
	if err != nil {
		return nil, err
	}
	err = sqlitex.ExecuteTransient(conn, `PRAGMA foreign_keys = ON;`, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	stepID, err := lastStep(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("unable to read last history step value: %w", err)
	}
	return &Connection{log: log.Named(driverName), conn: conn, stepID: stepID}, nil
}

func (c *Connection) Disconnect() {
	if c == nil || c.conn == nil {
		return
	}
	if err := c.conn.Close(); err != nil {
		c.log.Error("Problems closing history database", zap.Error(err))
	}
	c.conn = nil
}

// driver interface

func (c *Connection) Name() string {
	return driverName
}

func (c *Connection) UniqueID() string {
	return driverName
}

func (c *Connection) MkDir(obj *objects.ObjectInfo) error {
	return ole.NewError(ole.E_NOTIMPL)
}

func (c *Connection) Remove(obj *objects.ObjectInfo) error {
	return ole.NewError(ole.E_NOTIMPL)
}

func (c *Connection) Copy(obj *objects.ObjectInfo) error {
	return ole.NewError(ole.E_NOTIMPL)
}

func (c *Connection) GetObjectInfos() (ois objects.ObjectInfoSet, err error) {
	ois, err = stepObjectInfos(c.conn, c.stepID)
	if err != nil {
		return nil, err
	}
	return
}

// implementation

func (c *Connection) StepID() int64 {
	if c == nil {
		return -1
	}
	return c.stepID
}

func (c *Connection) SaveObjectInfos(src, dst string, ois objects.ObjectInfoSet) (err error) {
	var (
		stepID = int64(-1)
		endFn  func(*error)
	)

	endFn, err = sqlitex.ImmediateTransaction(c.conn)
	if err != nil {
		return fmt.Errorf("unable to start transaction: %w", err)
	}
	defer func() {
		if err == nil && stepID > 0 {
			c.stepID = stepID
		}
		endFn(&err)
	}()

	stepID, err = nextStep(c.conn, src, dst)
	if err != nil {
		return
	}

	for k, v := range ois {
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("unable to marshal object info for '%s': %w", k, err)
		}
		if err := sqlitex.Execute(c.conn, `INSERT INTO objects (step_id, path, data) VALUES (?, ?, json(?));`, &sqlitex.ExecOptions{
			Args: []any{stepID, k, string(data)},
		}); err != nil {
			return fmt.Errorf("unable to save object '%s' in history: %w", k, err)
		}
	}
	return
}

func lastStep(conn *sqlite.Conn) (int64, error) {
	var step int64
	if err := sqlitex.Execute(conn, `SELECT step_id FROM steps ORDER BY 1 DESC LIMIt 1;`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			step = stmt.ColumnInt64(0)
			return nil
		},
	}); err != nil {
		return 0, fmt.Errorf("unable to read last history step value: %w", err)
	}
	return step, nil
}

func nextStep(conn *sqlite.Conn, src, dst string) (int64, error) {
	if err := sqlitex.Execute(conn, `INSERT INTO steps (source, destination, created) VALUES (?, ?, ?);`, &sqlitex.ExecOptions{
		Args: []any{src, dst, time.Now().UTC().Unix()},
	}); err != nil {
		return 0, fmt.Errorf("unable to create next history step: %w", err)
	}
	return lastStep(conn)
}

func stepObjectInfos(conn *sqlite.Conn, stepID int64) (objects.ObjectInfoSet, error) {
	if stepID == 0 {
		return objects.ObjectInfoSet{}, nil
	}

	ois := objects.New()
	if err := sqlitex.Execute(conn, `SELECT path, data FROM objects WHERE step_id=?;`, &sqlitex.ExecOptions{
		Args: []any{stepID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			var oi objects.ObjectInfo
			path := stmt.ColumnText(0)
			data := stmt.ColumnText(1)
			if err := json.Unmarshal([]byte(data), &oi); err != nil {
				return fmt.Errorf("unable to unmarshal object info: %w", err)
			}
			ois[path] = &oi
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("unable to retrieve step '%d' objects from history: %w", stepID, err)
	}
	return ois, nil
}
