package files_test

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/files"
	"github.com/onsi/gomega"
	"github.com/rs/xid"
)

func Test_WalkSymLinks(t *testing.T) {
	g := gomega.NewWithT(t)

	root, err := createSymLinkedDirectory([]source{
		{name: "foo.txt", content: xid.New().String()},
		{name: "bar.txt", content: xid.New().String()},
	})

	g.Expect(err).To(gomega.BeNil())

	defer func() {
		_ = os.RemoveAll(root)
	}()

	results := make([]string, 2)
	err = files.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		results = append(results, filepath.Base(path))

		return nil
	})

	g.Expect(err).To(gomega.BeNil())
	g.Expect(results).To(gomega.ContainElements("foo.txt", "bar.txt"))
}

type source struct {
	name    string
	content string
}

// this function re-creates what kubernetes does when mounting a volume from a
// configmap where the actual files are double-symlinked from some random named
// path:
//
//   drwxrwxrwx. 3 root root 88 Jul  7 13:18 .
//   drwxr-xr-x. 3 root root 53 Jun  8 12:12 ..
//   drwxr-xr-x. 2 root root 35 Jul  7 13:18 ..2020_07_07_13_18_32.149716995
//   lrwxrwxrwx. 1 root root 31 Jul  7 13:18 ..data -> ..2020_07_07_13_18_32.149716995
//   lrwxrwxrwx. 1 root root 28 Jul  7 13:18 aws-sqs-source-v1alpha1.json -> ..data/aws-sqs-source-v1alpha1.json
func createSymLinkedDirectory(sources []source) (string, error) {
	dir, err := ioutil.TempDir("", "fn-")
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

	for _, source := range sources {
		dst := path.Join(contentDir, source.name)

		err = ioutil.WriteFile(dst, []byte(source.content), 0644)
		if err != nil {
			return "", err
		}

		err = os.Symlink(path.Join(dataDir, source.name), path.Join(catalogDir, source.name))
		if err != nil {
			return "", err
		}
	}

	return dir, nil
}
