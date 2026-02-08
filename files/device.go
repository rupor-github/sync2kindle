package files

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"go.uber.org/zap"

	"s2k/common"
	"s2k/config"
	"s2k/objects"
	"s2k/thumbs"
)

// should be usable in the zap log.Named()
const driverName = "file-system"

type Device struct {
	log   *zap.Logger
	roots []string
	mount string
	tmbs  *config.ThumbnailsConfig
}

func Connect(paths, mount string, tmbs *config.ThumbnailsConfig, log *zap.Logger) (*Device, error) {
	if len(paths) == 0 {
		return nil, common.ErrNoFiles
	}

	d := &Device{mount: mount, tmbs: tmbs, log: log.Named(driverName)}

	ps := filepath.SplitList(paths)
	for _, p := range ps {
		base := filepath.ToSlash(p)
		if !slices.Contains(d.roots, base) {
			if len(mount) > 0 {
				base = path.Join(mount, base)
			}
			d.roots = append(d.roots, base)
		}
	}
	return d, nil
}

// driver interface

func (d *Device) Disconnect() {
	// nothing to do at the moment
}

func (d *Device) Name() string {
	return driverName
}

func (d *Device) UniqueID() string {
	return driverName
}

func (d *Device) MkDir(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("MkDir is called with nil object")
	}
	if len(d.mount) > 0 {
		obj.FullPath = path.Join(d.mount, obj.FullPath)
	}

	defer func(start time.Time) {
		d.log.Debug("Executed action MkDir", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	return os.Mkdir(obj.FullPath, 0755)
}

func (d *Device) Remove(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Remove is called with nil object")
	}
	if len(d.mount) > 0 {
		obj.FullPath = path.Join(d.mount, obj.FullPath)
	}

	defer func(start time.Time) {
		d.log.Debug("Executed action Remove", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	return os.Remove(obj.FullPath)
}

func (d *Device) Copy(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Copy is called with nil object")
	}

	if len(d.mount) > 0 {
		obj.FullPath = path.Join(d.mount, obj.FullPath)
	}

	defer func(start time.Time) {
		d.log.Debug("Executed action Copy", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	from, err := os.Open(obj.ObjectName)
	if err != nil {
		return fmt.Errorf("unable to open source file '%s': %w", obj.ObjectName, err)
	}
	defer from.Close()

	to, err := os.OpenFile(obj.FullPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("unable to create destination file '%s': %w", obj.FullPath, err)
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			d.log.Warn("Unable to close destination file", zap.String("path", obj.FullPath), zap.Error(err))
		}
	}(to)

	// our files are typically quite small...
	written, err := io.CopyBuffer(to, from, make([]byte, 256*1024))
	if err != nil {
		return fmt.Errorf("failed to copy file '%s' to '%s': %w", obj.ObjectName, obj.FullPath, err)
	}
	if written != obj.ObjSize {
		return fmt.Errorf("failed to Copy file '%s' (%d) to '%s' (%d), not all bytes have been written", obj.ObjectName, obj.ObjSize, obj.FullPath, written)
	}
	return nil
}

func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {

	// To get the same behavior for different connection protocols (MTP, USB, files) we will check source path here, rather than on Connect()
	// NOTE: for source path it should never happen since configuration is validated

	// to speed up hashing
	var buf []byte

	oset := objects.New()
	for _, root := range d.roots {
		if _, err := os.Stat(root); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, err // inaccessible
			}
			continue // does not exist
		}
		if len(buf) == 0 {
			buf = make([]byte, 256*1024)
		}
		if err := filepath.Walk(root, func(next string, info os.FileInfo, err error) error {
			if err != nil {
				d.log.Warn("Skipping path during file enumeration", zap.String("path", next), zap.Error(err))
				return nil
			}
			if info.Mode().IsRegular() || info.IsDir() {
				key := next
				if len(d.mount) > 0 {
					key, _ = filepath.Rel(d.mount, next)
				}
				key = filepath.ToSlash(key)
				if _, exists := oset[key]; exists {
					d.log.Warn("Duplicate path during file enumeration, ignoring", zap.String("path", key))
					return nil
				}
				o := &objects.ObjectInfo{
					Name:     info.Name(),
					Dir:      info.IsDir(),
					Modified: info.ModTime(),
					ObjSize:  info.Size(),
					FullPath: key,
					File:     info.Mode().IsRegular(),
				}
				if !info.IsDir() {
					hash, err := hashFileContent(next, buf)
					if err != nil {
						return fmt.Errorf("unable to hash file content for '%s': %w", next, err)
					}
					o.PersistentID = hash
					if d.tmbs != nil {
						// see if file needs thumb extraction
						if name := thumbs.ExtractThumbnail(key, d.tmbs, d.log); len(name) > 0 {
							o.ThumbName = name
						}
					}
				}
				oset[key] = o
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("unable to enumerate files in '%s': %w", root, err)
		}
	}
	return oset, nil
}

// implementation

func hashFileContent(path string, buf []byte) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.CopyBuffer(h, file, buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
