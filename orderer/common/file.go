package common

import (
	"os"
	"path/filepath"
)

// RemoveAllFilesAndDirs: remove a directory and its subdirectories and files
// params:
// - dirPath: the path of directories of files that need to be removed altogether
// return:
// - error
func RemoveAllFilesAndDirs(dirPath string) error {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip the directory itself
		if path == dirPath {
			return nil
		}

		// delete files or directory
		return os.RemoveAll(path)
	})

	if err != nil {
		return err
	}

	// delete the directory itself
	return os.RemoveAll(dirPath)
}
