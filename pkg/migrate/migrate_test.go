package migrate

import (
	"testing"
	"testing/fstest"
)

func TestList_OrderAndFilter(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/0002_knowledge.up.sql":  {Data: []byte("x")},
		"migrations/0001_init.up.sql":       {Data: []byte("x")},
		"migrations/0001_init.down.sql":     {Data: []byte("x")}, // must be ignored
		"migrations/0003_fare_cache.up.sql": {Data: []byte("x")},
		"migrations/readme.txt":             {Data: []byte("x")}, // ignored
	}
	got, err := list(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 up-migrations, got %d (%+v)", len(got), got)
	}
	want := []string{"0001_init", "0002_knowledge", "0003_fare_cache"}
	for i, w := range want {
		if got[i].version != w {
			t.Fatalf("order[%d] = %s, want %s", i, got[i].version, w)
		}
	}
}
