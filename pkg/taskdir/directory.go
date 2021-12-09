package taskdir

import (
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type TaskDirectory struct {
	// rootPath is the absolute path to the task's root directory.
	rootPath string
	// path is the absolute path of the airplane.yml task definition.
	defPath string
	// closer is used to clean up TaskDirectory.
	closer io.Closer
}

// New creates a TaskDirectory struct with the (desired) definition file as input
func New(file string) (TaskDirectory, error) {
	var td TaskDirectory
	var err error
	td.defPath, err = filepath.Abs(file)
	if err != nil {
		return td, errors.Wrap(err, "converting local file path to absolute path")
	}
	// For a new defPath, assume the root is the directory of the defPath
	td.rootPath = filepath.Dir(td.defPath)
	return td, nil
}

// Open creates a TaskDirectory struct from a file argument
// Supports file in the form of github.com/path/to/repo/example and will download from GitHub
// Supports file in the form of local_file.yml and will read it to determine the full details
func Open(file string, use_0_3 bool) (TaskDirectory, error) {
	if strings.HasPrefix(file, "http://") {
		return TaskDirectory{}, errors.New("http:// paths are not supported, use https:// instead")
	}

	var td TaskDirectory
	var err error
	if strings.HasPrefix(file, "github.com/") || strings.HasPrefix(file, "https://github.com/") {
		td.defPath, td.closer, err = openGitHubDirectory(file)
		if err != nil {
			return TaskDirectory{}, err
		}
	} else {
		td.defPath, err = filepath.Abs(file)
		if err != nil {
			return TaskDirectory{}, errors.Wrap(err, "converting local file path to absolute path")
		}
	}

	if !use_0_3 {
		def, err := td.ReadDefinition()
		if err != nil {
			return TaskDirectory{}, err
		}
		td.rootPath = path.Join(filepath.Dir(td.defPath), def.Root)
	} else {
		def, err := td.ReadDefinition_0_3()
		if err != nil {
			return TaskDirectory{}, err
		}
		root, err := def.Root(filepath.Dir(td.defPath))
		if err != nil {
			return TaskDirectory{}, err
		}
		td.rootPath = root
	}

	if !strings.HasPrefix(td.defPath, td.rootPath+string(filepath.Separator)) {
		return TaskDirectory{}, errors.Errorf("%s must be inside of the task's root directory: %s", path.Base(td.defPath), td.rootPath)
	}

	return td, nil
}

func (td TaskDirectory) DefinitionPath() string {
	return td.defPath
}

func (td TaskDirectory) DefinitionRootPath() string {
	return td.rootPath
}

func (td TaskDirectory) Close() error {
	if td.closer != nil {
		return td.closer.Close()
	}

	return nil
}
