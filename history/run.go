package history

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	cli "github.com/urfave/cli/v3"
	"go.uber.org/zap"

	"s2k/objects"
	"s2k/state"
)

// RunList lists basic details for all local history database files.
func RunList(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)

	return forEachDB(env.Cfg.HistoryPath, "", log, func(dbpath string) error {
		return reportList(dbpath, log)
	})
}

func reportList(dbpath string, log *zap.Logger) error {
	conn, err := openReadOnly(dbpath)
	if err != nil {
		return err
	}
	defer conn.Close()

	values, err := identifiers(conn)
	if err != nil {
		return err
	}

	step, err := lastStep(conn)
	if err != nil {
		return fmt.Errorf("unable to read history last step: %w", err)
	}

	log.Info("Report", zap.String("id", shortID(dbpath)), zap.String("path", dbpath), zap.Int64("last step", step), zap.Strings("identifiers", values))
	return nil
}

// RunSteps lists all sync steps for each history database.
func RunSteps(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)

	return forEachDB(env.Cfg.HistoryPath, cmd.String("db"), log, func(dbpath string) error {
		return reportSteps(dbpath, log)
	})
}

func reportSteps(dbpath string, log *zap.Logger) error {
	conn, err := openReadOnly(dbpath)
	if err != nil {
		return err
	}
	defer conn.Close()

	values, err := identifiers(conn)
	if err != nil {
		return err
	}

	steps, err := allSteps(conn)
	if err != nil {
		return err
	}

	if len(steps) == 0 {
		log.Info("No sync steps", zap.String("path", dbpath), zap.Strings("identifiers", values))
		return nil
	}

	log.Info("Sync steps", zap.String("path", dbpath), zap.Strings("identifiers", values), zap.Int("count", len(steps)))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Step\tCreated\tSource\tDestination\tObjects")
	for _, s := range steps {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n",
			s.StepID,
			s.Created.Format(time.DateTime),
			s.Source,
			s.Destination,
			s.ObjectCount,
		)
	}
	w.Flush()
	fmt.Println()
	return nil
}

// RunObjects lists all objects in the latest (or specified) step.
func RunObjects(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)
	stepID := cmd.Int64("step")

	return forEachDB(env.Cfg.HistoryPath, cmd.String("db"), log, func(dbpath string) error {
		return reportObjects(dbpath, stepID, log)
	})
}

func reportObjects(dbpath string, requestedStep int64, log *zap.Logger) error {
	conn, err := openReadOnly(dbpath)
	if err != nil {
		return err
	}
	defer conn.Close()

	values, err := identifiers(conn)
	if err != nil {
		return err
	}

	stepID := requestedStep
	if stepID == 0 {
		stepID, err = lastStep(conn)
		if err != nil {
			return err
		}
	}
	if stepID == 0 {
		log.Info("No sync steps", zap.String("path", dbpath), zap.Strings("identifiers", values))
		return nil
	}

	ois, err := stepObjects(conn, stepID)
	if err != nil {
		return err
	}

	log.Info("Objects", zap.String("path", dbpath), zap.Strings("identifiers", values),
		zap.Int64("step", stepID), zap.Int("count", len(ois)))

	if len(ois) == 0 {
		return nil
	}

	keys := sortedKeys(ois)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Path\tSize\tModified\tHash")
	for _, k := range keys {
		oi := ois[k]
		hash := oi.PersistentID
		if len(hash) > 12 {
			hash = hash[:12] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			k,
			formatSize(oi.ObjSize),
			oi.Modified.Format(time.DateTime),
			hash,
		)
	}
	w.Flush()
	fmt.Println()
	return nil
}

// RunDiff shows changes between two steps (defaults to last two).
func RunDiff(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)
	fromStep := cmd.Int64("from")
	toStep := cmd.Int64("to")

	return forEachDB(env.Cfg.HistoryPath, cmd.String("db"), log, func(dbpath string) error {
		return reportDiff(dbpath, fromStep, toStep, log)
	})
}

func reportDiff(dbpath string, fromStep, toStep int64, log *zap.Logger) error {
	conn, err := openReadOnly(dbpath)
	if err != nil {
		return err
	}
	defer conn.Close()

	values, err := identifiers(conn)
	if err != nil {
		return err
	}

	steps, err := allSteps(conn)
	if err != nil {
		return err
	}
	if len(steps) < 2 && fromStep == 0 && toStep == 0 {
		log.Info("Not enough steps to diff", zap.String("path", dbpath), zap.Strings("identifiers", values), zap.Int("steps", len(steps)))
		return nil
	}

	// Resolve from/to defaults: last two steps
	if fromStep == 0 && toStep == 0 {
		fromStep = steps[len(steps)-2].StepID
		toStep = steps[len(steps)-1].StepID
	} else if fromStep == 0 {
		// find step before toStep
		for i, s := range steps {
			if s.StepID == toStep && i > 0 {
				fromStep = steps[i-1].StepID
				break
			}
		}
		if fromStep == 0 {
			return fmt.Errorf("cannot find step before %d", toStep)
		}
	} else if toStep == 0 {
		toStep, err = lastStep(conn)
		if err != nil {
			return err
		}
	}

	oldOIS, err := stepObjects(conn, fromStep)
	if err != nil {
		return err
	}
	newOIS, err := stepObjects(conn, toStep)
	if err != nil {
		return err
	}

	added := newOIS.Subtract(oldOIS)
	removed := oldOIS.Subtract(newOIS)
	changed := newOIS.DiffByFunc(oldOIS, func(a, b *objects.ObjectInfo) bool {
		return a.PersistentID == b.PersistentID
	})

	log.Info("Diff",
		zap.String("path", dbpath),
		zap.Strings("identifiers", values),
		zap.Int64("from", fromStep),
		zap.Int64("to", toStep),
		zap.Int("added", len(added)),
		zap.Int("removed", len(removed)),
		zap.Int("changed", len(changed)),
	)

	if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
		fmt.Println("  No changes")
		fmt.Println()
		return nil
	}

	if len(added) > 0 {
		fmt.Println("  Added:")
		for _, k := range sortedKeys(added) {
			fmt.Printf("    + %s (%s)\n", k, formatSize(added[k].ObjSize))
		}
	}
	if len(removed) > 0 {
		fmt.Println("  Removed:")
		for _, k := range sortedKeys(removed) {
			fmt.Printf("    - %s (%s)\n", k, formatSize(removed[k].ObjSize))
		}
	}
	if len(changed) > 0 {
		fmt.Println("  Changed:")
		for _, k := range sortedKeys(changed) {
			fmt.Printf("    ~ %s (%s)\n", k, formatSize(changed[k].ObjSize))
		}
	}
	fmt.Println()
	return nil
}

// RunStats shows aggregate statistics for the latest (or specified) step.
func RunStats(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)
	stepID := cmd.Int64("step")

	return forEachDB(env.Cfg.HistoryPath, cmd.String("db"), log, func(dbpath string) error {
		return reportStats(dbpath, stepID, log)
	})
}

func reportStats(dbpath string, requestedStep int64, log *zap.Logger) error {
	conn, err := openReadOnly(dbpath)
	if err != nil {
		return err
	}
	defer conn.Close()

	values, err := identifiers(conn)
	if err != nil {
		return err
	}

	stepID := requestedStep
	if stepID == 0 {
		stepID, err = lastStep(conn)
		if err != nil {
			return err
		}
	}
	if stepID == 0 {
		log.Info("No sync steps", zap.String("path", dbpath), zap.Strings("identifiers", values))
		return nil
	}

	steps, err := allSteps(conn)
	if err != nil {
		return err
	}

	ois, err := stepObjects(conn, stepID)
	if err != nil {
		return err
	}

	log.Info("Statistics", zap.String("path", dbpath), zap.Strings("identifiers", values), zap.Int64("step", stepID))

	// Compute statistics
	var (
		totalSize  int64
		fileCount  int
		dirCount   int
		oldest     time.Time
		newest     time.Time
		extCounts  = make(map[string]int)
		extSizes   = make(map[string]int64)
		thumbCount int
	)

	for _, oi := range ois {
		if oi.Dir {
			dirCount++
			continue
		}
		fileCount++
		totalSize += oi.ObjSize
		ext := strings.ToLower(filepath.Ext(oi.Name))
		extCounts[ext]++
		extSizes[ext] += oi.ObjSize
		if len(oi.ThumbName) > 0 {
			thumbCount++
		}
		if oldest.IsZero() || oi.Modified.Before(oldest) {
			oldest = oi.Modified
		}
		if newest.IsZero() || oi.Modified.After(newest) {
			newest = oi.Modified
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  Total syncs:\t%d\n", len(steps))
	fmt.Fprintf(w, "  Files:\t%d\n", fileCount)
	if dirCount > 0 {
		fmt.Fprintf(w, "  Directories:\t%d\n", dirCount)
	}
	fmt.Fprintf(w, "  Total size:\t%s\n", formatSize(totalSize))
	if thumbCount > 0 {
		fmt.Fprintf(w, "  With thumbnails:\t%d\n", thumbCount)
	}
	if !oldest.IsZero() {
		fmt.Fprintf(w, "  Oldest file:\t%s\n", oldest.Format(time.DateTime))
		fmt.Fprintf(w, "  Newest file:\t%s\n", newest.Format(time.DateTime))
	}
	w.Flush()

	if len(extCounts) > 0 {
		fmt.Println()
		fmt.Println("  By extension:")
		exts := make([]string, 0, len(extCounts))
		for ext := range extCounts {
			exts = append(exts, ext)
		}
		sort.Strings(exts)

		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "    Extension\tCount\tTotal size")
		for _, ext := range exts {
			fmt.Fprintf(w, "    %s\t%d\t%s\n", ext, extCounts[ext], formatSize(extSizes[ext]))
		}
		w.Flush()
	}
	fmt.Println()
	return nil
}

// RunOrphans finds history databases whose identifiers no longer match any connected device or existing source directory.
func RunOrphans(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	log := env.Log.Named(driverName)

	found := false
	err := forEachDB(env.Cfg.HistoryPath, cmd.String("db"), log, func(dbpath string) error {
		orphan, err := reportOrphan(dbpath, log)
		if err != nil {
			log.Error("Unable to check history", zap.String("path", dbpath), zap.Error(err))
			return nil
		}
		if orphan {
			found = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !found {
		log.Info("No orphaned history databases found")
	}
	return nil
}

func reportOrphan(dbpath string, log *zap.Logger) (bool, error) {
	conn, err := openReadOnly(dbpath)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	values, err := identifiers(conn)
	if err != nil {
		return false, err
	}

	steps, err := allSteps(conn)
	if err != nil {
		return false, err
	}

	var reasons []string

	if len(steps) == 0 {
		reasons = append(reasons, "no sync steps recorded")
	} else {
		lastStepInfo := steps[len(steps)-1]

		// Check if last sync source directory exists
		if info, err := os.Stat(lastStepInfo.Source); err != nil || !info.IsDir() {
			reasons = append(reasons, fmt.Sprintf("source directory missing: %s", lastStepInfo.Source))
		}

		// Check if last sync was more than 180 days ago
		age := time.Since(lastStepInfo.Created)
		if age > 180*24*time.Hour {
			reasons = append(reasons, fmt.Sprintf("last sync was %d days ago", int(age.Hours()/24)))
		}
	}

	if len(reasons) > 0 {
		log.Warn("Possibly orphaned",
			zap.String("path", dbpath),
			zap.Strings("identifiers", values),
			zap.Strings("reasons", reasons),
		)
		return true, nil
	}
	return false, nil
}

// helpers

const shortIDLen = 8

// shortID extracts the short identifier (first 8 hex chars) from a history database filename.
func shortID(dbpath string) string {
	name := strings.TrimSuffix(filepath.Base(dbpath), ".db")
	if len(name) > shortIDLen {
		return name[:shortIDLen]
	}
	return name
}

func forEachDB(historyPath, filter string, log *zap.Logger, fn func(dbpath string) error) error {
	entries, err := os.ReadDir(historyPath)
	if err != nil {
		return fmt.Errorf("unable to read history directory '%s': %w", historyPath, err)
	}
	matched := false
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" {
			continue
		}
		dbpath := filepath.Join(historyPath, e.Name())
		if len(filter) > 0 {
			name := strings.TrimSuffix(e.Name(), ".db")
			if !strings.HasPrefix(name, filter) {
				continue
			}
		}
		matched = true
		if err := fn(dbpath); err != nil {
			log.Error("Unable to process history", zap.String("path", dbpath), zap.Error(err))
		}
	}
	if len(filter) > 0 && !matched {
		log.Warn("No history database matched", zap.String("filter", filter))
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
