package files

import (
	"os"
	"path/filepath"
)

// Walk walks the file tree rooted at root including symlinks, calling fn for
// each file in the tree.
func Walk(root string, fn filepath.WalkFunc) error {
	return walk(root, "", fn)
}

func walk(filename string, linkDirName string, fn filepath.WalkFunc) error {
	if linkDirName == "" {
		linkDirName = filename
	}

	symWalkFunc := func(path string, info os.FileInfo, err error) error {
		relName, relErr := filepath.Rel(filename, path)
		if relErr != nil {
			return relErr
		}

		path = filepath.Join(linkDirName, relName)
		if err != nil {
			return fn(path, info, err)
		}

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			finalPath, fileErr := filepath.EvalSymlinks(path)
			if fileErr != nil {
				return fileErr
			}

			info, err = os.Lstat(finalPath)
			if err != nil {
				return fn(path, info, err)
			}

			if info.IsDir() {
				return walk(finalPath, path, fn)
			}

			path = finalPath
		}

		if info != nil && !info.IsDir() {
			return fn(path, info, err)
		}

		return nil
	}

	return filepath.Walk(filename, symWalkFunc)
}
