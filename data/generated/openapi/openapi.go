// Code generated for package openapi by go-bindata DO NOT EDIT. (@generated)
// sources:
// ../../openapi/managed-services-api.yaml
package openapi

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _managedServicesApiYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x58\x4b\x8f\xdb\xb6\x13\xbf\xfb\x53\x0c\xfc\xff\x1f\xda\xc3\x5a\xde\x24\x3d\x54\xb7\x4d\x9f\xe9\x2b\x45\x36\x69\x0f\x45\xb1\x18\x93\x23\x8b\x59\x89\x64\xc9\x91\x37\xde\xa2\xdf\xbd\x20\x25\x5b\x0f\xcb\x2f\xb4\x80\x37\x40\xf7\xb4\x24\x7f\xf3\xe4\xcc\x8f\x23\x1b\x4b\x1a\xad\x4a\xe1\xf9\x6c\x3e\x9b\x4f\x94\xce\x4c\x3a\x01\x60\xc5\x05\xa5\xf0\x23\x6a\x5c\x92\x84\x5b\x72\x2b\x25\x08\x6e\x7e\x7e\x35\x01\x90\xe4\x85\x53\x96\x95\xd1\xfb\x20\x2b\x72\x3e\x1e\xcf\x67\xf3\xd9\xf5\xc4\x93\x0b\x3b\x41\xf3\x15\x54\xae\x48\x21\x67\xb6\x3e\x4d\x12\xb4\x6a\x16\x7c\xf0\xb9\xca\x78\x26\x4c\x39\x01\xd8\xb1\xa0\x34\x7c\x62\x9d\x91\x95\x08\x3b\x9f\x42\xad\x6e\x5c\x99\x67\x5c\xd2\x31\x95\xb7\x8c\x4b\xa5\x97\x3b\x8a\x92\x5d\xa8\xa8\x9c\x23\xcd\x20\x4d\x89\x4a\x4f\x2c\x72\x1e\xe3\x08\xc6\x92\xb2\x0e\xfe\xca\xd7\xc1\xfb\xab\xb0\xb9\xba\x4e\xee\x31\xbb\x47\x9f\xfc\xa9\xe4\x5f\x69\x54\xb9\x24\xae\xff\x01\xf0\x55\x59\xa2\x5b\xa7\xf0\x0d\x31\x20\x44\x28\x38\xfa\xa3\x22\xcf\xb0\x58\x83\x92\x1b\x20\x89\xca\x29\x5e\x6f\x04\x83\x97\x2f\x09\x1d\xb9\x14\x7e\xfb\xbd\xd9\x74\xe4\xad\xd1\x9e\x7c\x8b\x9a\x3e\x9b\xcf\xa7\xed\x72\x10\xcf\xf7\x3d\x7b\x99\xa9\xb4\xec\x59\x0d\x7f\xc2\x68\x26\xcd\x5d\x1d\x00\x68\x6d\xa1\x04\x06\x2d\xc9\x7b\x6f\x74\xff\x14\xc0\x8b\x9c\x4a\x1c\xee\x02\xfc\xdf\x51\x96\xc2\xf4\x7f\x89\x30\xa5\x35\x9a\x34\xfb\xa4\xc6\xfa\x24\x3a\xf3\xa6\xf6\x65\xda\x06\xf0\x62\x7e\xbd\x3f\x80\x9b\x8a\x73\x60\x73\x4f\x1a\x94\x07\xa5\x57\x58\x5c\xc6\xf9\xaf\x9c\x33\xae\xe7\xf5\xf3\xfd\x5e\xbf\xd3\x58\x71\x6e\x9c\x7a\x24\x09\x6c\xc0\x92\xcb\x8c\x2b\xc1\x58\x72\xd1\xad\xa7\x11\xc1\x8b\xfd\x11\xfc\x64\x06\xb5\xfa\xa0\x38\x07\x6f\x49\xa8\x4c\x91\x04\x25\x81\x3e\x28\xcf\xfe\x29\x44\xf2\xd9\xa1\x16\x78\xa7\xe9\x83\x25\xc1\x24\x81\x82\x1c\x18\x11\xbb\xfc\xd2\x55\x64\xd1\x61\x49\xdc\x10\x25\xc4\x86\x1f\x13\x6d\x71\x89\x92\xd3\xd3\xb8\xa8\xd6\x68\x8d\xdf\xe5\xa1\x2f\x1c\x21\x13\x20\x68\x7a\x68\xae\xb8\x69\xc9\xf3\x88\x28\x8a\xbc\x34\xb2\x83\x1b\x61\x1e\x89\x8c\xdb\xf3\x20\xa4\x1c\xc9\x14\xd8\x55\x34\x39\x90\xfc\xc3\xa9\x1f\x4f\xfc\xb9\xcc\x33\xce\xa5\xcf\x0e\x50\x91\x10\x64\xf9\x32\x75\xb3\x97\x3a\x0f\x14\xfe\x2f\x81\x2a\xa3\x1b\x75\xe1\xfb\xa7\x52\xf9\xf0\x1f\xeb\x5f\x32\x82\xcf\x8f\x8d\x0b\x58\x38\x42\xb9\xfe\x58\x08\xfe\x46\x43\xb5\x8f\xe3\x41\x04\xba\x0b\xc3\x1f\xe7\xd4\x84\xd7\xa7\xbb\x4b\xc4\xd5\x9e\x04\xf1\x0d\xe1\xde\x06\xd0\x86\x8e\x1a\xc6\x6d\xb4\xf3\xda\x52\x3d\xf8\x4e\x3a\xc6\x29\x85\x45\x84\x35\x9b\xf5\xe2\x6b\xe3\x4a\xe4\x14\xbe\xfb\xf5\xed\x64\xe3\x65\xa3\xf4\xf5\xe2\x3d\x09\x7e\x43\x19\x39\xd2\x82\xfa\xda\x4d\x3c\x6c\xb6\xac\x0b\x45\xcb\xaa\xcb\x8e\x4a\x76\x83\xad\x85\x3c\x3b\xa5\x97\xdb\xed\x7b\xa5\x8f\x83\xf2\x90\xa0\x43\xa0\x1f\x54\xfb\x6e\x9d\xe8\xdb\x49\x86\x2d\x2e\x69\x17\xa4\x34\xd3\x72\x9b\x43\x00\xaf\x1e\x4f\x40\xb1\x61\x2c\x8e\xc1\xb6\x0f\x5e\xe7\x29\x0d\x9e\x76\x96\xc1\xa7\xce\x32\x18\xef\x2c\xa3\x95\xce\x5a\x31\x95\x75\x43\xc6\x4a\xda\xe8\xc5\xa2\x78\x9d\x75\x8d\x1c\xaa\xc1\x41\x11\x4c\xbb\xe6\x76\x93\xbd\x2f\xe1\x10\xdb\x46\xd2\xb0\xfe\x47\x13\x5f\xe7\x02\x47\x9a\x68\x2f\x7c\xcb\x99\x77\xfd\xb2\x1b\x15\x8a\xc9\xe8\x56\xcd\x59\x09\x09\x82\xff\x20\x0b\xf1\x4e\xc6\x5d\x44\xe7\x70\x3d\x38\x19\x85\x9f\xcc\x86\xdd\x39\xe0\xd2\xb7\x5f\x54\x9e\xc9\xbd\xfa\xf2\xe4\x3b\xf5\x8c\x5c\xed\x49\xd5\x08\x5c\x14\xa6\x92\x77\xd6\x99\x95\x92\x2d\x15\x1e\x15\x2b\xab\x82\xd5\x1d\x3e\x8e\x0b\x2c\x8c\x29\x08\xf5\xa0\x34\x97\xea\x9c\xd2\x7c\xd0\x67\xb8\xa3\xb1\x3c\xbd\x49\x16\xc6\xb0\x67\x87\xf6\x36\xfe\x54\xf1\x6d\x67\x80\x3f\x9e\xae\x38\xd8\xcb\x3b\x3c\x5d\x04\x20\x6b\x1e\x0b\x89\x4c\x57\xac\x4a\xea\x9d\x57\x56\xfe\x5b\x2a\xbb\x85\xfb\x71\x77\xea\xc0\xe3\xf6\xef\xb0\xef\xc3\x09\x7e\xf8\xf9\xd7\xd2\x5c\xac\x98\xf6\x27\x1a\xa5\x53\xb0\xc8\x79\xb3\xec\x8d\x3e\x6f\x73\x0a\x5f\xe2\x26\x03\x47\xc2\x38\x39\x7c\x77\xba\x1f\x5a\xc3\x79\xa5\x77\x85\x7f\x07\x00\x00\xff\xff\xa1\xc6\x50\xf7\x19\x14\x00\x00")

func managedServicesApiYamlBytes() ([]byte, error) {
	return bindataRead(
		_managedServicesApiYaml,
		"managed-services-api.yaml",
	)
}

func managedServicesApiYaml() (*asset, error) {
	bytes, err := managedServicesApiYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "managed-services-api.yaml", size: 5145, mode: os.FileMode(420), modTime: time.Unix(1600883381, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"managed-services-api.yaml": managedServicesApiYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"managed-services-api.yaml": &bintree{managedServicesApiYaml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
