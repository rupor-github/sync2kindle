package history

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	cli "github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"s2k/state"
)

func RunList(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)

	entries, err := os.ReadDir(env.Cfg.HistoryPath)
	if err != nil {
		return fmt.Errorf("unable to read history directory '%s': %w", env.Cfg.HistoryPath, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".db" {
			continue
		}
		err := report(filepath.Join(env.Cfg.HistoryPath, e.Name()), log)
		if err != nil {
			log.Error("Unable to report history", zap.String("path", filepath.Join(env.Cfg.HistoryPath, e.Name())), zap.Error(err))
		}
	}
	return nil
}

func report(dbpath string, log *zap.Logger) error {
	conn, err := sqlite.OpenConn(dbpath, sqlite.OpenReadOnly)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = sqlitex.ExecuteTransient(conn, `PRAGMA foreign_keys = ON;`, nil)
	if err != nil {
		return err
	}

	var values []string
	if err := sqlitex.Execute(conn, `SELECT value FROM identifiers;`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			values = append(values, stmt.ColumnText(0))
			return nil
		},
	}); err != nil {
		return fmt.Errorf("unable to read history identifiers: %w", err)
	}

	step, err := lastStep(conn)
	if err != nil {
		return fmt.Errorf("unable to read history last step: %w", err)
	}

	log.Info("Report", zap.String("path", dbpath), zap.Int64("last step", step), zap.Strings("identifiers", values))
	return nil
}
