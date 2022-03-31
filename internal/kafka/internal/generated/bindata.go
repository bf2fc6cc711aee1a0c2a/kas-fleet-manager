// Code generated by go-bindata. (@generated) DO NOT EDIT.

//Package generated generated by go-bindata.// sources:
// .generate/openapi/kas-fleet-manager.yaml
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

var _kasFleetManagerYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x7d\x6b\x73\xdb\x38\xb2\xe8\xf7\xfc\x8a\xbe\x9a\x7b\x4a\xe7\xcc\xb5\x64\x4a\x96\x1f\x51\xed\x6c\x95\x93\x38\x19\xcf\x24\x4e\x62\x3b\x93\xc9\x4e\x6d\xc9\x30\x09\x49\x88\x49\x82\x06\x40\xdb\xca\xdc\xfd\xef\xa7\x00\xf0\x01\x92\x20\x45\xf9\x11\xdb\x33\xf6\xd6\xd6\x44\x12\xd0\xe8\x17\x1a\x8d\x46\xa3\x41\x23\x1c\xa2\x88\x8c\x61\xa3\xef\xf4\x1d\xf8\x01\x42\x8c\x3d\x10\x73\xc2\x01\x71\x98\x12\xc6\x05\xf8\x24\xc4\x20\x28\x20\xdf\xa7\x97\xc0\x69\x80\x61\xff\xd5\x1e\x97\x5f\x9d\x85\xf4\x52\xb7\x96\x1d\x42\x48\xc0\x81\x47\xdd\x38\xc0\xa1\xe8\x3f\xfb\x01\x76\x7d\x1f\x70\xe8\x45\x94\x84\x82\x83\x87\xa7\x24\xc4\x1e\xcc\x31\xc3\x70\x49\x7c\x1f\x4e\x31\x78\x84\xbb\xf4\x02\x33\x74\xea\x63\x38\x5d\xc8\x91\x20\xe6\x98\xf1\x3e\xec\x4f\x41\xa8\xb6\x72\x80\x04\x3b\x0a\x67\x18\x47\x1a\x93\x1c\x72\x27\x62\xe4\x02\x09\xdc\x59\x03\xe4\x49\x1a\x70\x20\x9b\x8a\x39\x86\x4e\x80\x42\x34\xc3\x5e\x8f\x63\x76\x41\x5c\xcc\x7b\x28\x22\xbd\xa4\x7d\x7f\x81\x02\xbf\x03\x53\xe2\xe3\x67\x24\x9c\xd2\xf1\x33\x00\x41\x84\x8f\xc7\xf0\x2b\x9a\x9e\x21\x38\xd2\x9d\xe0\xb5\x8f\xb1\x80\x77\x0a\x14\x7b\x06\x70\x81\x19\x27\x34\x1c\xc3\xa0\xbf\xd1\x77\x9e\x01\x78\x98\xbb\x8c\x44\x42\x7d\xd9\xd0\x57\xd3\x72\x88\xb9\x80\xdd\x0f\xfb\x12\x49\x8d\x5f\xd2\x87\x84\x5c\xa0\xd0\xc5\xbc\xff\x4c\xe2\x8b\x19\x97\x28\xf5\x20\x66\xfe\x18\xe6\x42\x44\x7c\xbc\xbe\x8e\x22\xd2\x97\xdc\xe6\x73\x32\x15\x7d\x97\x06\xcf\x00\x4a\x18\xbc\x43\x24\x84\xff\x8e\x18\xf5\x62\x57\x7e\xf3\x3f\xa0\xc1\xd9\x81\x71\x81\x66\x78\x19\xc8\x23\x81\x66\x24\x9c\x59\x01\x8d\xd7\xd7\x7d\xea\x22\x7f\x4e\xb9\x18\xef\x38\x8e\x53\xed\x9e\xfd\x9e\xf7\x5c\xaf\xb6\x72\x63\xc6\x70\x28\xc0\xa3\x01\x22\xe1\xb3\x08\x89\xb9\xe2\x80\x44\x73\xfd\x4c\xb2\x88\x4f\x82\x59\x20\xd6\x2f\x06\x63\xd5\x7b\x86\x85\xfe\x07\x48\x05\x64\x48\x82\xd9\xf7\xc6\xf2\xfb\xdf\xb4\x8c\xde\x61\x81\x3c\x24\x50\xd2\x8a\x61\x1e\xd1\x90\x63\x9e\x76\x03\xe8\x0c\x1d\xa7\x93\x7f\x04\x70\x69\x28\x70\x28\xcc\xaf\x00\x50\x14\xf9\xc4\x55\x03\xac\x7f\xe5\x34\x2c\xfe\x0a\xc0\xdd\x39\x0e\x50\xf9\x5b\x80\xff\xcb\xf0\x74\x0c\xdd\x1f\xd6\x5d\x1a\x44\x34\xc4\xa1\xe0\xeb\xba\x2d\x5f\x2f\xa1\xd8\x35\x3a\x17\xd8\x92\xb4\x83\xa0\x48\x0b\x8f\x83\x00\xb1\xc5\x18\x0e\xb1\x88\x59\xc8\x95\xc2\x5f\x94\xdb\xda\xd9\xb7\x8e\x19\xa3\x8c\xaf\xff\x49\xbc\xff\x2c\x65\xe5\x9e\x6c\xfb\x62\xb1\xef\x3d\x44\x26\x2a\xe4\x6a\x59\xf7\x06\x0b\x50\xa4\x4a\xe3\x92\x11\x60\xe5\x5c\xd6\x8c\xa4\xcd\x04\x9a\x19\x24\xf6\x74\x0b\x9e\x7c\x11\x21\x86\x02\x2c\x92\x39\x9a\x36\xd1\x98\x76\x0a\x98\xe6\x2d\xd7\x89\xd7\x69\x16\x48\x3b\x59\xf0\x07\x2b\x88\xb7\x84\x8b\x5a\x61\xc8\x1f\x81\x4e\x21\xa2\x9c\x13\x69\xf0\x0b\x0c\xb5\x0a\xc5\x2f\x77\x91\x66\xb3\xd0\xad\x46\x48\x35\x5c\xd6\x1f\xdb\xa9\xbd\xb2\xc9\x0f\x55\xed\x15\x72\x87\xf8\x3c\xc6\x45\x86\xcb\x3f\x7c\x85\x82\xc8\x37\xf1\x4c\xff\xcc\x5e\x6f\xb0\x38\x4c\x28\xda\xd3\x1d\xaa\xed\xed\x38\xa4\xf0\x0b\x48\x24\x30\xca\xb8\xd4\x8e\xf9\x99\x88\xf9\x6b\x44\x7c\xec\xbd\x64\x58\xf1\xe6\x48\x20\x11\xf3\xdb\xc0\xa5\x01\x6e\xad\x72\xea\x15\x98\x69\x00\x30\xa5\x71\xe8\x29\x9b\xf1\x2a\x17\xf6\xc8\x19\x3c\x10\x1b\xd7\x2c\xe5\x91\x33\xb8\x2e\x17\xf3\xae\xb5\x8c\xda\x8d\xc5\x1c\x04\x3d\xc3\xa1\xf4\x66\x48\x78\x81\xfc\xcc\x62\x2a\x26\x6d\x3c\x12\x26\x6d\x5c\x9f\x49\x1b\xcb\x98\xf4\x89\x63\x06\x21\x15\x80\x62\x31\xa7\x8c\x7c\xd3\xde\x2b\x72\x5d\xcc\xb5\x65\x4b\x1c\x52\x93\x71\xa3\x47\xc2\xb8\xd1\xf5\x19\x37\x5a\xc6\xb8\x03\x5a\x9a\x89\x97\x44\xcc\x81\x47\xd8\x25\x53\x82\x3d\xd8\x7f\x05\xf8\x8a\x70\xc1\x73\xc6\x6d\x3e\x18\xd7\xa3\x99\x71\x9b\x8e\x73\x5d\xc6\xe5\x5d\xeb\x35\x2e\xc4\x57\x11\x76\x05\xf6\x12\x4f\x86\xba\xca\x9d\xce\x7c\x1e\xec\xc6\x8c\x88\x85\xb9\x56\xbe\xc0\x88\x61\x36\x86\x3f\xe0\xdf\x75\x8b\x30\x2a\x89\x23\x37\x89\x1e\xf6\xb1\xc0\xd6\xc5\x53\xff\x54\x5e\x3f\xed\x1e\x13\x09\xc7\x70\x1e\x63\xb6\x30\x08\x0b\x51\x80\xc7\x80\xf8\x22\x74\xeb\xc8\xfd\x80\xd9\x94\xb2\x40\x4d\x25\xa4\x36\x39\x40\x42\xb9\x11\x55\xbd\xe6\x8c\x86\x34\xe6\x72\x77\x15\xaa\xdd\x4a\x93\x98\xc5\x22\xc2\x63\x38\xa5\xd4\xc7\x28\x34\x7e\x91\x24\x13\x86\xbd\x31\x08\x16\xe3\x46\x27\x60\xf8\xf0\x14\xb0\x0c\xe9\x87\x03\x0a\x2f\x35\x62\x75\x3c\x7d\xa5\xc4\x56\xb0\xe5\x8f\x63\x66\x8d\x1c\x47\xe1\x4e\x68\x78\x7d\xd3\x54\x06\x51\xbf\x1d\x93\x0b\x9e\xa2\x37\x71\x36\xcb\x53\xed\xc9\x55\x78\x72\x15\x9e\x5c\x05\xed\x2a\x68\x9b\x72\x03\x87\xa1\x00\xe0\x6f\xea\x36\xdc\x8c\x89\x65\x00\xd7\x77\x21\x52\xe7\x40\x83\x6b\x72\x0e\xda\xf9\x1b\x11\x12\xee\x7c\x5c\x86\xfe\x29\xf2\x90\xc0\x19\xf0\x34\x28\x5a\x08\xcd\xb4\xf3\x66\x0a\x4e\x49\xac\xc0\x56\x37\xf5\x0a\xf5\x17\xd4\x33\x60\x15\xb9\xa2\xd1\xa1\x97\x21\x66\x40\xa7\xa0\x42\x08\xcf\x1a\xb4\xa6\x59\x67\xec\x1a\xb3\x74\xab\xaf\xb1\xa8\x6c\xf8\x57\xf0\x51\x8a\xda\x6e\xd9\xfb\x6a\x06\x95\x77\xbd\x8f\x2a\xa6\xf1\x81\xf2\xbb\x0d\x6a\x54\x5c\xa2\x02\x1f\x5f\x20\x2f\x55\xa8\x47\x60\x58\xde\x11\xce\x49\x38\xfb\x90\xba\xe5\x37\x70\x9d\x6a\x40\x75\xeb\x1d\xa2\x15\xfc\x84\x87\xcb\xc1\xe5\xde\x13\xac\xe4\x3e\x55\x3c\xa2\xaa\xa3\x40\xb8\xe9\x2b\xf0\xa5\xbe\xc2\x43\x66\xde\xad\x7a\x55\x15\xa7\xc8\xee\x1f\xe8\xc0\x9e\xf2\x0e\x14\xbb\x0c\x0f\xe1\x51\xf0\xec\x56\x63\x2f\x15\x1f\x68\x25\x77\xe0\x21\x33\xea\x16\x63\x2d\xd5\xb0\x45\xab\x63\x9e\xa6\xf3\x07\x0d\x28\xa2\xdc\x7e\xf6\xe0\x32\x9c\x7a\x2a\xc9\xcf\x7f\x91\xd0\xc9\x32\x57\x4b\x4f\x51\xe3\x88\xf3\xfb\xf9\x57\xa9\x03\x81\x16\x3e\x45\x5e\x51\xd1\xea\xd4\xec\xd3\xd1\x21\x9e\x91\xaa\x7e\x2f\x51\xb0\xb4\x5b\xcd\x89\xc9\xde\xa7\x6b\x41\x4d\xbb\xd5\x41\xbd\x92\x4c\x23\xe2\x88\x7c\xab\xf7\x8c\x9a\x07\xa8\x42\xb8\x96\x1f\x7a\x4f\xb1\xb2\x47\xe0\x5c\x96\xbd\x22\xd7\xc5\xd1\x63\x8d\xc7\xa5\x87\x6f\x37\x70\x2a\x4b\x20\x9e\xe2\x71\x4f\xf1\xb8\x94\x49\xb7\x1c\x8f\xcb\xc0\xbe\x43\x57\xbb\xbe\x4f\x2f\xb1\xb7\x9f\x44\x1d\x0e\x31\x72\xe7\xd8\xbb\xc1\x78\xcb\x60\x5a\x11\x39\xc6\x2c\xe0\x07\x54\xa4\x36\xe0\x06\xe3\xd7\x80\x6a\x8e\x47\x4e\x29\x3b\x25\x9e\x87\x43\xc0\x44\xcc\x31\x83\x53\xec\xa2\x98\x63\xe5\x34\xc4\xd5\x8d\x48\x6d\xd0\x12\x68\xb1\x6f\x80\xae\x48\x10\x07\x10\xc6\xc1\xa9\x8e\xa7\x64\x49\x6f\x20\xe6\x48\x80\x8b\x42\x38\xc5\x89\x0f\xa4\x82\x11\x2a\xcb\x50\x8d\x39\x47\x1c\x4e\x31\x0e\x81\x69\x0e\xf6\xeb\xbd\xff\x87\xab\xbb\x77\x79\x7a\x7a\x3c\xc7\xa9\x9b\x85\x3d\xb9\xfe\xd2\x98\xb9\x18\x3c\x8a\x79\xd8\x15\x3a\x04\x6a\xf2\xec\xf9\x23\xe1\xd9\xf3\x03\x14\xe0\x97\x34\x9c\xfa\xc4\x15\xd7\xe7\x9f\x0d\x4c\xbd\xb1\x94\xfc\x50\x2d\x73\xbd\xf3\xb0\xd0\x1b\x22\x12\x2a\x6d\x76\x93\x25\x4a\xea\xb1\x52\xd3\x94\xe5\xf5\x5b\xac\x87\xca\xe4\xbb\x3d\x9d\xde\x0d\x21\xae\xdb\x4e\xc2\xe5\x9c\xf8\x29\x2f\xc3\x99\x62\x6c\x21\xae\x9c\x00\x5d\xf1\x04\x5b\xb9\x0f\xd5\x20\xb5\x6a\x66\x64\x7d\x59\x4e\xbc\xd3\xa4\xb3\x42\x3f\x6e\xdb\xa9\xa5\x59\x62\x7c\x25\x14\x57\x8e\xcf\xee\x36\xa3\xf4\x5d\xd5\xca\xf4\x60\x8b\xd9\x7e\x7f\xa5\xd8\xe8\xbe\xf6\x8d\x3e\xca\xdd\xf5\x0d\x5c\x58\x0b\x98\xa7\x98\xe8\xcd\x42\xa2\x0f\x97\xee\x07\x76\x48\xfc\x14\xdb\x6b\xb3\x52\x35\xa5\x71\x77\xeb\xe2\x7b\x11\x9a\x19\xa2\x5a\xda\x9c\x93\x6f\xab\x34\xa7\xcc\xc3\xec\xc5\x62\x95\x01\x30\x62\xee\xbc\x5b\x13\x73\x74\x7d\x1a\x7b\x93\x88\xd1\x0b\xe2\x61\x4b\x8a\x79\x63\xe2\x35\x8f\xa3\x88\x32\xa9\x27\x0a\x0c\x64\x60\x6a\x96\xc3\x97\xb2\xd5\x87\x52\xa3\x6b\x2f\x8b\xdd\xa1\xe3\x74\x6b\x95\x58\xe3\x8b\xbd\xd6\xc8\xc2\xf7\xd4\xea\x02\x27\x8a\x2b\x65\x77\xe4\x0c\xea\xc9\x7a\xb2\xfc\x9a\x49\x9b\x4d\xb2\x7f\x32\x60\xf7\x60\xc0\x5a\x58\x17\x75\xb5\x62\x9d\xa9\x50\xf4\xb5\x4d\x4d\xd2\x5d\xef\xaa\x70\xed\xb4\x6e\x63\x82\x74\x50\xfc\xa1\x18\xa2\x94\xb2\x7b\xb3\x47\x9a\x1d\x4f\xd6\xe8\xc9\x1a\x65\x7f\xdf\xcd\x1a\x2d\x39\x2e\x2d\x36\xbe\x27\xdf\x2b\x0d\x46\x4e\xc4\x22\xaa\x35\x79\x85\x46\x7c\xfd\xcf\xa2\x09\xfc\x4f\xfa\x85\x9e\xeb\xd5\x7b\x66\x2d\x8d\x60\x29\xfb\x4d\x8d\x05\x28\xf4\x40\x92\xc4\x61\x4a\x7c\x81\x99\x8e\x90\x96\x8c\x8c\x6c\xa4\x07\xaf\xb1\x90\x69\x20\xfa\x58\xc2\x7c\xb1\x28\x18\xcc\xdd\x30\x31\x12\x77\x1b\xd2\x68\x30\x99\xb7\x48\x38\x7c\xcf\x79\x77\x94\x52\xa0\x08\x28\xf0\xb8\x12\x28\x79\xda\xf8\x2f\xdd\xf8\x3f\xed\x5f\xaf\x6f\x70\x49\x38\x86\x08\x89\xb9\x01\x5f\x27\x86\x14\x4d\x95\xf1\xb3\xf5\x00\xbf\x8e\x21\x3a\xed\x83\x0b\x46\xc2\x59\x9d\x84\xf6\x5f\xb5\x74\xdf\x5a\xe0\x5b\x99\xd3\xb7\x8e\xed\x01\x0a\xf0\x32\x7c\x73\xd3\x62\x5b\x14\x92\xf8\xcb\x04\xb9\x2e\x8d\x43\x51\xf5\x7d\x57\xcd\xe1\x71\x7d\x82\x43\x31\x29\xcc\xfd\x9c\xee\x29\xf2\xf9\x6d\x10\x9e\x8d\x92\x51\x9f\x9c\xdb\x25\x74\x80\xa0\x70\x8a\x81\x61\xc1\x08\xbe\xc0\x0d\x77\xa1\x2b\x2e\xf2\xf7\x33\xbc\x1a\xe5\x5d\x8d\x71\xe3\x15\xf4\xea\xb2\x53\x24\x97\xd7\x7b\xc5\x0f\xd5\x9c\xdc\x67\xca\x40\x77\xe4\x6c\x3c\x12\x26\x3d\xac\xe8\x6c\x65\x3b\xf1\x50\x19\xf7\x18\x2e\xad\x96\x4b\x40\xa4\xbd\x6a\xbc\xdf\xa2\xb9\xa8\x2d\x3f\x81\x9a\x6d\x84\x99\xbd\xb9\x3c\xb3\xf1\xa8\x64\x55\xcb\x27\x61\xdf\x21\xcd\xb1\x48\xb6\x35\x13\xae\x4e\x0d\x78\x4b\x1d\xcb\x44\x6f\x1d\xeb\xda\x49\x83\x0f\x65\x65\x69\x3f\x6b\x12\x8d\x49\xa4\xbd\xf2\xcc\x29\x0e\xbb\x6c\x12\x95\x75\x2b\x49\x9d\x79\x5a\xc9\x9e\x56\xb2\x95\x57\xb2\xb7\x4b\xdd\xa2\xa7\x85\xeb\xf6\x16\x2e\x4b\xd6\x7f\x71\xea\xb7\x5b\xe0\x2c\x29\x2f\x25\xf9\xb5\xdc\xb3\xd8\xeb\x22\xdd\x30\xa6\xf7\xd7\x30\xe8\x96\x71\x56\x32\xe2\x2f\x16\xfb\x4b\x33\x2f\x73\xcf\xa3\xbc\x09\x5b\xf5\x62\xed\x32\xa7\xc7\xb8\x00\xdb\x56\xb7\xb2\x9d\x53\x3d\x6e\x59\xdb\x37\x58\xd8\x9a\x25\xe6\xb6\x52\xa0\xcd\xb6\xed\xcc\x6e\x68\xcd\xc8\x85\xb4\xd8\x69\x57\xb3\xe6\xc8\x9d\x28\xe6\xe8\x81\x58\xb7\xc6\xca\x1c\x4f\x4b\xfa\x5f\x6b\x49\xbf\x01\x93\x1e\xfa\xe6\x14\xfe\x84\xff\xfc\x75\x17\x6d\x6d\x90\x6e\x6c\x5c\xf3\x82\x0a\x75\xd6\xb5\xf5\xf2\xbd\xce\x30\xc7\x62\xe2\x32\xec\xe1\x50\x10\xe4\x5b\x6e\x1b\x3e\xad\xe8\x72\x45\xef\x29\x4e\xdd\xf1\xe6\xec\x50\x8e\x01\x86\x34\x9e\x6c\xf8\x93\x0d\x7f\xb2\xe1\x0f\xc9\x86\x2b\x33\x50\x9c\xd5\x2f\x19\xf6\xea\x0a\xcc\xd6\x3b\xc8\x1c\x0b\x9e\x5e\x0b\x49\xa7\x3b\x4c\x29\x6b\x30\xeb\x3f\xc8\xff\xc3\xf1\x1c\x73\x0c\x88\xe5\xd7\xab\x7a\x53\xe4\x92\x70\x06\x0c\xfb\xea\x1a\x54\x56\xec\x3c\xe9\xb3\xa4\xb6\xed\x7a\x80\x05\x23\x2e\x5f\x57\x47\x4b\x13\x86\xc2\x19\xae\xec\xeb\x2a\x11\xcf\xa4\x53\xe2\x7b\x93\x00\x73\xcc\x08\xe6\xa0\xba\xeb\x53\x2a\x89\xb8\x3e\x9f\xcf\xf6\x23\xe5\x9d\xc6\x3b\x0d\xe5\xc5\xe2\x50\x76\xfb\x68\x9c\x6d\xdd\x75\xf6\xd5\x2f\x47\xef\x0f\x00\x31\x86\x16\x40\xa7\xf0\x81\xd1\x00\x8b\x39\x8e\x73\xc2\xe8\xe9\x57\xec\x0a\x0e\x53\x46\x03\xa0\xa7\x52\x28\x48\x50\x46\xe2\xe0\x3e\x2c\x4c\xc2\xa8\x9c\x4d\x4f\x69\x59\x4f\x69\x59\xd9\xdf\xc3\x4c\xcb\xaa\x6d\xec\xc5\xda\x08\xac\xd0\x85\x84\x42\x4e\x40\x7f\x85\x2e\x3a\xf7\x87\x37\x57\xd7\xb0\x58\xc0\x15\x6d\x9f\xce\x3d\x12\xab\x9b\x3c\x9d\xf4\x23\x9e\x8c\xde\x32\xa3\x67\x32\xea\xc9\xec\x3d\x99\xbd\xec\xef\x91\x99\xbd\x6b\x18\xa4\x29\xf6\xa4\xf5\x68\xe1\x8f\x21\xdf\xcf\x66\x31\x09\x81\xbb\x0c\x45\x58\xbd\x94\x33\xa5\x2c\x40\x22\xf1\x2d\x75\x84\xf4\x4c\x67\x4d\x7a\x36\x13\x95\x0e\x99\x4c\xbe\xef\x64\x99\xb4\xd1\x34\x08\x40\xa6\x79\x12\xf8\x4a\x24\x74\x2c\x53\x4b\xd9\x74\x3d\xf2\x11\x69\xad\x90\xd6\xcc\xa7\xee\xa8\x09\xed\xc7\x75\x3f\xf5\x7b\xd6\xee\x7b\xb2\xc8\x6d\x2c\xf2\xa8\x74\x72\x60\x29\x6c\x45\x3c\xb5\x87\x57\x25\xe8\x1e\x05\x87\x6e\xb5\x54\xc5\xd3\x9a\x75\xb7\x6b\xd6\xb3\xfc\x27\xd9\x33\xa1\x45\x03\x79\xaf\x7c\xc0\x43\x3c\xc5\x0c\x87\x6e\x86\xa6\x36\x93\xda\x41\x4c\x87\x67\x72\xe5\x10\xc4\xa4\x93\x78\x26\x5d\x56\xdb\x7a\x46\xc2\xe5\x8d\xe6\x92\x88\xa6\x46\xd2\x13\x1c\x67\xeb\x4e\x92\x1c\x64\x70\x41\x8e\x62\x7c\x8c\xd0\x0c\x1b\x1f\x39\xf9\x66\x7e\x14\x54\x20\xdf\xf8\x4c\x04\x0e\xf8\x6a\x84\xb7\xa2\x4a\x62\x51\x6d\x24\x37\x37\x33\x23\xc7\x59\x22\xb7\xbc\x95\xc2\x79\x79\x33\x45\x4a\xb5\x99\xda\x05\x18\xdf\x56\x9a\x81\x55\x8f\x52\xad\x2f\x29\x89\xf6\x82\xd4\x54\x48\x61\x20\xdf\x7f\x3f\x5d\xa6\x96\x8d\xe0\x12\xd1\x54\xd9\x5f\x27\x02\x50\xf3\xde\xab\xcc\xac\x9a\xdc\x66\xa9\x37\xc8\x62\x05\x6a\x9b\x67\x7e\xd2\xa4\xa8\xe5\xd6\x4e\xd9\x13\x57\xd7\x62\x88\xec\x78\x03\x2e\x58\xa4\x59\x27\xf8\xda\xe6\xcd\x0a\xa0\xc8\xd3\x18\x9a\x55\x3e\xbe\x93\xf4\xab\x13\x5e\x37\x67\x18\xc5\x62\x8e\x43\x91\x58\xf9\x09\x0e\xa5\x0f\xec\x95\x9a\x05\xb1\x2f\xc8\x04\x7d\x6b\xc1\x49\xae\x1e\x84\x2a\xf3\xa6\xb0\x1c\x75\x7e\x43\x7e\x8c\xf9\x18\xfe\x40\x49\xd9\xac\x35\x88\x18\x8e\x90\xd4\x85\x35\x7d\x25\x80\x13\x1a\xaa\x4f\x0c\x23\x6f\xb1\x06\x53\xf5\xea\xd4\x1a\x78\x38\xfb\x79\x4d\x1f\x10\x92\x70\xf6\x6f\xe8\xb4\x55\xc9\xe2\x1d\x8d\x66\x34\xd3\x7b\x0b\xea\x0a\x17\xc4\x49\x3d\x60\x0f\x47\x3e\x5d\xf4\xe1\x35\x65\xe9\xb2\x05\xbb\x9f\x8f\x5a\x63\x90\xf2\xd2\xae\x6d\xd5\x72\x9f\x90\x5c\x8d\x68\xc3\xd2\xec\xe9\x4f\xe3\x92\x6f\x52\x85\xd7\x2d\x5d\xb8\x28\x10\x30\x86\x98\xf7\x30\xe2\xa2\x37\x50\xfb\x9e\x55\xe8\x51\xb5\xdb\x5b\x9b\x04\x75\xfd\xa2\x6d\xe3\x53\x4a\x05\x17\x0c\x45\x13\xfd\x32\xe6\x64\x6e\x9c\xb3\x2e\xed\x9d\xa4\x6a\x4e\x50\xa5\x8b\xde\x19\x8d\xc1\x43\x02\xf7\x04\x09\x70\x5b\x90\x49\x15\xf7\xdb\x04\xa9\x15\x7b\xb2\xa2\x65\x4d\x1f\x49\x6d\xdb\xbe\x70\xab\xf2\x7a\xbd\x26\x2b\x89\xae\xce\xb0\xb4\xd7\x7a\xb5\xe7\x9e\x70\x41\x19\x9a\xe1\x49\x79\x89\x6f\x1c\x5c\x36\x6e\xb3\xe4\xe4\x74\xce\x18\xe6\x7c\x22\xe6\x8c\xc6\xb3\x79\x14\x8b\x49\x84\xd9\x84\x63\xb7\x35\x08\x7c\x63\x08\xca\x3d\x99\x04\xe8\x6a\xe2\xd2\x30\xc4\xaa\x76\x70\xcd\x92\x44\x0a\xcf\x0c\x01\xc8\x4e\x11\x62\x82\xac\xd8\xc7\x43\x02\x4d\x18\x96\x7e\xbf\x14\x51\x84\x19\xa1\xed\xb9\x56\x44\x75\x82\x84\xc0\x41\x24\x78\x33\xe1\x45\x34\xac\xcf\x29\xd9\x56\xbe\xa6\x1a\xaf\xd5\x45\xf5\xae\xbd\x08\x2b\xda\xca\x9f\x85\x4e\x19\x8f\xa2\x1d\x55\xfe\x2c\x74\x06\x9d\x8a\xbe\x56\xbf\xd5\xfe\x6a\xe5\x6b\xe9\x7b\x94\x79\x7b\x93\xb2\xb8\x77\xeb\x12\x95\xd8\x9f\xff\x35\x0b\xc2\xc4\x59\x93\x5f\x7a\xcf\xf7\x3b\xf9\x4d\x4d\x92\xde\xfd\xb0\x9f\x20\x55\x12\x90\xfc\xf1\xa2\x24\xb5\xb9\x46\xcb\x12\xc7\xec\x94\xdc\x71\xdf\xaf\x99\xfb\x3d\x0d\x59\xf7\x2e\x2f\xcf\x4d\x23\xac\xd7\x75\x31\x55\xb6\xac\xab\xf5\xfb\x85\x5a\x04\xbf\x97\x72\x58\xc5\x68\x29\x34\x9e\x42\x2e\x5e\xe8\x50\x40\x94\x97\x23\xf2\x82\x9e\x70\x4a\xbd\x05\x70\xac\xef\x64\x26\x0c\x83\x0f\xef\x8f\x8e\x13\x18\xb6\x1d\xb3\x5c\x10\x9f\x99\xa4\x2f\xdd\xf3\xd6\x7b\x9f\x95\x4a\xa3\xa5\xeb\xb1\x97\xea\xb5\xf6\xbc\x7a\xa3\xeb\xc7\x5c\xc8\xef\x13\x87\x2f\x2d\xe9\x4a\xc2\xca\xce\xb5\x64\xba\x6d\xfe\x67\xe9\xca\x8b\xd0\xf5\x36\x05\x55\x77\x87\xe4\x7f\x5d\x1a\x4e\xc9\x2c\xb6\xa2\xa0\x2f\xb1\x2a\xb0\xbb\xff\xaa\x8c\x5e\x5e\xda\xcb\x5e\x44\x61\xe8\xae\xa4\x3c\x34\xae\x0b\x17\x46\xea\xc3\xbe\x80\x20\xe6\x42\xa2\xc3\x93\xcb\x14\x3e\xbd\xc4\xac\xe7\x22\x8e\x01\xf9\xd1\x1c\x85\x71\x80\x99\xf4\x76\xe7\x88\x21\x57\x60\xc6\x81\x32\xe8\x76\x7b\xdd\xee\x9a\xdc\x9c\xb0\x24\xfd\x19\x85\xba\xfd\x29\x16\x66\xeb\x35\x55\xe7\x00\xa7\xcf\x58\xa4\xad\x2a\x50\x75\x3b\x17\x85\x2a\xea\x78\x8a\xc1\xa7\xe1\x4c\x32\x63\x8e\x42\xd8\x18\x1a\xc3\xf7\xbb\xcb\x24\x52\xf5\xef\x2d\x75\x67\x65\x93\x5b\xd4\x82\x36\xfe\x59\x01\x8b\xcf\x73\xac\x6a\x15\xe7\x2b\x7e\x05\x06\x10\x0e\x09\x18\xc9\xf3\x90\x8a\x3e\xec\x4f\x81\x63\x91\xaa\xd2\x5a\x63\x77\x1a\x1a\xa4\x65\x25\x2a\xf2\x2d\x8d\x9e\x81\x80\x2f\x30\x5b\xc0\x26\x04\x24\x8c\x05\xe6\x7d\xc5\x20\x0f\x4f\x51\xec\x0b\xb8\x90\xfb\x20\x89\x48\xe9\xee\x7a\x9d\x9f\x19\xc6\xbe\x2f\x31\x2e\x5d\x76\x8f\x7c\x54\x2f\x10\x7d\x1a\x24\x9b\xe8\x83\x97\xe4\xc0\x88\x4e\xe1\x1f\x05\x8f\xf9\x9f\xfd\x7f\x24\x9e\xe8\x3f\x9b\xc4\xb1\xa4\xb4\x45\xed\x4a\xb7\xca\x92\x55\xac\xaa\xb2\x92\xfb\x50\x8f\x9e\xc4\x6e\x15\x87\xa2\x11\x87\xdb\x5f\x3c\x56\x2d\x24\xd2\x5d\x22\x0d\xeb\x72\xd2\x3d\x6a\x2a\xad\xd2\x5d\x6d\x79\x20\xf5\xd3\xaf\xfb\x29\x24\xe7\x52\xb3\x55\xee\xdb\x94\xe8\xf2\xde\x96\xe9\x22\x87\x5a\x6e\x72\x54\xb5\x97\xfa\xc1\x2a\x45\x71\x33\xf0\xba\x4c\x0c\xba\x40\xc4\x4f\x0f\x4b\xf5\x62\x61\xa5\xbb\x5e\xb0\x16\xa1\x5a\x05\xba\x8a\x30\x8f\xb2\x72\x44\xd5\xef\xdb\x09\xef\xc8\x28\x68\x74\x77\x32\x23\xdc\xc6\xd5\xe5\x42\x6b\xb7\x4d\x2d\xa2\xf0\x2e\x29\x09\x9f\xf4\x85\xbc\x2f\x44\x98\x01\xc7\x2e\x0d\x3d\x43\x9e\xd2\x50\xb7\x47\xb0\x62\x7c\x56\x13\xd6\x8b\x85\xc0\x5c\x45\xae\xf6\x05\x0e\x72\xf0\xad\x36\xd3\x76\x3a\xf1\x23\x22\x73\xe9\x8e\xdf\x4e\x22\x0a\x54\x96\xab\x54\x25\x09\xc0\x58\x52\xf9\xb5\x29\x2c\x1f\x82\x58\xa2\x03\xe5\x08\x8c\x1d\x39\xd9\x09\x92\xa0\xcd\x43\xe3\x77\x7d\x98\xa4\x1d\xa3\xf3\xbe\x77\xc9\xe7\x6a\x14\xa6\x81\xd3\x59\x37\xd0\xdd\xae\x8d\x58\x79\x93\xd0\x3e\xb6\x63\xc1\x8e\xc4\x81\xe9\xe8\xa5\xbd\x6f\x63\x2a\x96\x19\x78\x1e\x53\x81\x24\xaa\x3c\x0e\x1a\xbc\xd7\xee\x47\xd9\x0e\xd2\x76\xd9\xcb\x17\x37\x19\xb4\x1c\x4a\xb5\x0d\x28\xdb\xe8\x93\x83\xd5\x46\x2c\x89\xc3\x45\x11\x72\x89\x58\xb4\x20\xf4\x95\x54\x0b\xe9\x9b\xe2\x6c\x77\x90\xf6\xbe\x0d\xf2\x97\xcd\xb6\x15\x4f\xbf\x4f\x65\xe7\xea\x61\xab\x7e\xc0\x44\x7d\x5d\xa9\xf4\x7b\x5f\x81\xbd\x0a\x22\xf7\x1f\xd9\x2b\xa0\xf4\x58\x42\x7b\x05\xa4\x3b\xb9\x8c\xf3\xea\xa9\xf7\x2a\xe1\x1c\x8d\x07\x22\xdf\xda\x97\xe7\x1e\xae\x74\x35\xca\x9d\xea\xfc\xb5\x7b\xe1\x2f\x8b\x07\x93\xdd\x06\x7b\x51\x4e\x1a\x29\x02\xda\x0f\x3d\xe2\xaa\x22\x0d\x72\x63\xa4\x6c\x6f\xea\x70\x6b\x45\xe8\xc3\xe7\x24\x9a\xd0\xed\x16\x10\xeb\x76\xc1\x27\xe1\x59\x0b\x1f\xfc\x3a\x5b\xb4\x64\xf0\xa5\xc0\x3d\xc2\x23\x1f\x2d\x2a\x27\x6d\xc5\x61\xcc\x72\x7a\xa5\x28\xa1\xdc\x8d\x25\x40\x20\x8a\x59\x44\x39\x6e\x11\x7f\x6a\x1e\xee\xe7\x38\x40\x21\x4c\x19\xc1\xa1\xe7\x2f\x2c\xd4\x15\x71\x58\x53\x48\xa4\x07\xe3\x27\xe8\x92\x9f\x2c\xc7\x60\x59\xf0\xa9\x9b\x46\x9f\x2c\x34\x1b\x41\x27\x45\xbe\x3a\x9e\x27\xe1\x0c\x50\x08\xef\x8f\x5e\x65\xc1\xc3\x2a\x12\xc5\x68\x90\x2d\xc2\x6b\x66\x43\x18\x9a\x6d\x57\xe3\x57\xf9\x27\xc9\x1a\x94\x06\xed\xd4\xbf\xdd\xfb\xd3\x71\x8d\x73\xb7\xfb\xe8\x94\x3b\xe1\x9f\x4d\xa9\x4b\x5a\x76\xd0\x87\xdf\x08\x9b\x91\x90\xa0\xdb\xd6\xb6\x04\x89\xdb\xd2\x32\x3d\x98\x8a\x55\x96\xeb\x4c\x66\xa5\x31\x27\xf5\x91\xb2\x6a\xb4\xdc\x5a\xd0\x37\xaf\xb2\x99\x7a\x7a\x9a\x0c\x0b\x7a\x2d\x02\x33\x8d\x2e\x69\x1b\x4d\xbd\xcc\xf9\xc9\x54\x70\x36\xf3\x48\x7d\x3c\xd5\x3b\xf4\x3b\x0e\x1f\xd9\x17\x2b\x3d\x33\x5e\x26\xc8\xc8\x35\x5f\xba\xb0\x9d\x96\x06\x41\x7f\x53\x27\x35\xa3\x49\x4a\xad\x76\xa0\x0b\x77\x61\x6b\xc2\x52\xc9\x85\xd6\xdd\x62\xd1\x31\x20\x21\xbc\xdb\x3d\xea\x1d\x1d\xbd\xcf\xce\xaf\xb4\xf8\x5f\x26\x9e\xbe\xba\xb3\x50\x08\xaa\x77\xaf\xe3\x4b\xdd\x5e\x76\x61\x35\x09\xa3\x48\xa9\xce\xeb\x81\x19\x0e\xd5\x1d\x0a\x0f\xe2\xd4\xcc\xd4\x94\x4c\x2d\x27\x0e\xaf\x94\x68\x54\x1c\xbb\x35\x28\xb3\xdb\xed\x40\xcc\x0a\xc3\xb6\x4f\x66\xd2\x3d\x38\x76\x59\xb5\xac\xc2\x2d\xe5\x66\x81\xca\xae\xc3\x72\xce\x56\x8b\xff\xe6\xf9\x54\xa7\x8b\xfb\x4b\xc1\x5a\x3d\x61\xc4\x5a\x51\xa2\x63\x99\x8a\xa5\x84\xcc\xd2\x8c\xb4\x9f\x1a\x0b\x9a\x90\x58\xbd\x85\xde\x6d\xb0\x22\xab\x1f\x1c\xaf\x76\x6a\xda\x30\x67\xec\x4b\xb3\x5d\xc1\x8b\x83\xec\x9a\x9f\x33\x4e\xac\x36\x54\x45\x7c\x2b\x88\xce\x96\xf4\x63\x37\xe0\x76\x11\xf2\x5c\x84\x28\xbd\xd0\x65\x2e\x3a\xf9\xa2\x44\xc2\x64\xb9\x5c\xf5\x28\xa0\x2e\xc9\xae\x88\x88\x65\xec\x76\xf1\xb8\x34\x08\x94\x3c\xd0\x5a\x3f\xc2\xd4\x47\x33\x20\x7a\xf9\x95\x3e\xca\xa5\xe9\x3d\xa7\x54\xa6\x12\x2c\x32\x21\x79\x7b\x33\xf7\x7a\x92\xc1\xda\xf9\x35\x35\xd6\x23\x0b\xf6\x4d\x96\x9c\x3b\xa5\x87\x4e\x79\x74\xd0\x7a\xfc\x64\x7b\xcb\x56\x21\x9e\x39\x3a\x92\xe4\x10\x04\x3a\x53\x2e\x5a\xba\x8c\xc6\x8c\x61\xf9\xdf\x94\x05\xf9\x93\x06\xc8\x07\x9f\x04\x44\xf0\xdb\x72\x90\x6c\xd3\xbe\xa0\x1f\xc6\xf7\x25\xf6\x58\x6c\x53\xb3\x66\xff\x6d\xd7\xf8\xda\x65\xb4\x88\x80\x6e\xf6\x5d\x7c\x8a\x96\x56\xb8\x71\x14\xeb\xa2\x5d\x1c\x46\x35\xb9\xe9\x38\xd7\x5e\xee\xab\xe2\xb5\x54\xc8\xd5\x3b\x0f\x5d\x70\xa5\xbd\x40\xaf\xed\x2f\xb4\xc0\x49\x9a\x05\x55\x78\x45\xa0\x20\xba\x0d\xe7\xaf\x91\xb3\x26\x3a\x5e\x31\x32\x50\x2b\xb4\xea\xa4\xbf\x95\xe4\x8f\x24\xbc\x59\x85\x5e\x0d\x4f\x5a\x92\x0e\x57\xa8\xd7\x95\x9a\xa9\x15\x82\x95\xe5\x60\x47\x23\x5f\xef\x35\xb2\x69\x27\xd5\x64\x61\xdd\x25\x9b\xc2\xcd\x3a\x28\xdd\x97\xfb\xa1\x50\x92\x28\xbd\xd0\x9d\x96\x26\xfa\x41\xb5\xb1\x16\xb3\xb9\x4d\xd5\xb0\x0e\x60\xc9\x6a\x1d\x84\xa7\xd1\xd1\xb6\xf3\xb3\x17\x7f\xc0\x23\xdf\x11\x74\xe7\xeb\xd1\x6c\xf8\xf2\xed\xb7\x69\xdc\x42\x97\x1a\x35\xa9\x82\xc2\x9d\x29\xd1\x23\xd1\xb7\x9c\x13\x89\xaf\x9b\x7d\x5e\xf1\x8c\x4d\xeb\x54\xf5\x90\xad\xa2\x21\xc8\xf3\xd4\xf1\x36\xf2\x3f\xd4\x30\xda\xca\xa9\x0b\x7d\x9b\xeb\x3a\x8e\x53\xd3\xb9\xbe\x06\xab\xc5\x5f\x1c\xa2\x25\xdd\x99\xad\xaf\xa2\x56\x3e\xd0\xcd\xd7\x17\x12\x8a\xad\x51\x91\xb4\xc6\xe3\xc9\x62\x6f\x8f\xc6\xa7\x3e\x6e\x70\x01\x15\x40\x73\x4e\x97\x6b\xb5\xdc\xc1\xac\x2e\x0f\x71\x2f\xf3\xda\x44\xe2\xef\x3e\xb3\x4d\x5e\x74\x4c\x65\x78\xad\x4b\x89\x10\x1a\x1e\x62\x1e\xfb\xa2\xa8\xf0\x06\x19\x26\x84\x07\x66\x0d\x1e\xf6\xac\x53\xdb\xc9\x4f\xea\x16\x5f\x29\xde\xd3\x92\x7d\x3f\xa8\x7d\x73\x48\x2f\xb5\x9b\xae\x32\x8d\xe7\x18\x68\xe8\x2f\x8c\xa8\xfb\x94\x60\x5f\x1f\x14\xe8\x1b\x83\x59\xf7\x8a\x6f\x5f\xa3\xa1\x35\x69\xc9\x7f\xa1\xac\xed\x0a\x0f\x96\xe6\x66\x3f\xab\xd6\x6f\xc8\x67\xbc\x7e\xd6\x2b\x2b\x8e\x52\x49\xa0\xd7\x2f\xa6\x31\xec\x52\x96\x55\xb3\x2c\x15\xaf\xb0\x88\xa2\xf4\x74\x9a\xe5\x4d\x95\xb4\x34\x5b\x11\x8f\xf4\x5b\x03\x8c\xf9\x0e\x59\x05\x3b\x1f\x87\x33\x31\x57\x7b\x03\x12\xa8\x20\x4c\xc2\x26\xa5\x43\x97\x73\xe2\xce\xa5\x30\x98\xaa\xfe\xa3\xd8\x1d\x14\xaa\x0d\x59\x1f\x7b\xb1\xd3\x57\x9e\x84\xf6\x29\x98\x1d\x51\x6d\xe6\x86\x83\x84\x24\x88\x83\x31\x0c\xf2\xaf\x74\x2a\xda\x18\x46\x1b\x43\x27\xf9\xb6\x5a\xc9\xa3\xcc\x22\xc8\xa6\x78\x02\x3d\xad\x55\x57\x92\x65\xf2\x6d\x5b\x1e\xa6\xed\x55\x35\x27\x95\x64\xc6\xe1\x14\x8b\x4b\x8c\x43\x9d\x2b\x97\xd5\xf8\xbc\x5b\x8e\x6d\x38\xad\x58\x36\x70\x76\x9c\x7a\x9e\x95\x59\x62\xf0\x2c\x81\x9f\x14\xc7\x2a\xf2\x2c\xf9\xb2\x0d\xcb\xd2\xc7\x48\xd2\x1d\x87\xa0\x30\xc5\xc2\x9d\xf7\xe1\xb5\xfc\x4f\xa1\x3e\x96\x0a\xa6\xe1\x20\x12\x8b\xbe\xee\x87\x43\xa1\x8a\x97\x22\x96\x4f\x7c\x81\x59\x88\xd2\x3e\x0a\x9f\x6c\x92\xdb\xf9\x5a\x5c\x68\x6b\xca\x6e\x54\x22\xd5\x09\x97\xd3\x1a\x5a\x66\x81\x10\xcd\x03\xa3\x70\x49\x23\x03\x3e\xa0\x99\x54\x1a\x0f\x5f\x55\x54\xc2\x3c\x97\x6d\x61\x25\xaa\xe2\x2b\x97\x2d\x49\x44\x97\x26\x04\x99\xa9\xb4\x1a\x69\xa3\xbc\x4a\x23\xd2\x07\x6a\x0d\x94\x72\x53\xfc\x92\xba\x8e\x91\x3b\x37\x89\xbe\x45\x32\xca\x29\xbf\x19\x19\x8e\xa3\x09\xa1\xcc\xc3\xec\xc5\xc2\x1a\x96\xfc\xff\xbd\xac\xe7\x91\x2e\x41\x90\xe4\x2c\xa8\x4e\xea\x81\x59\x46\x04\x66\x04\xe9\x1b\x3c\x7c\x11\x0a\x74\x95\x25\x33\x64\xa6\x1e\x08\x37\x10\x0a\x88\x8f\x98\xce\x19\x2d\x76\xc1\x70\x92\x02\x3e\x01\xd7\x47\x31\x57\x81\x38\x14\xc2\xd1\xc7\xb7\xaa\x2a\x05\x0e\x70\x28\xf2\x75\x67\x4f\xf2\x4d\x17\xa1\x4c\x42\xc9\xaa\xbf\x8e\x5c\xa1\x70\x91\x82\x9d\x52\xdf\xa7\x97\x72\x73\x7e\x72\x66\xdc\x31\xe4\x27\x7a\x95\xe7\xe3\x67\x19\xc8\x1f\xed\x15\x0b\x8c\xdf\x6b\x9e\x08\xfd\x31\xcd\x1f\x30\x5f\xa4\xfc\xd1\x08\x87\x19\x5f\xce\x19\x9e\x1a\x1f\x0b\x1d\xec\x11\xe6\x1f\xab\xf5\x3b\x7e\x34\xcf\xa0\xe4\x47\xca\x66\x28\x24\x3c\xad\xd6\x62\xfe\x22\x5d\x16\xe3\xf3\xd2\x92\x21\x3f\x96\x9f\x09\xfe\x31\x29\x0a\x62\x7c\x91\x17\x52\x30\xbe\x4c\x8a\x1a\xe4\xfc\x34\x2a\x54\xac\x19\xeb\x9f\x34\x4d\x45\x77\x83\x9b\xb2\x13\x73\x4c\x98\xa2\x6f\x0d\xa4\x16\x14\x85\xa8\x75\xc6\x10\xda\xc9\xc9\x09\x3f\xcf\x6b\x19\xa9\x08\x2e\xe2\xae\xf9\x7b\xde\xf8\x78\x75\x24\x60\x82\x42\x6f\x92\x85\x45\x25\xdd\x37\xc1\x6b\xcd\xd0\x8a\x7a\x3c\xf7\xb5\xee\x9a\x93\x28\xec\x8a\x34\xff\xc8\x5b\x93\x9e\x1e\xd1\x6d\xb2\x1b\x73\xca\xc0\xaf\xc9\xef\x72\xd1\xe9\xb3\x20\xb9\x17\xd1\xc6\xde\xa0\x50\x22\xd4\xcf\x4c\x47\xe4\x53\xaf\xe8\xad\x56\xcd\x49\xc9\x5a\x80\x61\x51\x52\xea\x3a\x35\x46\x50\x5b\xc9\x04\xc0\x4d\x0d\x1d\x17\x0b\xe9\x54\xca\x75\x5c\x9b\x63\x8c\x98\x3b\xb7\x1b\xb1\xdc\x86\xa9\x46\xb9\xcd\x32\x74\xa2\xd9\x78\x2d\x31\x5a\xea\x4a\x67\xd1\x62\xe5\x63\x16\x2c\x17\xec\x4a\x5d\x49\x77\x17\x3c\x3d\xaa\xd3\xd8\x2b\xe9\x9c\x14\xcd\xcb\xc9\x1a\x9c\x48\xc6\xc9\xff\xaa\x59\x2c\xff\xa1\xe7\xe6\x89\xbe\xbf\x7a\xa2\x27\xe6\x49\x0e\x5b\x6e\x57\x11\x43\x82\x32\x2d\xf0\x93\x7f\xfc\x53\xf6\xfa\xe9\x44\xa9\xcc\xc9\xdb\xfd\x5f\xf7\x4e\x72\x1b\x9a\xf6\xfa\x4a\x49\x98\xb4\xdf\x3d\x78\x75\xa2\x61\xbf\x3f\x3c\xe9\xc3\xcf\xf4\x52\x7a\xfe\x6b\xb0\xa0\xb1\xb2\xb3\x92\x4a\x94\xba\x41\x92\xde\x81\x93\x74\x57\x77\x29\x13\x6a\x94\xec\x0d\x1e\xef\x65\xca\x64\x9b\x8a\xd5\xcd\x47\xf2\xbe\x91\x52\xab\x93\x60\xd1\x53\x96\x5b\xe3\x65\x1c\x6f\xaa\xec\xc4\xb6\x93\xb1\x38\x13\x7f\x82\x14\xaa\xbe\x08\x5c\x60\x3c\xfc\x04\xe8\x92\x9b\x9d\xff\x88\x7a\xff\x6e\x8f\x3a\xd2\x63\xa8\x63\x4e\x75\x65\x39\x29\x9d\x7c\x12\x2c\xae\x89\xae\x4f\xce\x30\x04\x8b\xff\x1a\x6e\xde\x89\xbd\x50\xd6\xb0\xba\x0b\xe4\x86\x1d\x41\x22\x3b\x0e\x82\x39\x52\x77\x41\x02\xc2\xb9\x3a\x94\xa1\xc0\xb1\x2e\xd1\xcf\x92\xfa\xa6\x86\xe8\x0f\xa8\xc0\xfd\x14\x41\xbd\x5e\xe7\xb5\x30\xa5\x1a\x27\x35\x0d\xd5\xd9\x6f\xda\xbb\xde\x2c\x25\xfe\x96\x52\xb3\x1a\x63\x63\x37\x2c\x16\xf7\xa8\x60\x37\xa0\x6c\xce\x5a\xa8\x48\xe7\xfa\x46\xcb\x9a\x6c\x90\xee\x9c\xaa\x5e\x40\x65\xbb\x64\xc9\x1b\x54\x7b\x00\xb5\x83\x28\xd8\xfd\xd3\x45\x0d\x9f\x5a\x60\xdd\x96\x95\x1e\xbe\xc0\x3e\x8d\xa4\x03\x54\x97\x44\x91\xf2\x36\x6b\x9a\x47\x26\x65\x0f\x0f\x31\x6f\x79\xe7\xb4\xa5\xec\x9b\x96\xe8\x55\xb9\x3d\x29\x32\x49\x8d\x5e\x93\x42\x3c\x86\x53\xf5\x6d\xf2\xa5\xfe\xf0\x3a\xd9\x05\xfe\xf2\x39\x2d\x11\xa1\xc9\x9f\x0b\x11\x3d\x2b\x93\xf8\xe9\xa8\x90\xc5\x9f\x82\x2f\x05\xba\x92\x3a\x00\xd0\xc9\x8a\x71\xe5\x24\x96\x2a\x47\x40\xc7\xd0\x9e\x54\xee\x9d\xf4\x65\xd4\x88\x88\xac\x6e\xca\xde\xa7\x95\x86\xc6\x71\xef\x12\xdf\xd6\xd0\x57\x91\x4f\x5c\x22\x8e\xc8\x37\x7c\xc7\x94\xcb\x41\x27\x6a\xae\xe5\x3f\xa9\x3b\xfc\xb9\xc8\xfb\x57\x83\x6a\xa5\x90\x66\xb4\x74\x6c\x9c\x1c\x7d\xd9\x3a\xfc\xb8\xf1\xcb\xaf\xfb\x3b\x1f\x9d\xf7\xc7\xc1\xd7\x8f\xaf\xbd\x0d\xea\xbe\x3e\x9c\xe5\x63\x25\x11\xf7\x12\x06\x4b\x4b\xb2\xac\xb7\x02\x9e\x94\xd4\x83\x8e\xaa\x85\xd7\x96\x33\x59\x9d\x8f\x72\x08\xb1\x9e\xd5\x3a\x3a\x09\x1d\x14\x91\x49\x52\x7e\x4b\x8b\xb5\x41\xdc\xf9\x4f\xf6\x6a\x6d\x66\xdb\xde\x80\xf0\xc5\x16\x3b\xdf\xf8\x7a\x46\x76\xce\x1d\x2a\x82\xaf\xe7\x53\x49\xee\x94\xcd\xfa\x28\x8a\x78\x3f\x38\xeb\x9d\x0a\x31\x73\xbe\x86\x83\x6d\x67\x1e\xf5\xaf\x36\xe3\x9d\x3e\x1f\xf4\x3d\x7c\xc1\xe7\x64\x2a\xfa\x94\x19\x8c\x31\xd2\x05\xa0\x33\x74\x86\x4e\x6f\xe0\xf4\x9c\xcd\xe3\xc1\x70\xbc\x39\x18\x0f\x47\x7d\x67\x73\x63\x30\x1a\xfe\x2b\xef\x61\x14\x70\xab\xf4\xd8\x1a\x6f\x6c\xf5\x37\xb6\x86\x43\x67\xc7\xe8\x91\x56\x5a\x83\xce\xb0\xbf\xd5\x77\xf2\x1f\x8a\xb6\x26\xb3\x41\x06\xa3\x6b\x62\xb5\x45\x79\xa4\xb5\xca\xe0\x6a\x60\x80\xae\xbd\x00\x0d\x9d\xe1\x3b\x83\xe7\xf5\x37\xa5\x4b\x0d\xed\x77\x8d\x61\xe0\x18\xd1\xaa\xe2\xf5\xd8\xca\x6f\xf6\x7b\xaa\xd0\xf9\x30\x18\xbd\xea\x14\x5a\x36\xdd\x1c\xcd\xc0\x9a\xd3\xf0\xb5\x2a\x82\xf7\x32\xc9\xd2\x38\x52\xfa\xfe\xb8\xa6\xa6\x2e\xe3\xf7\x34\x37\xbf\xeb\xdc\x2c\xd6\x4e\x84\x0e\x4a\x0a\xd4\x1a\xae\x68\x9a\xa9\x9b\x65\x00\x95\x05\x75\x0b\xd3\xb8\x55\x85\x94\x06\x3d\xce\x3c\x9a\x82\x51\x28\x04\x43\xff\x30\xfe\x0d\xf0\x67\xe1\x93\x06\x72\x35\x58\x2b\x7f\xdb\x60\x45\xfe\xec\xa8\x9b\xb7\x9d\x31\x6c\x0c\x46\x9b\xdb\xc3\x1d\xe7\x3f\xe5\xee\x0d\xa6\xa5\x45\xef\x1a\x7b\xb3\xe1\x38\x4e\xb9\x69\x5d\xad\x01\x63\x98\x81\xb3\xbd\xb1\x3d\x1a\xec\x38\xf2\xaf\x32\x96\xc5\x70\xb5\x19\xa4\x68\xc0\x6c\x1d\x96\xd9\xb1\x72\x9f\xd2\x65\x74\xa8\x88\xc4\xb8\x38\x0e\x1d\x36\xa7\x1c\x9d\x55\x06\xae\xde\xf5\x2e\xc3\x29\x30\x40\x87\xc1\x6d\x37\x81\x6b\x34\xce\x76\x9d\xb9\x53\xb4\xab\x36\x0f\xab\xf0\x5d\xe1\x2e\x17\x74\x76\x03\xf4\x8d\x86\xf0\x19\x9f\xa6\x29\x6c\x46\xdb\xea\x7c\xa9\x5e\x6b\x6d\x81\xaa\x79\xa7\x34\x43\xd4\x62\x27\x4b\xa8\x7d\x3a\x82\x3d\xc4\xc5\x1a\x18\x57\xc4\x9a\x70\x83\xa6\x8b\x58\xf0\x47\xbe\x81\xf8\xf7\xb3\xb2\xc0\xc6\x85\x59\x5a\x99\xa3\x45\x3b\x93\x03\xb2\x2a\x5e\x39\xd5\x5b\x9f\xc3\x96\x5a\x96\xd3\xaa\xe1\x8f\xce\xd5\xa0\xb3\x06\x9d\xab\xa1\x81\x1e\x18\xcf\x0b\xea\x6f\x9b\xf2\xea\x6b\x24\x91\x70\x33\x58\xf4\x50\x14\xf5\xb8\xc1\xc2\x62\x19\xe2\x72\xe2\xe5\x94\x32\x08\x16\x80\xa2\xc8\x76\xe5\xa2\xcd\x12\x5f\x59\xc8\x8b\x20\x5a\xad\xe8\xe9\x2a\x97\x3e\xe5\x3a\xa8\x68\xf7\x8d\x09\x83\x42\x3a\x32\x74\x8e\x76\x7b\x83\xa1\xfc\x5f\xe5\xe7\xe4\x0a\x8f\x04\x29\xff\x51\x5d\xe1\xe5\x5e\xa6\x17\x73\x73\x56\x1a\xb9\xb8\x8d\xbf\xa7\x4b\xe7\xa0\xe7\x8c\x7a\xce\xf6\xf1\x60\x6b\x3c\x1c\x8d\x9d\xc1\xff\x73\x36\xc7\x1b\x8e\x4d\x04\xc6\xd3\xe9\x7f\x13\x31\xdc\x0b\x9b\x4b\x99\xb1\x37\x61\x75\x35\xf3\xf4\x89\xe5\x85\x2c\xaa\x4a\x0a\x69\x0d\xb7\xab\x99\x50\x13\xb5\x6a\x4c\x26\x63\xc8\x3d\x6c\xcc\x26\xa7\x8c\x9e\x61\x26\x68\x44\xdc\xe4\x44\x7c\xa2\x1c\x93\x09\x09\x27\xc5\x87\x30\x40\x45\xbf\x82\x6f\x64\x42\xe8\x24\x39\xd2\x4b\x80\xf5\xca\x8f\xc8\x4a\x1f\x29\x22\xee\x18\x26\xe9\x4a\xcf\x26\x74\x3a\xe5\x58\xe4\x61\xd3\x6a\x6a\x65\xcf\x48\xb0\x82\xc1\xd6\x60\xb0\xb5\xed\x0c\xa5\x57\xe5\x14\x16\x87\x24\xde\xb5\x33\x1a\x6c\x8e\x96\xf5\xde\xaa\xed\xbd\xb9\xb3\xb3\xb3\xac\xf7\xf3\xda\xde\xdb\x5b\xc3\xa1\x29\x17\x4b\x0a\xe0\xe3\x95\xcc\x52\x29\x54\x24\x30\x72\x1c\xf5\x12\xf9\x52\xaf\x47\x5b\x01\x67\xa3\x62\x07\x8c\x27\x24\xa0\x79\xda\xab\x88\x38\x5f\x2f\x00\x51\x0f\x7d\x40\xe7\xd7\xdd\xd7\xbf\xee\x1e\xf5\xde\xbd\x79\x77\xdc\x2b\xfc\x9e\xed\xa2\x8e\x16\xa1\x3b\x67\x34\xa4\x31\x07\xe4\xa6\x29\x62\x21\x15\xb9\x63\xa4\x0f\x21\x10\x5f\x84\xee\x4f\xaa\x44\x6c\x76\x70\x60\x4c\x7a\xf3\xf1\x0f\xb9\x57\xff\xbc\x4f\x82\xf3\x37\x2e\x7b\x15\xbf\xdd\x1a\xa0\x4f\x57\xfb\xff\x3a\x7f\x71\x7c\x7e\x70\x98\x58\x9e\x91\xe3\xa4\x01\x80\x27\xfe\xd8\xf9\xb3\xaf\x0f\x3d\x5a\xcc\x20\x05\x72\x78\x0b\x2c\x1a\x36\x73\x68\x68\x63\x90\x8e\xe6\x80\xa0\x92\x6c\x8e\x0b\x67\x7a\x63\xf8\x14\xa6\x65\xbf\xd4\x6d\xbf\xc2\x36\x3d\x79\xd1\xb9\x1c\xe2\x18\x43\x71\xcc\x31\x2c\x1b\x22\x4f\xc5\x74\xa9\x1f\x07\xa1\x3e\x05\x93\xc0\x93\x43\x1b\xe8\x12\xaf\xdb\xcf\xb7\xee\x66\x3b\x75\x92\x39\x4e\xa2\x31\x6b\x49\x26\x41\x31\xa0\x93\x7e\xab\xe3\x3f\x7d\xf8\xa8\xcf\xa5\xb4\x7c\xc6\x40\x3c\xf8\x09\x06\x26\x73\xca\xd2\xf6\x3f\xbf\x7a\x13\x2f\x4e\xf7\xd9\x5e\x78\xc5\x76\x71\xb0\x3d\x1c\xcd\xce\xcf\xce\xc8\xab\x8b\x4c\xda\x4b\x9e\xa1\xb3\x4a\xbc\xea\x3c\xac\x2e\xf1\x41\xb3\xc4\x07\x16\x89\x07\x1a\x55\x95\x2b\x99\xeb\xfa\x38\x7b\x37\xf1\x26\x7c\x28\xbf\x93\x66\xa3\x7b\xfb\xe6\x64\x6f\x37\x52\xbd\x6d\x21\xfa\x38\xbf\x08\x8e\x3d\x60\x98\xd3\x98\xb9\x18\x3c\x8a\xd5\xd9\x29\xbe\xca\x72\xed\x47\xce\x48\x99\xfe\x25\xa7\x20\xf7\x47\x4a\x12\x8b\x4d\x28\xd0\xcf\xf4\x7a\x3f\x75\x07\xe4\xd7\x0d\x2f\xfe\xed\xcb\xfe\xc5\xc5\xe6\x97\x8b\xb7\xfe\xe2\xdb\x20\x78\x73\xb8\xf1\xcb\xe2\xfc\xa0\x9b\xbf\xb6\xd7\x60\xd2\xbe\xbc\xdf\x9e\x0d\x67\x5b\x3f\x1f\x7b\x9f\x7e\xfd\x84\x86\x67\xfc\xe7\x9d\xe1\xd9\xc7\x57\x1b\x8b\x94\x2f\xe5\x67\x02\xad\xa6\xfe\x16\x94\x7a\xd0\xac\xd4\x03\x9b\x52\xe7\x86\xea\x02\x33\x32\x5d\xc0\x2f\x9f\x8f\xf5\x63\x8c\x63\x38\x4c\xd3\x9a\x51\x2c\xe6\x94\x91\x6f\xc9\xf5\x42\xf5\x54\x63\x2b\xce\x6c\x7c\x9a\xef\xcd\x2f\x83\xdf\x5f\x44\x9f\x3f\x4c\xf7\x87\xfe\x01\x3e\x8b\xbc\xd1\xbf\x5e\xa5\x9c\xd9\x68\xc1\x99\xd1\xcd\x19\x33\x6a\xe4\xcb\xc8\xc6\x16\x8e\x19\x74\xa7\x94\xf6\x4e\x11\xeb\xa6\x4b\x5f\xca\x07\x6d\x94\x91\xeb\xea\x2a\xac\xd9\x45\xc7\x7e\x83\x09\xf8\xb2\xf1\x89\xec\xcd\xbf\x85\x06\x2f\xbe\x46\xde\xe8\xcb\xcb\x8c\x17\xef\xd0\x55\x92\x6a\x92\x46\x57\x0f\x75\x60\xa4\x05\x93\x36\x6f\xce\xa4\xcd\x46\x26\x6d\x2e\x67\xd2\x1c\x65\x17\xe9\x8d\xe4\x97\x30\x4b\xe6\xdc\x02\x94\x64\xd2\x70\xc1\x30\x0a\xa4\x2d\x8d\x43\x22\xf8\x52\xb6\x9d\x5d\x49\xb6\xfd\xf6\x01\xef\x0f\xe9\x01\xfe\xea\x6d\xfc\xfe\x22\xe3\xda\x31\x66\x01\x3f\xa0\x62\x37\x79\x65\xab\xcd\x5c\x1b\xde\xc2\x5c\x1b\x36\xcf\xb5\xa1\x85\x5f\xd9\x7c\x12\x12\x67\x98\xa3\x0b\x9c\x14\xd6\xc7\x21\xa4\xaf\x84\xd5\xf2\xe2\xec\xf7\x97\xdf\x3e\x2b\x16\xa4\xbc\x78\x7b\xf1\xfa\xf9\xd7\x77\x1f\xbf\xa4\xbc\x78\x7e\x80\x02\xfc\x92\x86\x53\x9f\xb8\x6d\xc2\x4e\x1b\x5b\x37\xe7\x83\x09\xc3\xc2\x07\xf3\xe7\xa2\x21\xce\x0a\x77\x2a\xa7\x85\x70\x40\xbe\x3a\xf9\x55\x85\x44\x6b\x99\xb0\x75\xf6\xc5\x91\x0a\xf1\x2d\xe7\xc6\x17\x3c\xf7\x36\xf6\x12\x93\x52\x7d\x48\xd3\x46\xf8\xf3\x9b\xd3\xfd\xbc\x91\xec\xe7\x56\x4b\x9b\x3c\x52\x96\x3e\x50\xda\x60\x38\xf1\x5e\x2a\xdb\xad\x2f\xb3\xf9\xf4\xdd\xf3\xd9\x9b\x43\xfe\xf3\xc5\xde\xe7\x8c\xca\xd6\x4b\xed\xbd\xd0\xaa\x93\x95\xd2\x97\xeb\x40\x6e\x11\x38\x16\x63\x78\xff\xf2\x5d\x6f\xef\xf7\xde\xf3\x71\x72\x42\xa5\x9f\x9a\x93\x94\xe4\x6d\xf0\x95\xe8\x15\x4e\xec\xae\x9c\x0d\x3f\xf4\xfc\xe0\xdc\x39\x9f\xba\xdb\x9c\x08\xb4\xc9\xfd\xaf\x17\x3b\xe6\x5e\x56\x3a\xbd\xa9\x42\x49\xb2\x07\xb3\x4d\x6f\x67\xe7\xdc\xf1\x99\xeb\x5d\x8c\x66\xdb\xc8\x3f\xdd\xe6\xfe\x74\x16\x7e\xdd\xf0\xe6\xa7\xfc\xeb\x7f\xfd\x9f\xff\xde\xfb\xfd\xf8\x70\x17\x7e\xd4\x34\xf6\x15\x53\x7e\xca\x0b\xbf\x19\xb0\x09\xd7\x6f\xf3\xae\x29\xea\xd5\xc7\x97\x6f\x3f\x1d\x1d\xef\x1d\xa6\x0b\x88\x33\xea\xaa\xec\xa7\x4c\x8e\x66\x05\x39\xd9\x7e\x30\xdb\xa4\x6c\xd3\xb9\x20\xb1\xb3\x4d\xb1\x94\xd2\x9c\x9d\xb9\xc3\x2d\x6f\x36\x15\x5f\x07\xc8\x2d\xbc\x6a\x9b\xd6\xb9\xea\x2e\x23\xc2\x70\x4f\xfe\xa7\x69\x15\x3e\xe6\x9f\xd9\x62\x2b\xe4\xe7\xa7\x43\x7e\x10\xbc\xfe\xba\x79\xfa\x7b\xf4\x6a\xfb\x25\xea\x3c\xfb\xdf\x00\x00\x00\xff\xff\xc7\x41\x11\x8d\xc3\xe3\x00\x00")

func kasFleetManagerYamlBytes() ([]byte, error) {
	return bindataRead(
		_kasFleetManagerYaml,
		"kas-fleet-manager.yaml",
	)
}

func kasFleetManagerYaml() (*asset, error) {
	bytes, err := kasFleetManagerYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "kas-fleet-manager.yaml", size: 58307, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
	"kas-fleet-manager.yaml": kasFleetManagerYaml,
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
	"kas-fleet-manager.yaml": &bintree{kasFleetManagerYaml, map[string]*bintree{}},
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
