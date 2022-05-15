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
		if fname, err := filepath.Rel(filename, path); err == nil {
			path = filepath.Join(linkDirName, fname)
		} else {
			return err
		}

		if err == nil && info.Mode()&os.ModeSymlink == os.ModeSymlink {
			finalPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			info, err := os.Lstat(finalPath)
			if err == nil {
				return err
			}
			if info.IsDir() {
				return walk(finalPath, path, fn)
			}
		}

		if err != nil {
			return fn(path, info, err)
		}
		if info != nil && !info.IsDir() {
			return fn(path, info, err)
		}

		return nil
	}

	return filepath.Walk(filename, symWalkFunc)
}
