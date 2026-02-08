// Package sync provides a simple sync algorithm for syncing a local (directory) and
// remote (device) file system. The algorithm is based on the following rules:
//
// # | L H D | Cause                   | Operation --- result --->   | L H D
// --+------+-------------------------+------------------------------+------
// 1 | - - - | nothing                 | ignore                      | - - -
// 2 | - - + | manually added (D)      | ignore                      | - - +
// 3 | + - - | manually added (L)      | add to device (sync)        | + + +
// 4 | + - + | error (H)               | ignore                      | + + +
// 5 | - + - | manually removed (L,D)  | ignore                      | - + -
// 6 | - + + | manually removed (L)    | remove from device (sync)   | - - -
// 7 | + + - | manually removed (D)    | remove from local (sync)    | - - -
// 8 | + + + | nothing                 | ignore                      | + + +
//
// where L is local, H is history DB, and D is device.
//
// Additional caveat are books which have been synced to device and then changed locally (updated)
// This is possibly case #8 and we specifically handle it as part of case #3
//
// There is a special case (CLI switch "ignore-device-removals") which makes sync fully one
// directional (ignore case #7) - from local to device, otherwise by default we try to handle most
// useful day to day usage scenario.
package sync

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"

	"s2k/common"
	"s2k/config"
	"s2k/objects"
)

type action func(bool, *zap.Logger) error

type driver interface {
	Name() string
	UniqueID() string
	MkDir(*objects.ObjectInfo) error
	Remove(*objects.ObjectInfo) error
	Copy(*objects.ObjectInfo) error
	GetObjectInfos() (objects.ObjectInfoSet, error)
	Disconnect()
}

func PrepareActions(srcActor, dstActor, hstActor driver, cfg *config.Config, ignoreDeviceRemovals, email bool, logParent *zap.Logger) ([]action, objects.ObjectInfoSet, error) {
	log := logParent.Named("prepare")

	// Local file system

	start := time.Now()
	srcOIS, err := srcActor.GetObjectInfos()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get source files: %w", err)
	}
	log.Debug("Local artifacts (all)", zap.Duration("elapsed", time.Since(start)), zap.Int("count", len(srcOIS)), zap.Any("Infos", srcOIS))

	localBooks := srcOIS.
		SubsetByPath(cfg.SourcePath).
		SubsetByFunc(func(k string, v *objects.ObjectInfo) bool {
			return !v.Dir && slices.Contains(cfg.BookExtensions, filepath.Ext(v.Name))
		})
	if len(localBooks) == 0 {
		return nil, nil, fmt.Errorf("no books in the source path: %w", common.ErrNoFiles)
	}
	log.Debug("Local artifacts (filtered)", zap.Int("count", len(localBooks)), zap.Any("Infos", localBooks))

	// history

	start = time.Now()
	hstOIS, err := hstActor.GetObjectInfos()
	if err != nil {
		return nil, nil, fmt.Errorf("history objects cannot be read: %w", err)
	}
	log.Debug("History artifacts (all)", zap.Duration("elapsed", time.Since(start)), zap.Int("count", len(hstOIS)), zap.Any("Infos", hstOIS))

	historyBooks := hstOIS.
		SubsetByFunc(func(k string, v *objects.ObjectInfo) bool {
			return !v.Dir && slices.Contains(cfg.BookExtensions, filepath.Ext(v.Name))
		})
	log.Debug("History artifacts (filtered)", zap.Int("count", len(historyBooks)), zap.Any("Infos", historyBooks))

	// target device

	start = time.Now()
	dstOIS, err := dstActor.GetObjectInfos()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get files on the device: %w", err)
	}
	if email {
		// e-mail driver always returns empty set
		dstOIS = hstOIS.Clone()
	}
	log.Debug("Device artifacts (all)", zap.Duration("elapsed", time.Since(start)), zap.Int("count", len(dstOIS)), zap.Any("Infos", dstOIS))

	targetExists := dstOIS.Find(cfg.TargetPath) != nil
	thumbsAvailable := dstOIS.Find(common.ThumbnailFolder) != nil

	deviceBooks := dstOIS
	if targetExists {
		deviceBooks = dstOIS.
			SubsetByPath(cfg.TargetPath).
			SubsetByFunc(func(k string, v *objects.ObjectInfo) bool {
				return !v.Dir && slices.Contains(cfg.BookExtensions, filepath.Ext(v.Name))
			})
		log.Debug("Device artifacts (filtered)", zap.Int("count", len(deviceBooks)), zap.Any("Infos", deviceBooks))
	}

	log.Debug("Device state", zap.Bool("destination exists", targetExists), zap.Bool("thumbnails available", thumbsAvailable))

	var deviceThumbs objects.ObjectInfoSet
	if thumbsAvailable {
		var dirs = []string{}
		deviceThumbs = dstOIS.
			SubsetByPath(common.ThumbnailFolder).
			SubsetByFunc(func(k string, v *objects.ObjectInfo) bool {
				// loose all subdirectories
				if v.Dir {
					dirs = append(dirs, k)
					return false
				}
				return true
			}).
			SubsetByFunc(func(k string, v *objects.ObjectInfo) bool {
				// loose all thumbs in subdirectories too - do not know what to do with them
				for _, d := range dirs {
					if strings.HasPrefix(k, d) {
						return false
					}
				}
				return true
			}).
			SubsetByFunc(func(k string, v *objects.ObjectInfo) bool {
				return slices.Contains(cfg.ThumbExtensions, filepath.Ext(v.Name))
			})
		log.Debug("Device thumbnails (filtered)", zap.Int("count", len(deviceThumbs)), zap.Any("Infos", deviceThumbs))
	}

	// Analyze the situation and prepare actions

	var actions []action

	// case #7 ----------------------------------------------------------------
	// books were manually removed from device since last sync

	objs := historyBooks.Subtract(deviceBooks).Intersect(localBooks)
	if len(objs) > 0 && !ignoreDeviceRemovals && !email {
		log.Debug("Removed from device", zap.Int("count", len(objs)), zap.Any("Infos", objs))
		for _, obj := range objs {
			actions = makeRemoveActions(actions, obj, cfg.SourcePath, srcOIS, srcActor, log)
			for _, p := range getSupplementalArtifactsPaths(obj.FullPath) {
				if sobj := srcOIS.Find(p); sobj != nil {
					actions = makeRemoveActions(actions, sobj, cfg.SourcePath, srcOIS, srcActor, log)
				}
			}
			if thumbsAvailable && len(obj.ThumbName) > 0 {
				thumb := deviceThumbs.Find(obj.ThumbName)
				if thumb != nil {
					actions = append(actions, makeAction(dstActor, "Remove", thumb, log))
					dstOIS.Delete(thumb.FullPath)
					deviceThumbs.Delete(obj.ThumbName)
				}
			}
		}
		localBooks = localBooks.Subtract(objs)
	}

	// case #6 ----------------------------------------------------------------
	// books were manually removed from local storage since last sync

	objs = deviceBooks.Subtract(localBooks).Intersect(historyBooks)
	if len(objs) > 0 && !email {
		log.Debug("Removed locally", zap.Int("count", len(objs)), zap.Any("Infos", objs))

		// Kindle has a habit of creating additional directories and files, leave them untouched, only
		// remove files we are aware of, try not to touch anything else.
		for key, obj := range objs {
			actions = append(actions, makeAction(dstActor, "Remove", obj, log))
			dstOIS.Delete(obj.FullPath)
			for _, p := range getSupplementalArtifactsPaths(obj.FullPath) {
				if sobj := dstOIS.Find(p); sobj != nil {
					actions = append(actions, makeAction(dstActor, "Remove", sobj, log))
					dstOIS.Delete(sobj.FullPath)
				}
			}
			if thumbsAvailable {
				hobj := historyBooks.Find(key)
				if hobj != nil && len(hobj.ThumbName) > 0 {
					thumb := deviceThumbs.Find(hobj.ThumbName)
					if thumb != nil {
						actions = append(actions, makeAction(dstActor, "Remove", thumb, log))
						dstOIS.Delete(thumb.FullPath)
						deviceThumbs.Delete(hobj.ThumbName)
					}
				}
			}
			// NOTE: we do rely on one's ability to remove all local artifacts, so -
			// local leftovers in this case are not considered here
		}
		deviceBooks = deviceBooks.Subtract(objs)
	}

	// case #3 ----------------------------------------------------------------
	// books were manually added to local storage or have been changed locally since last sync

	changedLocalBooks := localBooks.DiffByFunc(historyBooks, func(a, b *objects.ObjectInfo) bool {
		return a.Dir || a.PersistentID == b.PersistentID
	})
	if len(changedLocalBooks) > 0 {
		log.Debug("Local artifacts (changed)", zap.Int("count", len(changedLocalBooks)), zap.Any("Infos", changedLocalBooks))
	}

	objs = localBooks.Subtract(deviceBooks).Union(changedLocalBooks)
	if len(objs) > 0 {
		log.Debug("Added or changed locally", zap.Int("count", len(objs)), zap.Any("Infos", objs))

		for _, obj := range objs {
			actions = makeCopyActions(actions, obj, cfg.SourcePath, cfg.TargetPath, dstOIS, dstActor, email, log)

			if email {
				continue // no thumbnails or page indexes for e-mail
			}

			for _, p := range getSupplementalArtifactsPaths(obj.FullPath) {
				if sobj := srcOIS.Find(p); sobj != nil {
					actions = makeCopyActions(actions, sobj, cfg.SourcePath, cfg.TargetPath, dstOIS, dstActor, false, log)
				}
			}
			if thumbsAvailable && len(obj.ThumbName) > 0 {
				from := path.Join(cfg.Thumbnails.Dir, obj.ThumbName)   // old path, where to copy from
				to := path.Join(common.ThumbnailFolder, obj.ThumbName) // new path, where to copy to

				fi, err := os.Stat(from)
				if err != nil {
					log.Debug("Unable to stat thumbnail, skipping", zap.Any("Info", obj))
					continue
				}

				oldThumb := deviceThumbs.Find(obj.ThumbName)
				if oldThumb != nil {
					actions = append(actions, makeAction(dstActor, "Remove", oldThumb, log))
					dstOIS.Delete(oldThumb.FullPath)
					deviceThumbs.Delete(oldThumb.ThumbName)
				}
				thumb := &objects.ObjectInfo{
					Name:       obj.ThumbName,
					File:       true,
					Modified:   obj.Modified,
					ObjSize:    fi.Size(),
					FullPath:   to,
					ObjectName: from,
					OIS:        dstOIS,
				}
				actions = append(actions, makeAction(dstActor, "Copy", thumb, log))
				deviceThumbs.Add(to, thumb)
				dstOIS.Add(to, thumb)
			}
		}
	}
	return actions, srcOIS, nil
}

// getSupplementalArtifactsPaths returns a list of names of some additional artifacts (page index files and such)
// for the given book. Kindle book could have page index file (same name as a book with extension .apnx)
// in the same directory as book itself or in .sdr subdirectory of the same directory as book itself.
func getSupplementalArtifactsPaths(fullPath string) []string {
	dir, file := path.Split(fullPath)

	dir = strings.TrimSuffix(dir, "/")
	ext := path.Ext(file)
	base := strings.TrimSuffix(file, ext)

	return []string{
		path.Join(dir, base+".apnx"),
		path.Join(dir, base+".sdr", base+".apnx"),
	}
}

func makeAction(actor driver, action string, obj *objects.ObjectInfo, log *zap.Logger) action {
	if obj == nil {
		panic("making action with nil object")
	}

	v := reflect.ValueOf(actor)
	method := v.MethodByName(action)
	if !method.IsValid() {
		panic("making action driver method not found")
	}

	subjectName := "file"
	if obj.Dir {
		subjectName = "directory"
	}

	log.Debug("Making action",
		zap.String("action", action), zap.String("actor", actor.Name()), zap.String("subject", subjectName), zap.String("object", obj.FullPath))

	return func(dryRun bool, log *zap.Logger) error {
		log.Named(actor.Name()).Info("Executing", zap.String("action", action), zap.String(subjectName, obj.FullPath))

		if dryRun {
			return nil
		}

		res := method.Call([]reflect.Value{reflect.ValueOf(obj)})
		if len(res) > 0 && !res[0].IsNil() {
			return res[0].Interface().(error)
		}
		return nil
	}
}

// makeRemoveActions creates actions to remove the given "obj" and all empty directories above it all the way to the "rootSrc" (not inclusive).
func makeRemoveActions(actions []action, obj *objects.ObjectInfo, rootSrc string, src objects.ObjectInfoSet, actor driver, log *zap.Logger) []action {
	actions = append(actions, makeAction(actor, "Remove", obj, log))
	src.Delete(obj.FullPath)

	dir := path.Dir(obj.FullPath)
	if dir == "." {
		return actions
	}
	return makeRemoveDirActions(actions, dir, rootSrc, src, actor, log)
}

// makeRemoveDirActions creates actions to recursively remove empty directories from the given "dir" (not relative), all
// the way up to the "root" (not inclusive).
func makeRemoveDirActions(actions []action, dir, root string, src objects.ObjectInfoSet, actor driver, log *zap.Logger) []action {
	if dir == root {
		return actions
	}

	left := src.SubsetByPath(dir)
	if len(left) > 0 {
		return actions
	}
	obj := src.Find(dir)
	if obj != nil {
		actions = append(actions, makeAction(actor, "Remove", src.Find(dir), log))
		src.Delete(dir)
	}
	return makeRemoveDirActions(actions, filepath.ToSlash(filepath.Dir(dir)), root, src, actor, log)
}

// makeCopyActions creates actions to copy files from the source "obj.FullPath" to the device, making
// sure that all necessary "parent" folders on the device are created first. Part of the source path relative to
// "rootSrc" will be created on the device relative to "rootDst" if necessary.
func makeCopyActions(actions []action, obj *objects.ObjectInfo, rootSrc, rootDst string, dst objects.ObjectInfoSet, actor driver, email bool, log *zap.Logger) []action {
	var dstPath string
	if !email {
		// we need to re-root every path from source to destination
		relPath := strings.TrimPrefix(obj.FullPath, rootSrc+"/")
		actions = makeCreateDirActions(actions, path.Dir(relPath), rootDst, dst, actor, log)
		dstPath = path.Join(rootDst, relPath)

		// If we do not remove files on device before copying updates Windows Explorer gets really confused.
		if prevObj := dst.Find(dstPath); prevObj != nil && !prevObj.Dir {
			actions = append(actions, makeAction(actor, "Remove", prevObj, log))
			dst.Delete(dstPath)
		}

	} else {
		// there is no need to re-root anything for e-mail, this is only used for diagnostics
		dstPath = strings.TrimPrefix(obj.FullPath, rootSrc+"/")
	}

	o := &objects.ObjectInfo{
		Name:         obj.Name,
		PersistentID: obj.PersistentID,
		File:         true,
		Modified:     obj.Modified,
		ObjSize:      obj.ObjSize,
		FullPath:     dstPath,      // new path, where to copy to
		ObjectName:   obj.FullPath, // original path, where to copy from
		OIS:          dst,
	}
	dst.Add(dstPath, o)
	actions = append(actions, makeAction(actor, "Copy", o, log))

	return actions
}

// makeCreateDirActions make actions to create folders on the device, always starting from "root" (inclusive) to the last
// element of "dir" (always relative).
func makeCreateDirActions(actions []action, dir, root string, dst objects.ObjectInfoSet, actor driver, log *zap.Logger) []action {
	head, parts := root, []string{}
	if dir != "." {
		parts = strings.Split(dir, "/")
	}
	for i := 0; ; i++ {
		if dst.Find(head) == nil {
			obj := &objects.ObjectInfo{
				Name:     path.Base(head),
				Dir:      true,
				Modified: time.Now(),
				FullPath: head,
				OIS:      dst,
			}
			dst.Add(head, obj)
			actions = append(actions, makeAction(actor, "MkDir", obj, log))
		}
		if i == len(parts) {
			break
		}
		head = path.Join(head, parts[i])
	}
	return actions
}
