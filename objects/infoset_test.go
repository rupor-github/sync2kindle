package objects

import (
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"time"
)

// helper: sorted keys from an ObjectInfoSet.
func sortedKeys(s ObjectInfoSet) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Clone
// ---------------------------------------------------------------------------

func TestClone(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		s := New()
		c := s.Clone()
		if len(c) != 0 {
			t.Fatalf("expected empty clone, got %d items", len(c))
		}
	})

	t.Run("cloned set has same keys and values", func(t *testing.T) {
		info := &ObjectInfo{Name: "a.azw3", File: true, FullPath: "D:/a.azw3"}
		s := ObjectInfoSet{"D:/a.azw3": info}
		c := s.Clone()
		if len(c) != 1 {
			t.Fatalf("expected 1 item in clone, got %d", len(c))
		}
		if c["D:/a.azw3"] != info {
			t.Fatal("cloned value should point to same ObjectInfo")
		}
	})

	t.Run("mutation of clone does not affect original", func(t *testing.T) {
		s := ObjectInfoSet{
			"D:/a.azw3": &ObjectInfo{Name: "a.azw3", File: true, FullPath: "D:/a.azw3"},
			"D:/b.azw3": &ObjectInfo{Name: "b.azw3", File: true, FullPath: "D:/b.azw3"},
		}
		c := s.Clone()
		delete(c, "D:/a.azw3")
		if len(s) != 2 {
			t.Fatal("deleting from clone should not affect original")
		}
	})
}

// ---------------------------------------------------------------------------
// Find
// ---------------------------------------------------------------------------

func TestFind(t *testing.T) {
	info := &ObjectInfo{Name: "a.azw3", File: true, FullPath: "D:/test/a.azw3"}
	s := ObjectInfoSet{"D:/test/a.azw3": info}

	tests := []struct {
		name     string
		path     string
		wantNil  bool
		wantName string
	}{
		{"existing key", "D:/test/a.azw3", false, "a.azw3"},
		{"missing key", "D:/test/b.azw3", true, ""},
		{"empty string", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.Find(tt.path)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
			} else {
				if got == nil {
					t.Fatal("expected non-nil result")
				}
				if got.Name != tt.wantName {
					t.Fatalf("expected Name=%q, got %q", tt.wantName, got.Name)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Add / Delete
// ---------------------------------------------------------------------------

func TestAddDelete(t *testing.T) {
	t.Run("add and find", func(t *testing.T) {
		s := New()
		info := &ObjectInfo{Name: "a.azw3", File: true, FullPath: "D:/test/a.azw3"}
		s.Add("D:/test/a.azw3", info)
		if s.Find("D:/test/a.azw3") == nil {
			t.Fatal("added item should be findable")
		}
	})

	t.Run("add empty path is noop", func(t *testing.T) {
		s := New()
		s.Add("", &ObjectInfo{Name: "x"})
		if len(s) != 0 {
			t.Fatal("adding empty path should not modify set")
		}
	})

	t.Run("delete existing", func(t *testing.T) {
		s := ObjectInfoSet{
			"D:/a.azw3": &ObjectInfo{Name: "a.azw3"},
			"D:/b.azw3": &ObjectInfo{Name: "b.azw3"},
		}
		s.Delete("D:/a.azw3")
		if s.Find("D:/a.azw3") != nil {
			t.Fatal("deleted item should not be findable")
		}
		if len(s) != 1 {
			t.Fatalf("expected 1 remaining, got %d", len(s))
		}
	})

	t.Run("delete empty path is noop", func(t *testing.T) {
		s := ObjectInfoSet{"D:/a.azw3": &ObjectInfo{Name: "a.azw3"}}
		s.Delete("")
		if len(s) != 1 {
			t.Fatal("deleting empty path should not modify set")
		}
	})

	t.Run("delete missing key is noop", func(t *testing.T) {
		s := ObjectInfoSet{"D:/a.azw3": &ObjectInfo{Name: "a.azw3"}}
		s.Delete("D:/no-such.azw3")
		if len(s) != 1 {
			t.Fatal("deleting missing key should not modify set")
		}
	})

	t.Run("add overwrites existing", func(t *testing.T) {
		s := New()
		s.Add("D:/test/a.azw3", &ObjectInfo{Name: "old"})
		newInfo := &ObjectInfo{Name: "new"}
		s.Add("D:/test/a.azw3", newInfo)
		if got := s.Find("D:/test/a.azw3"); got != newInfo {
			t.Fatal("add should overwrite existing entry")
		}
	})
}

// ---------------------------------------------------------------------------
// SubsetByFunc
// ---------------------------------------------------------------------------

func TestSubsetByFunc(t *testing.T) {
	s := ObjectInfoSet{
		"D:/test/a.azw3": &ObjectInfo{Name: "a.azw3", File: true, FullPath: "D:/test/a.azw3"},
		"D:/test/b.mobi": &ObjectInfo{Name: "b.mobi", File: true, FullPath: "D:/test/b.mobi"},
		"D:/test/c.pdf":  &ObjectInfo{Name: "c.pdf", File: true, FullPath: "D:/test/c.pdf"},
		"D:/test/d":      &ObjectInfo{Name: "d", Dir: true, FullPath: "D:/test/d"},
	}

	t.Run("filter files only", func(t *testing.T) {
		result := s.SubsetByFunc(func(k string, v *ObjectInfo) bool {
			return v.File
		})
		if len(result) != 3 {
			t.Fatalf("expected 3 files, got %d", len(result))
		}
	})

	t.Run("filter by extension", func(t *testing.T) {
		result := s.SubsetByFunc(func(k string, v *ObjectInfo) bool {
			return filepath.Ext(v.Name) == ".azw3"
		})
		if len(result) != 1 {
			t.Fatalf("expected 1 .azw3 file, got %d", len(result))
		}
	})

	t.Run("filter returns empty for no matches", func(t *testing.T) {
		result := s.SubsetByFunc(func(k string, v *ObjectInfo) bool {
			return false
		})
		if len(result) != 0 {
			t.Fatalf("expected 0, got %d", len(result))
		}
	})

	t.Run("filter returns all for always-true", func(t *testing.T) {
		result := s.SubsetByFunc(func(k string, v *ObjectInfo) bool {
			return true
		})
		if len(result) != len(s) {
			t.Fatalf("expected %d, got %d", len(s), len(result))
		}
	})
}

// ---------------------------------------------------------------------------
// Subtract
// ---------------------------------------------------------------------------

func TestSubtract(t *testing.T) {
	tests := []struct {
		name     string
		a, b     ObjectInfoSet
		wantKeys []string
	}{
		{
			name:     "disjoint sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			b:        ObjectInfoSet{"z": &ObjectInfo{Name: "z"}},
			wantKeys: []string{"x", "y"},
		},
		{
			name:     "overlapping sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			b:        ObjectInfoSet{"y": &ObjectInfo{Name: "y"}},
			wantKeys: []string{"x"},
		},
		{
			name:     "identical sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}},
			b:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}},
			wantKeys: []string{},
		},
		{
			name:     "subtract empty",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}},
			b:        New(),
			wantKeys: []string{"x"},
		},
		{
			name:     "from empty",
			a:        New(),
			b:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}},
			wantKeys: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Subtract(tt.b)
			got := sortedKeys(result)
			sort.Strings(tt.wantKeys)
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("expected keys %v, got %v", tt.wantKeys, got)
			}
			for i := range got {
				if got[i] != tt.wantKeys[i] {
					t.Fatalf("expected keys %v, got %v", tt.wantKeys, got)
				}
			}
		})
	}

	t.Run("returns values from receiver", func(t *testing.T) {
		infoA := &ObjectInfo{Name: "x", PersistentID: "aaa"}
		a := ObjectInfoSet{"x": infoA}
		b := ObjectInfoSet{"y": &ObjectInfo{Name: "y"}}
		result := a.Subtract(b)
		if result["x"] != infoA {
			t.Fatal("should return pointer from receiver set")
		}
	})
}

// ---------------------------------------------------------------------------
// Intersect
// ---------------------------------------------------------------------------

func TestIntersect(t *testing.T) {
	tests := []struct {
		name     string
		a, b     ObjectInfoSet
		wantKeys []string
	}{
		{
			name:     "disjoint sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			b:        ObjectInfoSet{"z": &ObjectInfo{Name: "z"}},
			wantKeys: []string{},
		},
		{
			name:     "overlapping sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			b:        ObjectInfoSet{"y": &ObjectInfo{Name: "y"}, "z": &ObjectInfo{Name: "z"}},
			wantKeys: []string{"y"},
		},
		{
			name:     "identical sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			b:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			wantKeys: []string{"x", "y"},
		},
		{
			name:     "one empty",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}},
			b:        New(),
			wantKeys: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Intersect(tt.b)
			got := sortedKeys(result)
			sort.Strings(tt.wantKeys)
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("expected keys %v, got %v", tt.wantKeys, got)
			}
			for i := range got {
				if got[i] != tt.wantKeys[i] {
					t.Fatalf("expected keys %v, got %v", tt.wantKeys, got)
				}
			}
		})
	}

	t.Run("returns values from receiver", func(t *testing.T) {
		infoA := &ObjectInfo{Name: "x", PersistentID: "aaa"}
		infoB := &ObjectInfo{Name: "x", PersistentID: "bbb"}
		a := ObjectInfoSet{"x": infoA}
		b := ObjectInfoSet{"x": infoB}
		result := a.Intersect(b)
		if result["x"] != infoA {
			t.Fatal("should return pointer from receiver set, not other")
		}
	})
}

// ---------------------------------------------------------------------------
// Union
// ---------------------------------------------------------------------------

func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		a, b     ObjectInfoSet
		wantKeys []string
	}{
		{
			name:     "disjoint sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}},
			b:        ObjectInfoSet{"y": &ObjectInfo{Name: "y"}},
			wantKeys: []string{"x", "y"},
		},
		{
			name:     "overlapping sets",
			a:        ObjectInfoSet{"x": &ObjectInfo{Name: "x"}, "y": &ObjectInfo{Name: "y"}},
			b:        ObjectInfoSet{"y": &ObjectInfo{Name: "yB"}, "z": &ObjectInfo{Name: "z"}},
			wantKeys: []string{"x", "y", "z"},
		},
		{
			name:     "both empty",
			a:        New(),
			b:        New(),
			wantKeys: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Union(tt.b)
			got := sortedKeys(result)
			sort.Strings(tt.wantKeys)
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("expected keys %v, got %v", tt.wantKeys, got)
			}
			for i := range got {
				if got[i] != tt.wantKeys[i] {
					t.Fatalf("expected keys %v, got %v", tt.wantKeys, got)
				}
			}
		})
	}

	t.Run("other set wins on key collision", func(t *testing.T) {
		infoA := &ObjectInfo{Name: "from_a"}
		infoB := &ObjectInfo{Name: "from_b"}
		a := ObjectInfoSet{"x": infoA}
		b := ObjectInfoSet{"x": infoB}
		result := a.Union(b)
		if result["x"] != infoB {
			t.Fatal("for overlapping keys, other set should win (last writer wins)")
		}
	})
}

// ---------------------------------------------------------------------------
// DiffByFunc
// ---------------------------------------------------------------------------

func TestDiffByFunc(t *testing.T) {
	now := time.Now()

	t.Run("returns entries present in both but not equal", func(t *testing.T) {
		a := ObjectInfoSet{
			"book.azw3": &ObjectInfo{Name: "book.azw3", PersistentID: "new-hash", Modified: now},
		}
		b := ObjectInfoSet{
			"book.azw3": &ObjectInfo{Name: "book.azw3", PersistentID: "old-hash", Modified: now},
		}
		result := a.DiffByFunc(b, func(x, y *ObjectInfo) bool {
			return x.PersistentID == y.PersistentID
		})
		if len(result) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(result))
		}
	})

	t.Run("returns empty when all equal", func(t *testing.T) {
		a := ObjectInfoSet{
			"book.azw3": &ObjectInfo{Name: "book.azw3", PersistentID: "same"},
		}
		b := ObjectInfoSet{
			"book.azw3": &ObjectInfo{Name: "book.azw3", PersistentID: "same"},
		}
		result := a.DiffByFunc(b, func(x, y *ObjectInfo) bool {
			return x.PersistentID == y.PersistentID
		})
		if len(result) != 0 {
			t.Fatalf("expected 0 diffs, got %d", len(result))
		}
	})

	t.Run("ignores keys only in one set", func(t *testing.T) {
		a := ObjectInfoSet{
			"only-in-a.azw3": &ObjectInfo{Name: "only-in-a.azw3", PersistentID: "x"},
		}
		b := ObjectInfoSet{
			"only-in-b.azw3": &ObjectInfo{Name: "only-in-b.azw3", PersistentID: "y"},
		}
		result := a.DiffByFunc(b, func(x, y *ObjectInfo) bool {
			return x.PersistentID == y.PersistentID
		})
		if len(result) != 0 {
			t.Fatalf("expected 0 (disjoint sets have no diffs), got %d", len(result))
		}
	})

	t.Run("returns values from receiver", func(t *testing.T) {
		infoA := &ObjectInfo{Name: "book.azw3", PersistentID: "new"}
		a := ObjectInfoSet{"book.azw3": infoA}
		b := ObjectInfoSet{"book.azw3": &ObjectInfo{Name: "book.azw3", PersistentID: "old"}}
		result := a.DiffByFunc(b, func(x, y *ObjectInfo) bool {
			return x.PersistentID == y.PersistentID
		})
		if result["book.azw3"] != infoA {
			t.Fatal("should return pointer from receiver set")
		}
	})
}

// ---------------------------------------------------------------------------
// SubsetByPath (original test refactored to t.Run + fix #50 prefix collision)
// ---------------------------------------------------------------------------

func TestSubsetByPath(t *testing.T) {
	s := ObjectInfoSet{
		"D:/test/01.azw3":    &ObjectInfo{FullPath: "D:/test/01.azw3", Name: "01.azw3", File: true},
		"D:/test/02.azw3":    &ObjectInfo{FullPath: "D:/test/02.azw3", Name: "02.azw3", File: true},
		"D:/test/01":         &ObjectInfo{FullPath: "D:/test/01", Name: "01", Dir: true},
		"D:/test/01/01.azw3": &ObjectInfo{FullPath: "D:/test/01/01.azw3", Name: "01.azw3", File: true},
		"D:/test/01/02.azw3": &ObjectInfo{FullPath: "D:/test/01/02.azw3", Name: "02.azw3", File: true},
		"D:/test/02":         &ObjectInfo{FullPath: "D:/test/02", Name: "02", Dir: true},
		"D:/test/02/01.azw3": &ObjectInfo{FullPath: "D:/test/02/01.azw3", Name: "01.azw3", File: true},
		"D:/test/02/02.azw3": &ObjectInfo{FullPath: "D:/test/02/02.azw3", Name: "02.azw3", File: true},
	}

	t.Run("full set from root", func(t *testing.T) {
		subset := s.SubsetByPath("D:/test")
		if len(subset) != 8 {
			t.Fatalf("expected 8, got %d: %v", len(subset), sortedKeys(subset))
		}
	})

	t.Run("filter by extension after subset", func(t *testing.T) {
		subset := s.SubsetByPath("D:/test").
			SubsetByFunc(func(k string, v *ObjectInfo) bool {
				return !v.Dir && slices.Contains([]string{".azw3", ".mobi", ".kfx"}, filepath.Ext(v.FullPath))
			})
		if len(subset) != 6 {
			t.Fatalf("expected 6 book files, got %d", len(subset))
		}
	})

	t.Run("subdirectory subset", func(t *testing.T) {
		subset := s.SubsetByPath("D:/test/01")
		if len(subset) != 2 {
			t.Fatalf("expected 2, got %d: %v", len(subset), sortedKeys(subset))
		}
	})

	t.Run("empty dir returns full set", func(t *testing.T) {
		subset := s.SubsetByPath("")
		if len(subset) != len(s) {
			t.Fatalf("empty dir should return full set, got %d", len(subset))
		}
	})

	t.Run("non-existent path returns empty", func(t *testing.T) {
		subset := s.SubsetByPath("D:/nonexistent")
		if len(subset) != 0 {
			t.Fatalf("expected 0, got %d", len(subset))
		}
	})

	// Fix #50: prefix collision test — "D:/test" must not match "D:/test2"
	t.Run("prefix collision", func(t *testing.T) {
		setWithCollision := ObjectInfoSet{
			"D:/test":           &ObjectInfo{FullPath: "D:/test", Name: "test", Dir: true},
			"D:/test/a.azw3":    &ObjectInfo{FullPath: "D:/test/a.azw3", Name: "a.azw3", File: true},
			"D:/test2":          &ObjectInfo{FullPath: "D:/test2", Name: "test2", Dir: true},
			"D:/test2/b.azw3":   &ObjectInfo{FullPath: "D:/test2/b.azw3", Name: "b.azw3", File: true},
			"D:/testing":        &ObjectInfo{FullPath: "D:/testing", Name: "testing", Dir: true},
			"D:/testing/c.azw3": &ObjectInfo{FullPath: "D:/testing/c.azw3", Name: "c.azw3", File: true},
		}

		subset := setWithCollision.SubsetByPath("D:/test")
		keys := sortedKeys(subset)
		// Should only contain items under D:/test, NOT D:/test2 or D:/testing
		for _, k := range keys {
			if k == "D:/test2" || k == "D:/test2/b.azw3" || k == "D:/testing" || k == "D:/testing/c.azw3" {
				t.Fatalf("prefix collision: %q should not appear in subset of D:/test", k)
			}
		}
		// D:/test is the dir itself (excluded by SubsetByPath when base == dir)
		// So we expect only "a.azw3"
		if len(subset) != 1 {
			t.Fatalf("expected 1 item (a.azw3), got %d: %v", len(subset), keys)
		}
	})

	// Additional prefix collision test with subdirectory structure
	t.Run("prefix collision with subdirs", func(t *testing.T) {
		setWithCollision := ObjectInfoSet{
			"documents/test":            &ObjectInfo{FullPath: "documents/test", Name: "test", Dir: true},
			"documents/test/a.azw3":     &ObjectInfo{FullPath: "documents/test/a.azw3", Name: "a.azw3", File: true},
			"documents/test/sub":        &ObjectInfo{FullPath: "documents/test/sub", Name: "sub", Dir: true},
			"documents/test/sub/b.azw3": &ObjectInfo{FullPath: "documents/test/sub/b.azw3", Name: "b.azw3", File: true},
			"documents/test2":           &ObjectInfo{FullPath: "documents/test2", Name: "test2", Dir: true},
			"documents/test2/c.azw3":    &ObjectInfo{FullPath: "documents/test2/c.azw3", Name: "c.azw3", File: true},
		}

		subset := setWithCollision.SubsetByPath("documents/test")
		keys := sortedKeys(subset)
		for _, k := range keys {
			if k == "documents/test2" || k == "documents/test2/c.azw3" {
				t.Fatalf("prefix collision: %q should not appear in subset of documents/test", k)
			}
		}
		// Expect: a.azw3, sub (dir), sub/b.azw3 = 3 items
		if len(subset) != 3 {
			t.Fatalf("expected 3 items, got %d: %v", len(subset), keys)
		}
	})
}

// ---------------------------------------------------------------------------
// SubsetByPath key rewriting
// ---------------------------------------------------------------------------

func TestSubsetByPathKeyRewriting(t *testing.T) {
	s := ObjectInfoSet{
		"D:/root/sub/a.azw3": &ObjectInfo{FullPath: "D:/root/sub/a.azw3", Name: "a.azw3", File: true},
		"D:/root/sub":        &ObjectInfo{FullPath: "D:/root/sub", Name: "sub", Dir: true},
	}

	subset := s.SubsetByPath("D:/root")
	// Keys should have "D:/root/" prefix stripped
	if _, ok := subset["sub/a.azw3"]; !ok {
		t.Fatalf("expected key 'sub/a.azw3', got keys: %v", sortedKeys(subset))
	}
	if _, ok := subset["sub"]; !ok {
		t.Fatalf("expected key 'sub', got keys: %v", sortedKeys(subset))
	}
}

// ---------------------------------------------------------------------------
// Compositional: Subtract + Intersect identity
// ---------------------------------------------------------------------------

func TestSetAlgebra(t *testing.T) {
	a := ObjectInfoSet{
		"x": &ObjectInfo{Name: "x"},
		"y": &ObjectInfo{Name: "y"},
		"z": &ObjectInfo{Name: "z"},
	}
	b := ObjectInfoSet{
		"y": &ObjectInfo{Name: "y"},
		"z": &ObjectInfo{Name: "z"},
		"w": &ObjectInfo{Name: "w"},
	}

	t.Run("|A-B| + |A∩B| = |A|", func(t *testing.T) {
		diff := a.Subtract(b)
		inter := a.Intersect(b)
		if len(diff)+len(inter) != len(a) {
			t.Fatalf("expected %d = %d + %d", len(a), len(diff), len(inter))
		}
	})

	t.Run("|A∪B| = |A| + |B| - |A∩B|", func(t *testing.T) {
		union := a.Union(b)
		inter := a.Intersect(b)
		if len(union) != len(a)+len(b)-len(inter) {
			t.Fatalf("expected %d, got %d", len(a)+len(b)-len(inter), len(union))
		}
	})
}
