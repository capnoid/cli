package conf

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("read missing", func(t *testing.T) {
		var assert = require.New(t)
		var homedir = tempdir(t)
		var path = filepath.Join(homedir, ".airplane", "config")

		_, err := Read(path)

		assert.Error(err)
		assert.True(errors.Is(err, ErrMissing))
	})

	t.Run("write missing dir", func(t *testing.T) {
		var assert = require.New(t)
		var homedir = tempdir(t)
		var path = filepath.Join(homedir, ".airplane", "config")

		err := Write(path, Config{
			Token: "foo",
		})
		assert.NoError(err)

		cfg, err := Read(path)
		assert.NoError(err)
		assert.Equal("foo", cfg.Token)
	})

	t.Run("overwrite", func(t *testing.T) {
		var assert = require.New(t)
		var homedir = tempdir(t)
		var path = filepath.Join(homedir, ".airplane", "config")

		{
			err := Write(path, Config{
				Token: "foo",
			})
			assert.NoError(err)

			cfg, err := Read(path)
			assert.NoError(err)
			assert.Equal("foo", cfg.Token)
		}

		{
			err := Write(path, Config{
				Token: "baz",
			})
			assert.NoError(err)

			cfg, err := Read(path)
			assert.NoError(err)
			assert.Equal("baz", cfg.Token)
		}
	})
}

func tempdir(t testing.TB) string {
	t.Helper()

	name, err := ioutil.TempDir("", "cli_test")
	if err != nil {
		t.Fatalf("tempdir: %s", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(name)
	})

	return name
}