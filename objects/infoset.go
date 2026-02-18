package objects

import (
	"maps"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type ObjectInfo struct {
	Name         string    `json:"file_name"`
	PersistentID string    `json:"persistent_id,omitempty"`
	Dir          bool      `json:"is_dir"`
	File         bool      `json:"is_file"`
	Modified     time.Time `json:"modified"`
	ObjSize      int64     `json:"size"`
	FullPath     string    `json:"full_path"`

	// this part is only used by MTP driver.
	Oid        ObjectID `json:"oid,omitempty"`
	OidParent  ObjectID `json:"oidParent,omitempty"`
	ObjectName string   `json:"object_name,omitempty"`
	Deletable  bool     `json:"isDeletable,omitempty"`

	// if we are copying non personal document (EBOK, not PDOC) we will try to
	// extract thumbnail from it and copy it to device. We will also need to remember
	// this and attempt proper cleanup when book is removed. This is used by local
	// file system and history drivers.
	ThumbName string `json:"thumb_name,omitempty"`

	// this part is needed by actions which create objects on MTP devices
	// at the time when action is being created we do not know actual object properties
	// including parent object id, creation of parent may be requested by another action...
	// for use by MTP driver.
	OIS ObjectInfoSet `json:"-"`
}

type ObjectInfoSet map[string]*ObjectInfo

func New() ObjectInfoSet {
	return make(ObjectInfoSet)
}

func (s ObjectInfoSet) Clone() ObjectInfoSet {
	return maps.Clone(s)
}

func (s ObjectInfoSet) Find(fullPath string) *ObjectInfo {
	if len(fullPath) == 0 {
		return nil
	}
	fullPath = filepath.ToSlash(fullPath)
	if info, exists := s[fullPath]; exists {
		return info
	}
	return nil
}

func (s ObjectInfoSet) Add(fullPath string, fi *ObjectInfo) {
	if len(fullPath) != 0 {
		fullPath = filepath.ToSlash(fullPath)
		s[fullPath] = fi
	}
}

func (s ObjectInfoSet) Delete(fullPath string) {
	if len(fullPath) != 0 {
		fullPath = filepath.ToSlash(fullPath)
		delete(s, fullPath)
	}
}

func (s ObjectInfoSet) SubsetByFunc(f func(key string, fi *ObjectInfo) bool) ObjectInfoSet {
	nos := make(ObjectInfoSet)
	for k, v := range s {
		if f(k, v) {
			nos[k] = v
		}
	}
	return nos
}

func (s ObjectInfoSet) SubsetByPath(dir string) ObjectInfoSet {
	if len(dir) == 0 {
		return s
	}
	dir = filepath.ToSlash(dir)
	nos := make(ObjectInfoSet)
	for k, v := range s {
		base := k
		if !v.Dir {
			base = path.Dir(k)
			if base == "." {
				continue
			}
		} else if base == dir {
			continue
		}
		if strings.HasPrefix(base, dir+"/") || base == dir {
			nos[strings.TrimPrefix(k, dir+"/")] = v
		}
	}
	return nos
}

// DiffByFunc returns a new ObjectInfoSet that contains only the elements that are present
// in s and in other set, but are different (equal returns false).
// NOTE: same key in both sets could point to different values, values from s are returned in new set.
func (s ObjectInfoSet) DiffByFunc(other ObjectInfoSet, equal func(a, b *ObjectInfo) bool) ObjectInfoSet {
	nos := make(ObjectInfoSet)
	for k := range s {
		if _, exists := other[k]; exists {
			if !equal(s[k], other[k]) {
				nos[k] = s[k]
			}
		}
	}
	return nos
}

// Subtract returns a new ObjectInfoSet that contains only the elements that are present in s but not in other.
// This is a set difference operation on keys: (s - other).
// NOTE: same key in both sets could point to different values, values from s are returned in new set.
func (s ObjectInfoSet) Subtract(other ObjectInfoSet) ObjectInfoSet {
	nos := make(ObjectInfoSet)
	for k, v := range s {
		if _, exists := other[k]; !exists {
			nos[k] = v
		}
	}
	return nos
}

// Intersect returns a new ObjectInfoSet that contains only the elements that are present in both s and other.
// This is a set intersection operation on keys: (s ∩ other).
// NOTE: same key in both sets could point to different values, values from s are returned in new set.
func (s ObjectInfoSet) Intersect(other ObjectInfoSet) ObjectInfoSet {
	nos := make(ObjectInfoSet)
	for k := range s {
		if _, exists := other[k]; exists {
			nos[k] = s[k]
		}
	}
	return nos
}

// Union returns a new ObjectInfoSet that contains all the elements that are present in either s or other.
func (s ObjectInfoSet) Union(other ObjectInfoSet) ObjectInfoSet {
	nos := make(ObjectInfoSet)
	maps.Copy(nos, s)
	maps.Copy(nos, other)
	return nos
}
