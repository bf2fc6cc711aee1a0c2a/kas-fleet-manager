package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/rs/xid"
	"os"
	"path"
)

// this function re-creates what kubernetes does when mounting a volume from a
// configmap where the actual files are double-symlinked from some random named
// path:
//
//	drwxrwxrwx. 3 root root 88 Jul  7 13:18 .
//	drwxr-xr-x. 3 root root 53 Jun  8 12:12 ..
//	drwxr-xr-x. 2 root root 35 Jul  7 13:18 ..2020_07_07_13_18_32.149716995
//	lrwxrwxrwx. 1 root root 31 Jul  7 13:18 ..data -> ..2020_07_07_13_18_32.149716995
//	lrwxrwxrwx. 1 root root 28 Jul  7 13:18 aws-sqs-source-v1alpha1.json -> ..data/aws-sqs-source-v1alpha1.json
func createSymLinkedCatalogDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", prefix+"-catalog-")
	if err != nil {
		return "", err
	}

	contentDir := path.Join(dir, xid.New().String())
	if err := os.Mkdir(contentDir, 0755); err != nil {
		return "", err
	}
	catalogDir := path.Join(dir, "catalog")
	if err := os.Mkdir(catalogDir, 0755); err != nil {
		return "", err
	}

	dataDir := path.Join(dir, "data")
	if err := os.Symlink(contentDir, dataDir); err != nil {
		return "", err
	}

	source := "./internal/connector/test/integration/resources/" + prefix + "-catalog"
	source = shared.BuildFullFilePath(source)

	dirEntries, err := os.ReadDir(source)
	if err != nil {
		return "", err
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		src := path.Join(source, dirEntry.Name())
		dst := path.Join(contentDir, dirEntry.Name())

		data, err := os.ReadFile(src)
		if err != nil {
			return "", err
		}
		err = os.WriteFile(dst, data, 0644)
		if err != nil {
			return "", err
		}

		err = os.Symlink(path.Join(dataDir, dirEntry.Name()), path.Join(catalogDir, dirEntry.Name()))
		if err != nil {
			return "", err
		}
	}

	return dir, nil
}
