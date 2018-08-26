package surf

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestFilesystemCache(t *testing.T) {
	dir, err := ioutil.TempDir("", "surf_cache_fs_test_")
	if err != nil {
		t.Fatalf("cannot create temporary directory: %s", err)
	}
	t.Logf("temporary cache directory created: %s", dir)
	defer os.RemoveAll(dir)

	cache := NewFilesystemCache(dir)
	RunCacheImplementationTest(t, cache)
}
