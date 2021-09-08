// Code generated by go-bindata. (@generated) DO NOT EDIT.

//Package generated generated by go-bindata.// sources:
// .generate/openapi/fleet-manager.yaml
package generated

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
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
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

// ModTime return file modify time
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

var _fleetManagerYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5d\x7b\x53\x1b\xb9\xb2\xff\x9f\x4f\xd1\xd7\xb9\xa7\x7c\xce\xbd\xd8\x8c\x1f\x3c\xe2\xba\xb9\x55\x24\x90\x2c\xbb\x09\x49\x78\x6c\x36\x7b\x6a\xcb\xc8\x33\xb2\x2d\x98\x91\x06\x49\x03\x38\xe7\x9e\xef\x7e\x4b\xd2\xbc\x47\xf6\xd8\x40\x16\xd8\xc0\xd6\x56\xc0\xd6\xa3\xf5\x53\xab\xbb\xd5\xdd\x92\x58\x88\x29\x0a\xc9\x00\x7a\x6d\xa7\xed\xc0\x0b\xa0\x18\x7b\x20\xa7\x44\x00\x12\x30\x26\x5c\x48\xf0\x09\xc5\x20\x19\x20\xdf\x67\xd7\x20\x58\x80\xe1\x60\x6f\x5f\xa8\x8f\x2e\x28\xbb\x36\xa5\x55\x05\x0a\x71\x73\xe0\x31\x37\x0a\x30\x95\xed\xb5\x17\xb0\xeb\xfb\x80\xa9\x17\x32\x42\xa5\x00\x0f\x8f\x09\xc5\x1e\x4c\x31\xc7\x70\x4d\x7c\x1f\x46\x18\x3c\x22\x5c\x76\x85\x39\x1a\xf9\x18\x46\x33\xd5\x13\x44\x02\x73\xd1\x86\x83\x31\x48\x5d\x56\x75\x10\x53\xc7\xe0\x02\xe3\xd0\x50\x92\xb5\xdc\x08\x39\xb9\x42\x12\x37\xd6\x01\x79\x6a\x0c\x38\x50\x45\xe5\x14\x43\x23\x40\x14\x4d\xb0\xd7\x12\x98\x5f\x11\x17\x8b\x16\x0a\x49\x2b\x2e\xdf\x9e\xa1\xc0\x6f\xc0\x98\xf8\x78\x8d\xd0\x31\x1b\xac\x01\x48\x22\x7d\x3c\x80\x3d\x42\x99\x40\x11\x87\x63\x53\x0f\xde\xfa\x18\x4b\xf8\xa0\x5b\xe3\x6b\x00\x57\x98\x0b\xc2\xe8\x00\x3a\xed\x4e\xbb\xbb\x06\xe0\x61\xe1\x72\x12\x4a\xfd\xe1\xe2\xea\x66\x44\x47\x58\x48\xd8\xfd\x74\xa0\x48\x35\x54\x66\xd5\x08\x15\x12\x51\x17\x8b\xf6\x9a\x22\x1c\x73\xa1\x68\x6b\x41\xc4\xfd\x01\x4c\xa5\x0c\xc5\x60\x63\x03\x85\xa4\xad\x60\x17\x53\x32\x96\x6d\x97\x05\x6b\x00\x25\x3a\x3e\x20\x42\xe1\xef\x21\x67\x5e\xe4\xaa\x4f\xfe\x01\xa6\x39\x7b\x63\x42\xa2\x09\xae\x6b\xf2\x58\xa2\x09\xa1\x13\x6b\x43\x83\x8d\x0d\x9f\xb9\xc8\x9f\x32\x21\x07\x3b\x8e\xe3\x54\xab\xa7\xdf\x67\x35\x37\xaa\xa5\xdc\x88\x73\x4c\x25\x78\x2c\x40\x84\xae\x85\x48\x4e\x35\x02\x8a\xcc\x0d\x2f\x46\x49\x0c\x83\x49\x20\x37\xae\x3a\x03\xdd\xc0\x04\x4b\xf3\x0b\x28\x66\xe4\x48\xb5\x74\xe0\x0d\xd4\xe7\xbf\x9a\xc9\xfa\x80\x25\xf2\x90\x44\x71\x29\x8e\x45\xc8\xa8\xc0\x22\xa9\x06\xd0\xe8\x3a\x4e\x23\xfb\x13\xc0\x65\x54\x62\x2a\xf3\x1f\x01\xa0\x30\xf4\x89\xab\x3b\xd8\x38\x17\x8c\x16\xbf\x05\x10\xee\x14\x07\xa8\xfc\x29\xc0\x7f\x72\x3c\x1e\x40\xf3\xc5\x86\xcb\x82\x90\x51\x4c\xa5\xd8\x30\x65\xc5\x46\x89\xc4\x66\xae\x72\x01\x99\xb8\x1c\x04\xc5\xb1\x88\x28\x08\x10\x9f\x0d\xe0\x08\xcb\x88\x53\xa1\x99\xff\xaa\x5a\xd6\x0e\xe0\x86\x90\x48\x46\xa2\x16\xc7\x98\x9b\x8f\x75\xe9\xc7\x88\x62\x81\xc0\xb9\x18\x7e\xbc\xc8\x48\xdd\x2c\x91\x5a\x28\x78\x4a\xf1\x4d\x88\x5d\x89\x3d\xc0\x9c\x33\x0e\xcc\xd5\x9c\xe9\x3d\xc4\xd8\xf6\x15\x05\xcd\x52\x15\x7c\x83\x82\xd0\xcf\x83\x9f\xfc\x6c\x3a\xce\xbe\xf9\xb2\xfa\x9d\xbd\xa3\xa4\xad\x8d\xac\x6a\x73\x11\x7b\x19\xa6\x01\x36\x56\x3c\xc0\x22\xee\x62\xb1\x0e\x22\x72\xa7\x4a\x81\x5c\x4f\xb1\x92\xde\x10\xa0\x1b\x12\x44\x01\xc4\xf2\x17\x5c\x14\x22\x97\xc8\x19\x4c\x91\x80\x11\xc6\x14\x38\x46\xee\x34\x85\x54\x60\x37\xe2\x44\xce\x32\xa2\x5b\xf0\x1a\x23\x8e\xf9\x00\xfe\xf9\xc7\x7c\x0e\x4e\x3f\xd9\xf8\x17\xf1\xfe\x5d\xcb\xc9\x89\xa4\x7d\x3d\x3b\xf0\x1e\x23\x23\x27\xf4\x1d\xe1\xcb\x08\x0b\xb9\xfc\xb4\x97\x2a\xbe\xc3\xf2\x28\x1e\xd7\x6d\xb9\xa1\xd4\x62\x89\x33\x96\xe9\xf9\x0b\x91\xd3\xb7\x88\xf8\xd8\x7b\xc3\xb1\xc6\xc9\x2c\xd0\x7b\xa2\x68\x41\xd3\x73\x45\x40\xaa\x69\xb9\x69\x03\xc6\x2c\xa2\x9e\xb2\x40\x0e\xf6\xb2\xe9\xef\x3b\x9d\x87\x99\xfe\x15\xd7\x7a\xdf\xe9\xdc\x16\xcb\xac\xea\x5c\xac\x76\x23\x39\x05\xc9\x2e\x30\x55\x86\x0b\xa1\x57\xc8\x27\x5e\x1e\xa4\xde\x13\x01\xa9\x77\x7b\x90\x7a\x75\x20\x9d\x0a\xcc\x81\x32\x09\x28\x92\x53\xc6\xc9\x37\x63\xae\x22\xd7\xc5\x22\x16\x97\x46\x02\xe6\x81\xeb\x3f\x11\xe0\xfa\xb7\x07\xae\x5f\x07\xdc\x21\xab\x2e\xc6\x6b\x22\xa7\x20\x42\xec\x92\x31\xc1\x1e\x1c\xec\x01\xbe\x21\x42\x8a\xf9\x6a\xfb\xb1\x62\x77\xaf\x5a\xb8\x82\x5d\x9d\x7d\x52\xab\x4c\xc1\xa6\xdb\x51\x75\x42\x32\xb9\xe8\x61\x1f\x4b\x6c\x55\xab\xe6\x2b\x8b\x66\x0d\x11\x47\x01\x96\xf1\x2e\x26\x21\x84\xd0\x01\x5c\x46\x98\xcf\x72\xa3\xa3\x28\xc0\x03\x40\x62\x46\xdd\x79\x63\xfe\x84\xf9\x98\xf1\x40\x2f\x29\xa4\x37\x35\x40\xa8\xda\x81\xea\x5a\x53\xce\x28\x8b\x84\xda\x50\x51\xbd\x3b\x59\x34\xd7\x72\x16\xe2\x01\x8c\x18\xf3\x31\xa2\xb9\x6f\xd4\xa8\x09\xc7\xde\x00\x24\x8f\xf0\x42\xf3\xa0\xfb\xf8\xb8\xb0\xdc\xd2\x8b\x43\x06\x6f\x0c\x61\x73\xb5\xa1\x9e\xb9\x82\x4c\x7f\x1a\xcb\xab\xef\x38\x9a\x76\xc2\xe8\xed\x45\x54\xb9\x89\xf9\x7b\x2f\xa5\xf8\xf4\x78\xcd\x72\x13\xd5\xfd\xc0\xb3\xc9\xf0\x6c\x32\x3c\x9b\x0c\xca\x64\x30\x32\xe5\x0e\x86\x43\xa1\x81\x1f\xd7\x7c\xb8\x1b\x8e\xe5\x06\x6e\x6f\x4a\x24\x56\x82\x69\xae\xc6\x4a\x58\xca\xf4\x08\x91\x74\xa7\x83\x72\xfb\xa7\xa1\x87\x24\xce\x37\x9f\x38\x43\x55\xfb\x64\x25\xd3\xa6\x60\x9e\x44\xba\x61\xeb\xc6\x5f\x93\xff\x9a\x79\xb9\xd6\x8a\xe0\x18\x9a\xd8\x35\xc5\x1c\xd8\x18\x12\x67\xc3\xda\x02\xfe\x59\xcc\x3d\x76\xde\x59\xc6\x23\x60\x68\xa9\xf8\x05\x56\xb0\x59\x16\x38\xbc\x52\xcc\x0d\x58\xe5\x0d\xf1\x13\x74\x80\x7c\x62\xe2\xbb\x7b\x40\x2a\xe6\x52\x01\xd3\xd7\xc8\x4b\x58\xec\x11\x48\x9c\x8a\x81\xb2\x82\xde\x7e\x50\xaa\x7b\x0b\xdc\xb4\xc2\xc4\x54\x72\xba\x54\xd4\xea\xd2\x07\x1d\x4c\x7f\xfe\x60\xf2\x0a\xcd\x38\xa5\xb4\x3a\xd3\x43\xc8\xa9\xb4\xc7\x30\x8e\x27\xe9\x3b\xaf\x6e\x46\x5b\x71\xd5\x46\xa1\x6a\x56\x6e\x83\x78\x8d\x25\x1c\xcf\xa6\xb9\x90\x09\xbb\xd3\xd9\xe5\x38\xa7\x7e\xfe\x5a\x3b\xe3\x3a\xfd\x99\x32\x74\x2e\x62\xf5\xa7\x2a\xcd\x44\x19\xa0\x99\xcf\x90\x57\x54\x26\xf3\x54\xc9\xe9\xf1\x11\x9e\x90\x2a\xff\xd5\x28\x8b\xa4\xda\x1c\x3f\xf9\xfe\xe9\xad\x5a\x4d\xaa\x55\x5a\x7d\xfc\xbe\x8a\xa7\xa4\xc9\xcb\xea\xd0\x75\x71\xf8\x54\x1d\x23\x49\x40\xe4\x0e\x8e\x91\x52\x13\xcf\x8e\x91\x67\xc7\x48\x02\xd2\x3d\x3b\x46\xd2\x66\x3f\xa0\x9b\x5d\xdf\x67\xd7\xd8\x3b\x88\xb7\x7d\x47\x26\x3e\x7c\x87\xfe\xea\xda\xb4\x12\x72\x82\x79\x20\x0e\x99\x4c\x64\xc0\x1d\xfa\x9f\xd3\xd4\x62\xc7\xd0\x98\xf1\x11\xf1\x3c\x4c\x01\x13\x1d\x49\x1f\x61\x17\x45\x02\x6b\xf5\x1e\x55\x2d\xde\xb9\xde\x23\x60\xc5\xba\x49\x44\x9e\x46\xc1\xc8\x6c\x67\xd3\x6c\x23\x90\x53\x24\xc1\x45\x14\x46\x38\x36\x58\xf4\x16\x50\xe7\x79\xe9\x3e\xcb\x51\xfb\xf6\x7c\xab\xf6\xf1\xf2\xee\xf7\x0c\x67\x9d\x4c\x71\x62\x10\x61\x2f\x4d\x8c\x00\x8f\x61\x41\x9b\xd2\x38\xa2\xf2\x98\xbd\x7c\x22\x98\xbd\x3c\x44\x01\x7e\xc3\xe8\xd8\x27\xae\xbc\x3d\x7e\xb6\x66\xe6\x0b\x4b\x85\x87\x2e\x99\xf1\x9d\x87\xa5\xd9\x54\x10\xaa\xb9\xd9\x8d\x55\x94\xe2\x63\xcd\xa6\x09\xe4\xf3\xb7\x29\x8f\x15\xe4\xef\x1b\x2b\xdc\xa5\x10\xcd\xdb\x92\xc1\xf5\x94\xf8\x09\x96\x74\xa2\x81\x2d\x7b\xf7\xe2\x76\x57\x0b\x29\x6a\x03\xc2\xea\x2c\xd4\x25\x73\x09\x3a\x96\x28\xa4\x4f\x84\x54\xd3\x5a\xae\x2a\x6c\xdb\xab\x5c\x4e\x8f\x58\x85\xd4\x95\x5d\x64\xbb\xb5\x74\xfd\xa9\x3c\x56\xb2\x68\xdf\x93\xbc\x85\xfd\x94\x1c\x53\xe6\x67\xfe\xea\x38\x30\xb6\xd2\x67\xb5\x2f\xbe\x83\x49\x6b\x69\xe6\xa9\x3b\xc7\xea\x90\xbb\x67\x8b\xf6\xc7\x35\x52\xef\x1a\xbd\x7b\x92\xfe\xb2\x65\x90\xbe\x57\xcd\x65\xf7\x89\xd9\x1a\xc9\xb9\xe8\x42\x34\xc9\x4d\x55\x6d\x71\x41\xbe\xad\x52\x9c\x71\x0f\xf3\xd7\xb3\x55\x3a\xc0\x88\xbb\xd3\xe6\x82\x8c\x6b\xc3\x1c\x43\xe4\xba\x2c\xa2\xb2\x9a\x7b\x6d\x51\x4c\xcd\xae\xe3\x34\x1f\x64\xe5\xc5\x79\xd5\xbb\x86\xd8\xa2\x92\x29\xf1\xb1\xd1\xdf\xd8\x4b\xd5\x64\xb2\x0d\x49\x46\x9a\x0d\xa7\xef\x74\x1e\x66\x38\x4f\xc8\x25\xd0\xec\x3b\xbd\x27\x02\xd2\xe3\x92\xb6\xcd\xcd\x87\x5a\x2c\x8f\xca\xec\xbf\x97\x14\x41\x89\x26\x05\x61\x9c\x54\x9a\x63\x88\x17\xa5\x85\xa8\xb7\xf0\xad\x22\x22\x1f\x49\xa9\x0f\x31\x1c\x17\x9b\xa8\x18\xb6\x7f\x42\xb0\xa1\x38\x6c\xab\xbb\x7b\x1e\x17\x88\x25\x59\x2c\x9d\x79\x6b\x5f\xb7\x0e\x0e\x3c\x16\xc5\xb2\xfc\xa2\x89\x39\x26\x9e\xed\x95\x17\x4e\xb1\xdb\xba\x35\x54\xe6\xad\xd8\x33\xf6\xac\xc8\x9e\x15\xd9\xca\x8a\xec\x7d\xad\x55\xf4\xac\xb7\xee\x4d\x6f\x59\x02\xf0\xc5\x95\xbf\x9c\x7e\xb3\xb8\xb3\x4a\xd3\xb7\x06\xd0\x5c\xd2\xd0\xd7\xe7\xd3\x9a\x15\x73\x7f\xd1\xae\xa7\x26\x31\x01\xfe\x32\x42\xdd\xd2\xcf\x4a\x82\xfc\xf5\xec\xa0\x36\xb8\x92\x59\x1f\xa5\x39\x5c\x35\x7d\xb1\xce\xee\xc9\x65\x18\x2e\xcb\x5f\xe9\xde\x69\x3e\x69\x69\xd9\x77\x58\xda\x8a\xc5\x12\xb7\x30\x64\x55\x34\x1f\x10\x4a\x8a\xa7\x89\x45\x13\x72\xa5\x84\x76\x52\x35\x7f\xc2\xe3\xbb\xf0\x65\xff\x91\x08\xb8\x85\x87\x20\x9e\xb5\xfa\x5f\x4b\xab\xdf\x01\xa4\xc7\xbe\x3d\x85\x7f\xfd\xfb\x2f\xab\xb6\x8d\x38\xba\xb3\x68\xcd\xd2\xd6\xe7\xc9\xd6\x95\x14\xf8\x06\xc7\x02\xcb\xa1\xcb\xb1\x87\xa9\x24\xc8\x17\xf1\x64\xe6\x77\xad\xcf\x3a\x5d\xe9\xf4\x96\x86\xea\x3b\x6f\xd1\x8e\x54\x1f\x90\x9b\x8e\x67\x31\xfe\x2c\xc6\x9f\xc5\xf8\xe3\x11\xe3\x5a\x08\x14\xd7\xf4\x1b\x8e\x3d\xb1\xb2\x85\x2c\xb0\x14\x49\xe6\x47\xb2\xd8\x61\xcc\xf8\x02\xc9\xfe\x42\xfd\x0f\x27\x53\x2c\x30\x20\x9e\x65\x50\xb5\xc6\xc8\x25\x74\x02\x1c\xfb\x3a\xd3\x29\xbd\x51\x2a\xae\xb3\xd4\x55\x23\x1b\x01\x96\x9c\xb8\x62\x43\x67\x6c\x0f\x39\xa2\x13\x5c\xd9\xdd\x55\x7c\x9f\x71\xa5\xd8\x04\x27\x01\x16\x98\x13\x2c\x40\x57\x37\xc9\xdf\x8a\xfc\x34\xc5\x20\xdd\x98\x94\xf7\x1c\x1f\x4c\x43\xaf\x67\x47\xaa\xe6\xe7\x5c\xd6\xf8\x6d\xd3\x1f\x2a\x3a\xc6\x1e\xde\xf9\xf9\xf8\xe3\x21\x20\xce\xd1\x0c\xd8\x18\x3e\x71\x16\x60\x39\xc5\x51\x36\x34\x36\x3a\xc7\xae\x14\x30\xe6\x2c\x00\x36\x52\x93\x83\x24\xe3\x24\x0a\x1e\x42\xce\xc4\x38\x65\x28\x15\x83\x58\x15\x2d\xf1\x1c\xf9\xaf\xaa\x94\x8a\xb0\x7b\x0e\x60\xaf\x1a\xc0\x5e\xc2\xec\x5b\xa2\xb0\x17\x19\x19\xb0\x42\x15\x42\xa5\x5a\x80\xfe\x0a\x55\xc6\xc4\x57\xff\x2e\x73\xf6\xc5\x22\x09\x57\x94\x81\x26\x07\x55\xde\x4a\xf4\x99\xac\x5e\xf9\x2c\xfc\x6a\x84\x5f\x1e\xa7\x67\xf1\xf7\x2c\xfe\xd2\x9f\x27\x26\xfe\x52\xc1\xb4\xb6\x96\x15\x50\x9d\xc5\xc3\x37\xfd\x7e\xd4\x4b\xf0\x08\x8f\x31\xc7\xd4\x4d\x47\x66\x8e\xb6\x99\xf5\x99\x50\xcc\x95\x68\x91\x24\x0f\x0d\xf1\xf2\x50\x98\x4a\x42\x72\x42\x27\xe9\xc7\x17\x84\xd6\x17\x9a\xaa\xa1\x2c\x2a\xa4\x16\xe2\x20\x15\x48\x71\xbc\x36\x87\x85\xea\x25\xf7\x67\x88\x26\x38\x6f\x27\x93\x6f\xf9\x3f\x25\x93\xc8\xcf\xfd\x4d\x24\x0e\xc4\x6a\x03\x5f\x6a\x54\x8a\x8a\x6a\x21\xa5\x63\x26\xb9\xb3\x85\x8a\xb8\xfa\x52\x9a\xe6\xc5\xc5\x34\x3b\x27\x45\x90\xef\x7f\x1c\xd7\xb1\x56\xb2\x10\x4a\x4c\x90\x67\x32\x0b\x1e\xf3\x30\x01\xbd\x76\xbd\xca\xea\xb0\x62\x03\x7a\x22\x91\x65\x25\xcf\x2d\x9e\x6a\xb6\x61\x91\xed\xac\x95\x34\x18\x79\xae\x59\x09\x10\x55\xf1\x0e\x28\x68\x86\xb2\x93\xa8\xd5\x61\xe9\x1b\x6b\x71\x58\x48\xa0\x1e\x9e\xa1\xb0\x94\xdb\xfc\xc0\x0c\x90\xbf\x23\x35\xfb\x29\xe8\x80\xc6\xaf\xc8\x8f\xb0\x18\xc0\x3f\x51\x7c\xd4\x67\x1d\x42\x8e\x43\xa4\x26\x4f\xfd\xca\xae\x88\x20\x8c\xea\xbf\x38\x46\xde\x6c\x1d\xc6\xfa\xf6\xc2\x75\xf0\x70\xfa\xf5\xba\x71\x7a\x12\x3a\xf9\x03\x1a\xcb\xf2\x90\xeb\xb3\xc8\x1b\xea\x36\x3c\xcc\x17\x93\x79\x88\x02\xac\x0c\x97\x37\xaa\x8e\xda\x0a\x6b\x37\x87\x87\x43\x9f\xcd\xda\xf0\x96\xf1\x44\x57\xc0\xee\x97\xe3\xa5\x29\x08\x22\x5f\x92\x21\xfa\x66\x67\x8f\xea\x61\x62\xb5\x4c\x6c\x47\x61\x6d\x90\xa6\x17\x46\x9b\x2a\xfa\xd6\xd1\xf8\x38\xbc\x6b\x86\x0e\xc9\xd0\x0b\x03\x18\x40\x24\x5a\x18\x09\xd9\xea\x68\x27\xc1\x2a\xe3\xd1\xd7\x7d\x2c\xbd\x86\xf5\x01\xed\xa5\x27\xcb\x64\x90\x0c\x51\xc5\x2f\x3a\x66\x3c\x40\x72\x00\x1e\x92\xb8\x25\x49\x80\x97\x6d\x32\xbe\xb1\xe3\x3e\x9b\x34\xac\x39\x5c\x51\x98\x25\x37\x63\x2f\x5b\x3e\x39\x7d\x36\xd4\x25\x96\xab\x65\xbd\xea\xcc\x26\x06\x6a\x4e\xfe\x5a\xe5\xcc\xf7\x96\xad\x56\xe2\xb5\xda\x85\x86\x85\x94\x22\xbf\x6a\xcd\x0b\x8d\x4e\xf1\x53\xad\x69\x2b\x9f\x1a\xcd\x5a\xf9\x58\x09\xe5\x32\xce\x77\x3c\x35\xfd\x7d\xd5\x45\x69\x12\xb2\x9f\xc5\xd3\x51\x22\xdb\x80\x50\xba\x67\xfb\x4f\xd2\x29\x8b\xa6\x7c\xf7\xd3\x41\x4c\x54\x69\x9a\xd4\x97\x57\xa5\xb9\x9b\x1a\xb2\xec\xfb\xf1\x46\xc9\x60\xf1\x7d\xac\xaf\x81\xa8\x40\xda\x32\x8d\xa7\x0d\x94\x45\x62\x4d\x3f\x1b\x0b\x2a\x96\xf8\xb8\xcc\xc0\xf3\xad\xab\xb9\xc4\xfe\x59\xec\x62\x9d\xd5\xc2\x85\xe2\x49\x9b\xc5\xe4\x40\x5d\x5d\x2b\x97\x7c\x8e\x45\x7c\x39\x76\xe2\x62\x80\x11\xf3\x12\xf2\x2b\xcc\x50\xba\x42\xc4\xfc\x04\xe8\x66\x98\x5c\x96\x3d\x8c\x4f\xdb\x16\xf2\x38\x97\xb4\xe7\xad\x8d\x57\x8e\xac\xe6\xce\xb2\xc5\x27\x56\x51\x48\xe2\x41\x54\x4c\xf3\x0a\x7b\x57\xf7\x2d\x06\x6e\xdb\x18\x96\x60\x06\xeb\xd0\x17\x19\x09\x07\xd4\x53\x3b\x68\x9c\x5d\x3b\x7e\x8d\x61\x8a\xae\x70\x72\x4c\x39\x1b\x5f\x72\xfa\x39\x69\xbf\xd6\x56\xb1\x5f\x21\xb2\x0c\x2b\xa4\xd7\xa2\x31\x6f\x06\x02\x53\xa9\x8c\xac\x6c\xed\xc0\xa7\x8f\xc7\x27\x0b\x36\x7e\xca\xa0\x58\x6d\xaa\xe7\x9b\x80\x95\xf9\x2e\x9a\x4c\x0a\xb6\x38\x2e\x92\x02\xe5\xfa\x91\x90\xea\xab\xd8\xf0\x4a\x8e\x83\x13\x5a\xb7\x39\xb4\xd9\x81\xa5\x7c\x5a\x69\xce\xea\x4a\xa6\x19\x5a\xfd\xeb\x32\x3a\x26\x93\x68\x1e\x15\x92\x29\x1a\x74\xcb\xbb\xbf\x57\x08\x28\xdb\x96\x65\x5b\xac\xd0\x7b\x53\x8d\x9f\xc6\x16\xb0\xad\xb3\x36\x1c\x48\x08\x22\x21\x15\x51\x22\xce\xd7\xf4\xd9\x35\xe6\x2d\x17\x09\x0c\xc8\x0f\xa7\x88\x46\x01\xe6\xca\xf6\x9c\x22\x8e\x5c\x89\xb9\x00\xc6\xa1\xd9\x6c\x35\x9b\xeb\x6a\xdd\xf0\x38\xbd\x0a\x51\x53\x7e\x84\x65\xbe\xf4\x3a\x20\xaa\x03\x4e\xc5\x52\x95\x56\x4d\x39\x17\x51\x1d\x96\x1c\x61\xf0\x19\x9d\x28\x3c\xa6\x88\x42\xaf\x9b\xeb\xbe\xdd\xac\x9b\x97\xaa\xb5\x6d\x39\xb9\xae\x8a\xdc\x1f\x3b\x14\x03\x7e\xd6\x45\xd3\x4c\xb2\xab\x77\x8b\xd9\xd5\x40\x28\x7c\xd8\x3d\x6e\x1d\x1f\x7f\x4c\x57\x54\x4a\xcd\x9b\x98\x1a\x1d\xf7\x8b\xe4\x14\x53\x19\xbb\xd0\x9a\x0f\xbb\x49\xac\xee\xdf\x8b\x83\x35\xaf\xa6\xc0\x04\x53\xb5\xe3\xc7\x1e\x44\x94\x5c\x46\x18\x88\x97\x70\x63\x29\x88\x59\x76\xe2\xad\xb4\xff\x28\xf6\xbd\x74\x53\xf9\x6a\xf7\xd3\xa2\xeb\x13\x4c\xe5\x32\xde\x8d\x52\x0d\x81\x5d\x5e\x4d\x1e\xb9\xa7\x2d\xdb\xbd\xef\xc2\x56\xdf\x96\x58\xb3\x5c\x1a\x96\xb5\x53\xf2\x81\x94\x96\x90\x5d\xf1\x28\xb1\xaa\x87\x58\x8d\x8d\x37\xef\x55\xf1\xac\x26\x6f\x17\x70\xb8\x15\xd7\x39\xec\x58\xec\x64\x37\xff\x77\xc5\x1a\x5b\xae\xab\xca\xf4\xad\x30\x75\xb6\x7d\x65\xf5\xd4\xe4\x41\x6e\xeb\x65\x5f\x4a\x3f\x9c\xdc\x9a\x2b\x1a\x8a\x04\x98\x62\x7f\x8a\x9c\x5c\x92\x57\x17\xf6\x62\x15\x44\xc5\x6e\xd2\xdb\x64\xef\x84\xde\xad\x45\x58\x75\x7e\x2b\x47\x9c\xd4\x4a\xd2\xb9\x29\x12\x05\xe1\x7d\x68\x8f\xb9\x75\xca\xe4\xe4\xd7\xf2\x22\x84\xaa\x2b\x6c\xee\x86\xfe\x16\x3b\xf3\x6a\xeb\x55\x47\x8a\x65\x6f\xbd\x42\x5a\x63\x22\x13\x56\x70\xa9\x94\xe3\x33\x0b\x71\x7d\x50\xff\x8b\x7d\xa8\x8d\x25\xf6\x8d\x85\x98\x17\x94\x22\x59\x2f\x0a\xb9\x5b\x49\xc0\x3d\xc9\xe1\x7a\xa1\xcb\x58\xb3\x7d\xee\x93\x35\xac\x1d\x58\xfc\x37\x1d\x3a\x0a\x8f\xb7\x9d\x9f\xbc\xe8\x13\xee\xfb\x8e\x64\x3b\xe7\xc7\x93\xee\x9b\xf7\xdf\xc6\xd1\x12\xbc\xb4\x90\x93\x2a\x24\x7c\x37\x26\x7a\x22\xfc\x96\x21\x61\xa0\xcd\xfe\x5e\x31\xf6\x6b\x78\xaa\x1a\x92\xac\x70\x08\xf2\x3c\xa2\x64\x14\xf2\x3f\xcd\x01\xda\x8a\xd4\x95\x89\x12\x55\xda\x2f\x43\x64\x81\x67\x51\xd0\xdf\x34\x6b\xa6\xbf\xd8\xc5\x92\xe3\x4e\x65\x7d\x7d\xd0\x36\xd3\x2f\x84\xca\xad\x7e\x71\x68\xd5\xea\xe6\xae\x39\x4b\x6d\x8f\x45\x23\x1f\x2f\x30\x46\x75\x83\xf9\x35\x5d\x4e\x62\xf9\x0e\xab\xba\xdc\xc5\x83\xac\xeb\x3c\x11\x3f\xfa\xca\xce\x63\x61\xe0\xcd\x7f\xf2\xc8\x56\xf7\xe3\x5e\x45\xd6\x67\x04\x06\x0b\xaa\x6a\x33\x75\x35\x84\x5f\xe8\xfd\x17\x65\xd7\xf1\x8b\x09\xc4\x24\x6c\x33\xea\xcf\x40\x44\x61\xc8\xb8\xd2\xd4\x63\x82\x7d\x4f\x97\x34\xa1\xcb\xb4\x7a\xc5\x76\x2e\xc0\xbc\x56\xcd\x30\xca\xd8\xd8\xdc\x92\x9d\x26\xa7\x55\xfc\x5b\x07\x7b\xe6\xb9\x4a\x97\xf1\x34\x95\xbd\x94\x5e\x65\x99\x53\x42\x07\x10\x22\x39\x2d\xa3\x94\x79\xe3\x93\x84\xcc\x22\x1d\xc9\xa7\xb9\x66\xf2\xd7\x7a\x17\xa8\xd3\xe4\xf9\x98\x4e\xe4\x54\x5b\xbc\x24\xc0\x40\x28\x04\x84\x46\x12\x9b\x04\xf7\xeb\x29\x71\xa7\x6a\x53\xcf\x75\x1a\xa2\xb9\xcc\xd3\x08\x8b\xf9\x94\xcd\x1b\x60\x99\x15\xed\x8c\xe8\xe1\x31\x8a\x7c\x39\x80\xcd\x6c\xf9\x10\x4a\x82\x28\x18\x40\x27\xfb\xc8\xf8\xd4\x07\xd0\xef\x75\x9d\xf8\xd3\x6a\xb2\x59\x19\x23\x48\x19\x3d\x6e\x3d\x49\x51\x2d\x4d\x66\xfc\xe9\x32\x20\x2a\x0c\x93\xf2\x0a\x3d\x81\x5d\x46\x3d\x01\x23\x2c\xaf\xf5\xe5\x91\x48\x22\x48\x53\xfc\xbf\x2f\x62\x3d\x67\x29\xc8\x3a\xce\x8e\x33\x1f\xb3\x32\x24\x39\xcc\xe2\xf6\xe3\x5c\xb8\x22\x66\xf1\x87\xcb\x40\x96\xdc\x47\x90\xd8\xd1\x92\xc1\x18\x4b\x77\xda\x86\xb7\xea\x1f\xfd\x20\x77\x9a\xa8\x3b\xc5\x14\x70\x10\xca\x59\xdb\xd4\xc3\x54\xea\x53\x0b\x88\x17\xde\xae\x96\x98\x53\x94\x54\xd3\x24\x89\xf6\x42\x68\x8b\x1a\xa4\xa2\x3b\xe6\x78\x85\x62\xa0\xd3\xd7\x64\xb2\x94\x34\x83\x42\x2e\x55\x6e\x21\x04\x9f\xd0\x44\xb1\x8d\x87\x6f\x2a\x4c\x31\x46\xbe\x58\xcc\x15\x36\x07\x52\x8e\xf8\x72\xa2\x5c\x3c\x79\x49\x48\x3e\x9f\x21\x67\x88\xce\x25\xf4\x2d\x24\xfa\x30\xbb\xbd\x57\xc1\xa5\xb8\x1d\x23\x77\x9a\x1f\xf4\x3d\x0e\xa3\x9c\xc9\x97\x0e\xc3\x71\xcc\x40\xe2\x1b\xd2\xac\xbe\xad\xff\x6b\xa5\x35\x8f\xe3\x17\x2c\x62\x85\xa0\x2a\xc1\x68\x06\x2e\x27\x12\x73\x82\xda\x7a\x05\x8b\x19\x95\xe8\x26\xbd\xd0\x35\x95\xf6\x40\x44\x8e\xa0\x80\xf8\x88\x27\xcf\xbc\xe7\xab\x60\x38\x4b\x1a\x3e\x03\xd7\xd7\xf7\x1e\xb3\x31\x20\x0a\xc7\x9f\xdf\xeb\x90\x26\x36\x0f\xd4\x27\x6d\xed\x2b\xdc\x4c\xde\x79\x7c\xf5\xb1\xae\x6f\xae\x3e\x46\x74\x96\x36\xeb\x15\x43\x81\xe2\xcc\xe8\x30\x91\x35\x95\xcb\x3c\x5a\xcf\x49\x6a\xb5\x88\xaa\x6f\xbb\xe7\xbb\x91\x53\x4c\xb8\x66\x81\x75\x48\x2e\x6a\x1e\x33\xdf\x67\xd7\xfa\xad\x75\x3d\xbc\xc1\x5a\xda\xcf\xd9\xd9\x99\xb8\xcc\x12\x3d\xb5\xc7\x0a\x09\x37\xff\x7d\x56\xf8\xe4\x56\x74\xc0\x10\x51\x6f\x98\x46\x63\x94\x86\xbe\x0b\x69\xeb\x39\x57\xd5\x7c\x52\xcd\x9b\xff\x85\x29\xa7\x4d\x99\xc4\x29\xbd\x75\x60\x1c\x88\x29\xa3\x59\x50\x99\x16\x5a\x20\xad\xab\xcf\xb2\x50\x94\xf1\x82\x8b\xc8\x97\x46\x38\xe5\x46\xa8\x08\x6a\xa7\x8c\x1e\xfa\xcc\xc3\x05\xe1\x5f\x65\xfe\x12\x6f\xe7\xf9\x3f\x19\x5d\x63\xce\x92\x35\x6b\x3a\x6e\xe0\xae\xcb\x52\xc8\x99\xaf\x84\x3b\xe3\xe6\x10\x81\xb9\x63\xd0\xbe\xe4\xb2\x15\xa7\x0b\x65\x2b\x2c\xc7\x16\x8b\x97\x5a\xcd\x12\xd3\x21\xc2\xe2\xfa\xca\xfa\x2c\xac\x33\x88\xef\x5e\x8f\x17\x4b\x72\x71\xb3\xa1\x5e\xcf\xce\x59\x31\x70\x7d\xb6\x0e\x67\x0a\x38\xf5\xaf\xb6\x05\xd5\x2f\x26\x32\x79\x66\xe2\xa1\x67\x26\x31\xe1\x2c\x6b\x5b\x6d\x1a\x10\x47\x92\x71\x33\xe1\x67\xff\xf3\xbf\xaa\xd6\xab\x33\xcd\x32\x67\xef\x0f\x7e\xd9\x3f\xcb\x96\x69\x52\xeb\x9c\x11\x1a\x97\xdf\x3d\xdc\x3b\x33\x6d\x7f\x3c\x3a\x6b\xc3\x4f\xec\x1a\x5f\x61\xbe\x0e\x33\x16\x69\xa9\xa0\x46\x89\xd2\xec\x01\x36\x86\x8e\x13\x57\x27\x54\x87\x57\xf4\x68\xf4\xdc\xe7\x30\xde\x4f\x99\xc9\xb6\x1a\x2d\x4f\x9c\xa5\x57\x72\x68\xce\x3a\x0b\x66\xad\x44\xe8\x18\xea\x64\x16\xa4\x3d\x43\xd7\xe2\x6c\xd9\x25\x59\x5c\x8f\xaf\x20\xd7\xb0\x89\x30\x17\x66\x00\x5e\x01\xba\x16\xf9\xfa\xff\x0c\x5b\x7f\xac\x34\x06\x64\x7a\xd2\x17\xca\xeb\x70\x78\x7c\xce\xe7\x2c\x98\xdd\x92\x68\x9f\x5c\x60\x08\x66\x7f\xeb\x6e\x7e\x17\xd9\xa1\x85\x63\x21\xc8\x9c\x0a\xc8\x9c\x58\x41\x32\xbb\x7f\x7f\x8a\x04\x84\x98\x07\x44\x08\xed\x29\x67\x20\xb0\x39\x60\xca\xe3\x73\x42\x39\x4e\x38\x64\x12\xb7\x13\x1a\x8d\xb2\xc9\xce\xd2\x28\xae\x8e\x4f\x8d\xe8\x1b\xcc\x93\xda\xf3\xa5\x54\x6c\x2c\x68\xae\x9b\x23\x7b\xec\x72\xc6\xa2\xdb\x0b\x62\x04\xca\xd2\x6d\x39\x5e\x69\xdc\x56\x8c\x25\x07\xb5\x74\x08\x33\x21\x2b\x3e\xa9\x95\x6f\x13\x0f\x60\xa4\x3f\x8d\x3f\x34\x7f\xbc\x8d\x6d\xf0\x9f\xbf\x9c\x14\xb6\xa8\x53\x29\xc3\xb5\xf2\x60\x4b\x2f\xf3\x24\xcd\x97\x36\xdb\x71\x96\x04\x34\xd2\xc4\xe1\xcc\xa1\x53\x4a\xb0\x81\x46\x6e\xe4\xc9\x9c\x34\x92\xab\xe9\x42\x22\xd3\xac\xc3\xd2\xf3\x3d\x75\x5d\xe3\xa8\x75\x8d\xef\xa9\x6b\x7b\xf2\xe6\x1c\x0a\x8c\x6f\x8b\x1c\x7f\xdd\x3a\xfa\xdc\xfb\xf9\x97\x83\x9d\xcf\xce\xc7\x93\xe0\xfc\xf3\x5b\xaf\xc7\xdc\xb7\x47\x93\xac\xc7\xd8\x63\x96\x30\x46\xf6\xc5\x92\x99\x83\x1b\x4b\xf5\x12\xe7\xdc\x43\x43\x27\xcb\x2f\x8b\x46\x9a\x80\x94\x5f\x31\x8b\x67\xd6\xf8\x1c\x00\x1a\x28\x24\xc3\x84\xc8\x61\x8c\xe7\x02\x9c\x73\x24\x65\x91\x38\xfd\xd8\x92\xd3\xea\x38\x2d\x67\xf3\xa4\xd3\x1d\x6c\x76\x06\xdd\x7e\xdb\xd9\xec\x75\xfa\xdd\xdf\xb3\x1a\xb9\x8c\xed\x4a\x8d\xad\x41\x6f\xab\xdd\xdb\xea\x76\x9d\x9d\x5c\x8d\x24\xb5\x1a\x1a\xdd\xf6\x56\xdb\x69\xe4\xf6\x89\xf9\x1c\x6a\x05\x1b\xf5\x50\xec\xe2\x28\x31\xc0\x5b\x9d\xd4\x9d\xbc\xec\x63\x12\x1b\x9f\x28\x53\x98\xfc\xf4\x1f\x9e\x2b\x8a\x69\xfa\xd0\x40\xf1\xf1\x23\xeb\x33\x0e\x59\x64\xb7\x8c\xde\x22\x1e\x5a\x94\xeb\x30\x87\x6b\x62\x50\x82\x59\x0b\x85\x61\x4b\xa0\x46\x6e\xe7\x9e\x3f\xdf\x51\x8e\x3c\x8f\x19\x87\x60\x06\x28\x0c\x6d\x49\x14\xcb\x70\x6a\x85\x1f\x8b\x4d\x2c\xcb\x95\xc5\x2b\x46\xc5\x46\xa7\x32\xe1\x77\x1e\x1b\x14\xf2\x1f\xa0\x71\xbc\xdb\xea\x74\xd5\x7f\x95\xaf\xe3\x3c\x28\xd5\xa4\xfa\xa5\xc2\xa2\x0d\x25\xf0\x5b\xca\x2c\x99\xcf\x7b\x9d\x96\xd3\x6f\x39\xdb\x27\x9d\xad\x41\xb7\x3f\x70\x3a\xff\xed\x6c\x0e\x7a\x8e\x0d\xe5\xdc\x3d\x7a\x3f\x0e\xd2\xdf\x05\xc9\x52\x80\xff\x2e\x68\x56\x03\xe8\x3f\x10\xaa\xf3\x82\xdd\x73\x00\xad\xc6\x78\x86\x43\x35\xe8\xe1\x70\x00\x05\x39\x8e\xf9\x70\xc4\xd9\x05\xe6\x92\x85\xc4\x8d\x1d\x9d\xc3\xd1\x4c\x62\x31\x24\x74\x58\x3c\x4f\x0b\xda\x90\x0d\xbe\x91\x21\x61\xc3\x38\x21\x36\x6b\xaf\x55\x7d\x61\x56\x37\x3a\x80\xe1\xd0\x65\x54\x44\x01\xe6\x43\x36\x1e\x0b\x9c\xbb\xed\xb5\x1a\x0a\x6e\xe5\x02\x48\xd0\xd9\xea\x74\xb6\xb6\x9d\x6e\xcf\x71\x1c\x27\x57\x28\xb5\xd1\x77\xfa\x9d\xcd\x7e\x5d\xed\xad\xb9\xb5\x37\x77\x76\x76\xea\x6a\xbf\x9c\x5b\x7b\x7b\xab\xdb\xcd\xcf\x8e\x25\x64\xf9\xd4\xe7\xa7\x76\x2e\x2a\xf3\xd0\x8f\x5f\x5c\xaf\xb5\xf6\xcd\xa2\x77\x7a\x95\x65\x9f\x3b\x01\x0b\xb5\xab\xdc\xbc\xb7\xb8\x51\x68\x47\x1f\x55\x86\xc6\xde\xc1\xe1\xc7\xe3\xdd\xd3\xa3\xe3\xd6\x87\x77\x1f\x4e\x5a\x85\x22\xa9\xb1\x70\x9c\x7b\x1a\x36\x79\x34\xd6\xbc\x2b\x97\x46\xea\xcc\x76\x5d\x3f\x22\xfb\x4a\xa7\xeb\xa7\x5b\xec\xdc\x32\xcf\x9f\x60\x56\xd6\xe2\x97\x03\x12\x5c\xbe\x73\xf9\x5e\xf4\x7e\xab\x83\x4e\x6f\x0e\x7e\xbf\x7c\x7d\x72\x79\x78\x14\xcb\x9a\xf9\xef\x53\x3e\xa3\x54\x40\x69\xc1\xb3\x47\x36\xa4\xba\xf7\x84\x54\xb7\x16\xa8\xae\x0d\x27\xb3\xb9\x00\xc9\xd4\xe8\x05\x2e\xb8\xc5\x06\x70\x4a\xd1\xc8\xd7\xa7\x3d\xf4\xc5\xff\x95\xf7\xb3\x4c\x3a\xbf\xc5\xd4\x1e\x40\xa5\xf3\x01\xd4\xf5\x95\x45\x9a\x5d\xe6\x47\x01\x35\x5e\x24\xd5\x45\xec\xf1\x80\x26\xf1\x9a\x6d\x38\xb6\x95\xd3\x5e\xc1\x41\xbc\x39\x58\x8f\x1d\xf3\xc5\xfd\x45\xf2\xa9\xd9\x8e\xb4\xe1\xb3\x71\xea\x98\xf9\x1a\x00\xf1\xe0\x15\x74\xf2\x28\x95\x67\xdf\xff\xb2\xf7\x2e\x9a\x8d\x0e\xf8\x3e\xbd\xe1\xbb\x38\xd8\xee\xf6\x27\x97\x17\x17\x64\xef\x2a\x99\xfd\xf2\x43\x85\xb6\x19\xef\x3b\xfd\x7b\x99\xf1\xed\xba\x09\xdf\xb6\xcc\xf7\x32\xaf\x1d\xa6\x83\x31\x57\x70\x2e\x31\xa4\xed\x87\x1b\x50\xb6\x59\x8e\x87\x62\x2e\x27\xf2\x5e\x35\x3b\xe4\x97\x9e\x17\xfd\xfa\xf5\xe0\xea\x6a\xf3\xeb\xd5\x7b\x7f\xf6\xad\x13\xbc\x3b\xea\xfd\x3c\xbb\x3c\x6c\x6a\x49\xa0\x1f\x6b\x5f\xb0\xd6\xbf\x7e\xdc\x9e\x74\x27\x5b\x3f\x9d\x78\xa7\xbf\x9c\xa2\xee\x85\xf8\x69\xa7\x7b\xf1\x79\xaf\x37\x4b\x00\x2a\x5f\x56\x63\x95\x84\x55\x2b\xf1\x56\x82\xb0\x53\x2b\x07\x3b\x16\x74\xb2\x05\x7c\x85\x39\x19\xcf\xe0\xe7\x2f\x27\xe6\x62\xa0\x01\x1c\xc5\xee\xc6\xf4\xee\xc4\x38\x7f\x58\x5f\x1b\xb4\x14\x3e\xbd\xd3\xe9\xfe\xf4\x3a\xf8\xed\x75\xf8\xe5\xd3\xf8\xa0\xeb\x1f\xe2\x8b\xd0\xeb\xff\xbe\x97\xe0\x53\xbe\x96\xdf\xba\x1a\xee\x05\x9e\x7e\x1d\x3a\x7d\x1b\x38\x02\x73\x68\x8e\x19\x6b\x8d\x10\x6f\x2e\xfb\x3a\x6b\x7b\x81\x7c\xf8\xda\x3b\x25\xfb\xd3\x6f\x34\x87\xc8\x79\xe8\xf5\xbf\xbe\x49\x11\x59\xf6\xb5\x5c\x1b\x54\x9b\xf7\x02\xd5\x66\x1d\x54\x9b\xf5\x50\x4d\x91\x48\x0f\x6a\x22\xcb\xfb\xb4\x5b\x80\xe2\x80\x4d\xea\x92\xaf\x85\xed\xe2\x46\xc1\xf6\xeb\x27\x7c\xd0\x65\x87\xf8\xdc\xeb\xfd\xf6\x3a\x45\xad\xe6\x69\x5f\xeb\xba\xeb\xde\xcf\xba\xeb\xd6\xae\xbb\xae\x05\xaf\x74\x6d\x49\x45\xb9\x39\xd9\x6a\xce\x03\x62\x0a\xc9\x55\x23\x73\x11\xb9\xf8\xed\xcd\xb7\x2f\x1a\x88\x04\x91\xf7\x57\x6f\x5f\x9e\x7f\xf8\xfc\x35\x41\x64\xd1\xe3\xae\x36\x34\x7a\x5b\xf7\x82\x46\xbe\x19\x3b\x1a\xf9\x12\x15\x19\x9d\x9e\x49\xd4\x8a\x9d\x08\x40\xbe\x76\x13\xeb\x0b\x4d\xe6\xa2\xb1\x75\xf1\xd5\x51\xfc\xf1\x2d\x83\xe5\x2b\x9e\x7a\xbd\xfd\x58\xce\x54\xef\xc2\xb2\x21\xf0\xf2\x5e\x00\x78\x59\x37\xfe\x97\x56\x21\x1c\x5f\x7c\x92\xdc\x34\xb6\x40\xa6\xe2\xfd\x64\xaa\xb7\xbe\x4e\xa6\xe3\x0f\x2f\x27\xef\x8e\xc4\x4f\x57\xfb\x5f\xd2\xb1\x2e\xad\x94\x1f\x70\xc4\x26\x32\x96\xdc\x89\x03\xca\xc4\x16\x58\x0e\xe0\xe3\x9b\x0f\xad\xfd\xdf\x5a\x2f\x07\xb1\x2f\xd3\x5c\x62\xa3\xc6\x93\x95\xc1\x37\xb2\x95\x39\x66\x5b\x1d\x72\xe3\xf4\x7c\xea\xf9\xc1\xa5\x73\x39\x76\xb7\x05\x91\x68\x53\xf8\xe7\x57\x3b\xf9\x7d\xa1\x32\x12\x13\xe6\x52\x83\xef\x4c\x36\xbd\x9d\x9d\x4b\xc7\xe7\xae\x77\xd5\x9f\x6c\x23\x7f\xb4\x2d\xfc\xf1\x84\x9e\xf7\xbc\xe9\x48\x9c\xff\xed\x3f\xfe\xbe\xff\xdb\xc9\xd1\x2e\xfc\x97\x19\x66\x5b\x43\xf3\x8a\xe8\xfb\x69\xc7\xa4\x90\xb3\x49\x04\x34\xfb\x4e\xbf\xb9\xae\x01\xd0\x7f\xbe\x79\x7f\x7a\x7c\xb2\x9f\xea\x16\xa7\xdf\xd4\x71\xb6\x74\x36\x21\x6b\x48\x97\xef\x4c\x36\x19\xdf\x74\xae\x48\xe4\x6c\x33\xac\xe6\x6a\xca\x2f\xdc\xee\x96\x37\x19\xcb\xf3\x0e\x72\x0b\x97\xd4\x25\x47\x65\x9b\x75\x83\xc8\x59\x31\xff\x58\xa4\xa6\x4f\xc4\x17\x3e\xdb\xa2\xe2\x72\xd4\x15\x87\xc1\xdb\xf3\xcd\xd1\x6f\xe1\xde\xf6\x1b\xd4\x58\xfb\xff\x00\x00\x00\xff\xff\x1d\xb8\x84\x03\x53\xa2\x00\x00")

func fleetManagerYamlBytes() ([]byte, error) {
	return bindataRead(
		_fleetManagerYaml,
		"fleet-manager.yaml",
	)
}

func fleetManagerYaml() (*asset, error) {
	bytes, err := fleetManagerYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "fleet-manager.yaml", size: 41555, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
	"fleet-manager.yaml": fleetManagerYaml,
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
	"fleet-manager.yaml": &bintree{fleetManagerYaml, map[string]*bintree{}},
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
