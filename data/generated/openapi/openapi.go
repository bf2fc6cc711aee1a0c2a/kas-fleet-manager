// Code generated by go-bindata. (@generated) DO NOT EDIT.

 //Package openapi generated by go-bindata.// sources:
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

var _managedServicesApiYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x3d\x6b\x53\xdc\xb8\x96\xdf\xf3\x2b\xce\xf6\xdd\x5b\x9e\xd9\xa2\x9b\x7e\x01\x99\xae\xcd\x56\x91\x40\x32\xcc\x24\x24\xa1\x61\x98\x64\x6b\x0b\xd4\xb6\xba\x2d\xb0\x65\x23\xc9\x0d\x9d\xbb\xf7\xbf\xdf\x92\xe4\x87\x6c\xab\x5f\x40\x02\xdc\x21\xf3\x61\xb0\x2d\x1d\x1d\x1d\x9d\xb7\x8e\xd4\x51\x8c\x29\x8a\xc9\x00\x7a\xad\x76\xab\xfd\x82\xd0\x71\x34\x78\x01\x20\x88\x08\xf0\x00\x3e\x20\x8a\x26\xd8\x83\x21\x66\x53\xe2\x62\xd8\xfd\x74\xf0\x02\x60\x8a\x19\x27\x11\x1d\x40\xbb\xd5\x6e\x75\x5e\x00\x78\x98\xbb\x8c\xc4\x42\xbd\xb4\xf5\xe1\x98\xc9\x4e\x12\x72\x13\x12\x16\x0c\xc0\x17\x22\xe6\x83\xcd\x4d\x14\x93\x96\xc4\x81\xfb\x64\x2c\x5a\x6e\x14\xbe\x00\xa8\x01\x24\x14\x7e\x8a\x59\xe4\x25\xae\x7c\xf3\x33\x68\x70\x76\x60\x5c\xa0\x09\x5e\x06\x72\x28\xd0\x84\xd0\x89\x15\xd0\x60\x73\x33\x88\x5c\x14\xf8\x11\x17\x83\x97\xed\x76\xbb\xde\x3d\xff\x5e\xf4\xdc\xac\xb7\x72\x13\xc6\x30\x15\xe0\x45\x21\x22\xf4\x45\x8c\x84\xaf\x28\x20\xd1\xdc\x0c\x35\x95\x9a\x5c\x53\x89\x37\xe5\xcb\x69\x67\xf3\x12\x8d\x2f\x11\xdf\xfc\x07\xf1\xfe\x39\x50\x20\x27\x58\xe8\x3f\x00\xa2\x18\x33\x24\x61\x1f\x78\x03\xf9\xfe\x77\xd9\xf6\xf5\xec\xc0\x4b\xbf\x33\xcc\xe3\x88\x72\xcc\xb3\x0e\x00\x8d\x6e\xbb\xdd\x28\x1e\x01\xdc\x88\x0a\x4c\x85\xf9\x0a\x00\xc5\x71\x40\x5c\x05\x7a\xf3\x82\x47\xb4\xfc\x15\x80\xbb\x3e\x0e\x51\xf5\x2d\xc0\x7f\x32\x3c\x1e\x80\xf3\xb7\x4d\x37\x0a\xe3\x88\x62\x2a\xf8\xa6\x6e\xcb\x37\x15\x72\x47\xf8\x2a\xc1\x5c\x38\x95\x9e\xf8\x06\x85\x71\x60\xe2\x99\xfd\x33\x7b\xbd\xc3\xe2\x28\x9d\xd1\xbe\xee\x50\x6f\x6f\xc7\x21\x83\x5f\x42\x22\x85\x51\xc5\x65\xee\x98\xa7\x44\xf8\x6f\x11\x09\xb0\xf7\x86\x61\x45\x9b\xa1\x40\x22\xe1\xf7\x81\xcb\x02\xb8\x26\x7e\x25\x76\x52\xfd\x81\x69\x00\x30\x8e\x12\xea\xc1\x68\x06\xc4\x2b\x16\xbb\xdf\xee\x3c\xcc\x62\xef\x33\x16\xb1\xd5\x57\xb9\xdf\xee\xdc\x96\x8a\x45\xd7\xb9\x84\xda\x4d\x84\x0f\x22\xba\xc4\x14\x08\x07\x42\xa7\x28\x28\x13\xa9\xf7\x44\x88\xd4\xbb\x3d\x91\x7a\xcb\x88\x74\xc2\x31\x03\x1a\x09\x40\x89\xf0\x23\x46\xbe\x61\x0f\x44\x04\xc8\x75\x31\xe7\x20\x7c\x0c\xa9\x66\x6a\x99\x94\xeb\x3f\x11\xca\xf5\x6f\x4f\xb9\xfe\x32\xca\x1d\x46\x70\x59\x12\xc5\x6b\x22\x7c\xe0\x31\x76\xc9\x98\x60\x0f\x88\x07\xf8\x86\x70\xc1\x0b\xc2\x6d\x3d\x94\x12\x5e\x93\x70\x5b\xed\xf6\x6d\x09\x57\x74\x9d\xcf\x72\x14\xdf\xc4\xd8\x15\xd8\x03\x2c\xf1\x82\xc8\x55\x26\x32\x93\x4d\x8e\xdd\x84\x11\x31\x2b\xc6\x6e\xc2\x6b\x8c\x18\x66\x03\xf8\xdf\xff\xcb\x1a\x25\x61\x88\xd8\x6c\x00\xef\xb0\x00\x54\x59\x89\x42\x1d\x7a\x38\xc0\x02\x5b\x0d\xa7\xfe\xb4\x9a\xed\x7c\x84\xfc\x5e\x85\xf4\xb7\xc3\x08\xde\x68\xc4\xe6\x11\x7e\x4f\xcd\xf8\xd9\x4e\x3c\xdb\x89\x67\x3b\x51\xa6\x5c\x5f\x8b\xc6\x1d\xac\x45\x09\xc0\x5f\xd4\x66\xdc\x8d\x88\x55\x00\xb7\xb7\x1f\x99\x69\xd0\xe0\x16\x58\x87\x95\x6c\x4d\x8c\x18\x0a\xb1\x48\xc3\x56\xdd\x44\x4f\xa4\x51\x9a\x48\xd1\x6e\x93\x78\x8d\xd5\xe2\x3b\x0d\x31\x96\x11\xa6\xcd\x44\xb9\x32\x30\xd0\x26\x2a\xfd\x5c\x47\x46\xa2\x43\xe8\x00\xae\x12\xcc\x66\x06\xc9\x28\x0a\xf1\x00\x10\x9f\x51\x77\x1e\x21\x3f\x61\x36\x8e\x58\xa8\xa4\x17\xa9\x90\x1a\x08\x05\x44\x75\x2f\x9f\x45\x34\x4a\x38\x84\x88\x52\x15\x1b\x2f\x62\x26\x31\x8b\xf1\x00\x46\x51\x14\x60\x44\x8d\x2f\x92\xe6\x84\x61\x6f\x00\x82\x25\xf8\x45\xf1\x12\x73\xf1\x3a\xf2\x0c\xba\x5b\x62\x1c\x0f\x09\x94\x7f\xb7\xf0\xfd\x62\xae\xb7\xf3\xfc\xaa\xa1\xea\x27\x34\x0b\x22\xe4\x95\xf9\x7f\x1e\xf7\x9f\x0c\x8f\xf0\x84\xd4\xc5\x6e\x09\xc7\x67\xdd\xe6\x44\xa4\xfb\x27\xb7\x82\x9a\x75\xab\x41\x9d\xb3\x16\x36\x7f\xa7\xfb\x54\x73\x05\x9f\x22\xfe\xbd\x93\x05\x65\xdf\xc1\x75\x71\x5c\xf1\xa9\x9e\x86\xbe\xee\xb7\xdb\x59\xde\xe1\xf6\x66\xaf\x0a\x62\x2e\x9d\xfe\x90\x3e\x95\x6a\xa9\x75\x36\xaf\x2a\xed\x67\x6f\xf4\xd9\x1b\x5d\xe8\x8d\xe6\x60\x3f\xa0\x9b\xdd\x20\x88\xae\xb1\x77\x40\xb9\x40\xd4\xc5\x47\x18\xb9\x3e\xf6\xee\x30\xde\x32\x98\x8b\xdd\xe2\x71\xc4\x46\xc4\xf3\x30\x05\x4c\x84\x8f\x19\x8c\xb0\x8b\x12\x8e\x95\x61\x4d\x64\x0b\xc2\x57\xf2\x9d\x21\x2a\xf7\x0d\xd1\x0d\x09\x93\x10\x68\x12\x8e\x30\x83\x68\x0c\x24\x45\x4f\x76\x43\x02\x5c\x44\x61\x84\x53\x3f\x41\xa5\x01\x85\x4f\xb8\x1e\xd3\x47\x1c\x46\x18\x53\x60\x7a\x2a\x4f\xd3\x31\xff\x8e\x09\x9c\x63\x1f\x67\xae\x08\xf6\xa4\x21\x8c\x12\xe6\x62\xf0\x22\xcc\xa9\x23\xb4\x23\xfe\x24\xfd\xf0\xef\x98\xbb\xd9\xa5\x90\xcc\x73\xbf\x35\x1b\x12\x3a\x51\xbc\xab\x5d\xb8\xd4\x8c\xae\xee\x6d\x1b\xee\xbb\xb2\x2d\xd2\x7d\xa7\xf8\x3a\x75\xe1\x4d\x70\xc6\x86\x48\xde\xe5\x08\x8b\x84\x51\x0e\x08\x02\xc2\x85\x14\x98\x52\xb6\x9c\xdb\x9c\x6c\xd9\x52\xb5\xe2\xeb\x60\xb9\xca\x1e\x4b\x99\x72\x8b\x51\xfa\xa1\x2c\x65\x7a\x38\xef\x89\xe9\x6b\xd5\xdc\x97\xd2\x1c\x5e\x23\x2f\xc3\xfb\x09\x48\xc2\x81\xb6\x9d\x9f\x65\x60\x74\x07\x17\xc7\x02\xc6\x99\xef\xb4\xac\x61\xcb\x1f\x2f\xe5\xee\xd9\xc3\xf9\xeb\x3a\x2d\x77\x4e\xa1\x55\xad\xce\x5a\x89\x90\xc7\x4c\xea\x7b\x35\x53\xf6\xac\x88\x0d\x88\x91\xa4\x89\xd1\xc4\x58\xab\xa5\xcd\x39\xf9\xb6\x4e\xf3\x88\x79\x98\xbd\x9e\xad\x33\x00\x46\xcc\xf5\x9d\x65\x89\x23\x37\x88\x12\xef\x2c\x66\xd1\x94\x78\xf9\x84\xe7\xd8\x42\x46\xf0\x14\x6b\xd6\xca\x8c\x0f\x4f\xe2\x38\x62\x92\x65\x14\x20\xc8\x01\xb5\xe6\xd9\xc6\x37\xb2\xdd\xa7\xac\xd9\x1d\x6d\xa4\xd3\x6d\xb7\x9d\xb9\x0c\x9d\xa1\xec\x2d\x47\xf7\x21\x58\xbc\x44\x89\xb2\xdd\x74\xfa\xed\xce\xfc\x79\x3d\xdb\x01\x4d\xa4\xad\x45\x8b\xff\xac\xcd\x1e\x40\x9b\xad\xa3\x6a\x54\x31\xd2\x26\x53\xc9\xc5\x3b\xe8\x9d\x14\x80\x7c\xa9\x6c\xde\x1c\x09\x5f\x4d\x1f\xe9\x54\xe7\xa3\xd1\x4a\xd9\xe4\x1e\x4c\x3b\x69\x7a\x3c\xeb\xa6\x67\xdd\x94\xff\xfb\x61\xba\x69\xc9\x76\x58\xb9\xf1\x0f\x55\x64\xe9\x33\x72\xdd\x28\xa1\xa2\xae\xbb\x56\x51\x09\x3f\x6c\x79\xd3\x7a\xd9\x5d\x8d\x6c\x59\x94\x2b\x6c\xf9\x3e\xd3\x47\x69\xfa\x30\x9b\xe0\x7c\xd1\x7f\xac\x4c\xfa\x90\x39\x75\xa7\xdf\xee\x3d\x11\x22\x3d\x8a\xf0\x74\xbe\xce\x7c\xac\x84\x7b\x02\xf5\x6c\x35\x37\xa7\xac\x06\x78\xd5\xcb\x52\xa2\x6f\x95\x7b\xb3\x9a\x60\xf9\x4e\x7b\x05\x44\x2d\xbd\xf7\x03\xb6\xdd\xcb\x33\xb5\x6e\xff\xce\x5b\x5a\xbe\x22\xdf\xe4\xcb\x69\x1d\xeb\xd6\x3b\xe5\x8f\xc5\x48\xac\x2e\x09\x65\x4b\xb8\xb6\x34\x94\x87\x5d\x26\x18\x55\xde\x4a\xb7\xa9\x9e\xad\xd3\xb3\x75\x5a\xdb\x3a\x05\x76\x57\xe7\xd9\x16\x7d\x6f\x5b\xa4\x85\xb6\x2c\xf8\x55\x63\x94\xef\xd4\xd5\x57\xc7\x59\xc7\x31\x57\x19\x86\x74\x15\xeb\x55\x7c\xb6\xa5\xb7\xd4\x37\x97\x36\xd4\x33\xd5\xa3\xea\x2c\xe5\x87\x09\x99\x4a\x39\xb3\x55\x87\xdf\x31\xb6\xb1\xdb\x87\xfe\x23\x61\xca\x12\xa1\xbc\x4a\x21\xf8\xb3\x22\x56\x64\x41\xff\x2e\x8a\xf8\x0e\x44\x7a\xec\x61\x02\xfc\xe3\x9f\xff\xb6\xaa\x56\x8b\xe5\x62\x55\x9b\xea\xb8\xfb\x50\xb4\x9b\x0c\x73\x2c\x9a\x2e\xc3\x1e\xa6\x82\xa0\x80\xa7\xab\x64\x46\x10\xdf\x45\x29\x3e\x35\xa7\x19\x35\x15\xa9\xbe\xb3\xbb\xac\xc6\x00\x63\x39\x6e\xcd\x48\x0a\x52\x79\xf0\x37\x0c\x7b\xb5\x20\xb2\x36\x22\x8c\x23\x66\xb3\x9f\x2f\x8a\xd9\x49\x2c\x52\x82\x6a\x84\x3e\x8e\x2e\xb0\x2b\x8e\xf0\x18\x33\x4c\xdd\x5c\xd4\x74\x45\x78\xa4\x3e\x66\xbc\xc4\x24\x96\x82\x98\x34\x26\x9e\x49\x55\xdd\x89\x0b\x46\xe8\x24\x7f\x7d\x49\xe8\xf2\x46\xbe\x5c\x87\x45\x8d\x64\xc0\x6c\x86\xc5\x2a\xbe\x33\x28\x2a\x47\x31\x1e\x63\x34\xc1\xc6\x23\x27\xdf\xcc\x47\x11\x09\x14\x18\xcf\x44\xe0\x90\xaf\x37\xf1\x95\x66\x25\xb1\xa8\x37\x22\x54\xe0\x89\x51\x92\x2f\x91\x5b\xde\x4a\xe1\xbc\xb8\x99\x32\x34\x59\x13\x14\x04\x1f\xc7\xcb\x84\x3e\x13\xad\x0a\x13\x98\xf9\x5e\x0b\x3d\xe6\xd1\x04\x94\x36\xf0\x6a\xea\xda\x4a\x1b\x50\x0b\x89\x2c\xba\x61\x6e\xf3\x5c\x48\xce\xca\x6c\x67\xed\xa4\x88\x61\x72\xcd\x5a\x04\x91\x1d\xef\x40\x05\xc5\x50\x76\x14\x11\x63\x68\x56\xf9\x62\x6d\x0e\x0b\x11\x54\xd3\xd3\x18\x9a\xe5\x60\x0f\xbc\xfa\x5c\x1d\x81\x5f\x79\x41\xcb\x3b\x94\x2b\x77\x0b\x93\x40\x90\x33\xf4\xcd\xde\xa1\x7e\x8a\x05\xd2\x7d\xbd\xd5\x19\xed\x9a\xae\x81\x8e\x3a\xaa\xb3\x6a\xe3\x51\x14\x09\x2e\x18\x8a\x87\xea\xce\x8a\x5f\x0d\x5b\xbd\x9c\x5c\x3a\x01\x73\x86\x6a\x5d\xc6\x11\x0b\x91\x18\x80\x87\x04\x6e\x0a\x12\xe2\x55\x41\x26\xb1\x77\xdf\x20\xc7\xea\x52\x84\xb3\xb5\x64\xdb\x7a\x28\xd6\xc6\xb2\x8b\x4e\x79\xd4\xa5\xe1\x7b\x8b\xbf\x15\x6d\x65\x19\xa0\x51\xc5\xa3\x51\x6a\xa4\x2c\x03\x34\x3a\xe5\xb7\xca\x12\xd4\xde\x6a\xcd\x5f\x7b\x2d\x95\xc6\x2a\x79\xda\x55\x0f\xc6\x7c\x5f\x5d\x56\x21\x7f\xf1\x6f\xf1\x42\x98\x38\xd7\xd7\x37\x3d\xe5\x95\x81\x2d\xdf\x0d\xa3\x20\xe4\x1e\x51\x7e\x7a\x30\xf2\x66\xc0\x31\x15\x32\xea\x49\x8f\xf1\xc1\xa7\x8f\xc3\xe3\x05\xbe\x85\x14\xf0\xf5\xbc\x83\xf9\x8a\xad\x56\xc2\x5e\x2e\x41\x80\x6b\x1f\x33\x6c\x54\x60\xbb\x41\xc2\x85\x7c\x4f\x82\xc0\x3c\x2b\x40\xe8\x32\xe7\xc3\xa6\x25\xcb\x14\xc2\x42\x9f\x37\x10\x91\x4a\x94\xcb\xff\xbb\x11\x1d\x93\x49\x62\x45\x41\x44\x12\x01\x05\x76\xf7\x6b\x6d\xf4\xaa\xda\xad\x6a\xc5\xd2\xd0\x8e\x9c\xb9\x6c\x91\x95\x91\x94\x46\x6a\xc1\x81\x80\x30\xe1\x42\xa2\xc3\xd3\xcc\x61\x10\x5d\x63\xd6\x74\x11\xc7\x80\x82\xd8\x47\x34\x09\x31\x23\x2e\xb8\x3e\x62\xc8\x95\x71\x0b\x44\x0c\x1c\xa7\xe9\x38\x1b\xd2\x0e\xb1\x34\x6b\x84\xa8\x6e\x3f\xc2\xc2\x6c\xbd\x01\x88\x7a\x80\xa9\x57\x6e\x55\x83\xaa\xdb\xb9\x88\xaa\xa0\x79\x84\x21\x88\xe8\x44\x12\xc3\x47\x14\x7a\x5d\x63\xf8\x96\xb3\x6c\x45\xea\x56\xc8\x72\xa0\x41\x36\xb9\x27\x2e\xa8\x15\xbc\x3d\x94\x2e\xac\x21\xf2\xf0\xca\xb0\x84\xd2\x53\xd1\x86\x25\xa4\x1b\xc5\x1a\x17\x65\x43\x0f\xba\xc2\x05\x1a\x8f\x64\x7d\xe7\x9e\xcf\x7d\xbc\xab\xab\x51\x6e\xd4\xe5\xd7\x6a\xe4\x9c\x37\xe5\xa2\x3b\x67\x81\x4d\xaa\x46\xac\x65\x40\x07\xd4\x23\x2e\x12\x69\xd9\x9f\x9c\xb2\x56\xcd\x84\xa7\x8c\xd0\x82\xd3\x54\xf9\x38\x4e\x09\x31\xc7\x81\x80\xd0\xcb\xe5\xea\x8f\x2c\x18\xfe\x84\x92\xab\x04\x03\x51\x99\x8c\x31\xd1\xa7\xe5\x24\x26\xe9\xe0\x4b\x81\x7b\x84\xc7\x01\x9a\x9d\x2d\x36\x3b\x87\x86\xc9\xa9\x18\x5e\xe9\x28\xa4\x40\x20\x4e\x58\x1c\x71\xbc\x82\x4a\x5f\x3c\xdc\xaf\x49\x88\x28\x8c\x19\xc1\xd4\x0b\x66\x96\xd9\x95\x71\xd8\x50\x48\xa4\x2c\x0c\xe7\xe8\x9a\x9f\x2f\xc7\x00\x53\x34\x0a\xf0\x02\xd2\x9e\xfa\x58\x9d\x6a\xb4\xcc\x99\xf0\xac\xbb\x9e\x3e\x8e\x83\x68\x46\xe8\x44\x9a\xc3\x8f\xc3\xbd\xdc\x1e\xd7\x91\x28\x5b\x7b\x9b\xd3\x94\x02\xae\x2a\x29\x3b\x1b\xef\x15\x4f\x92\x34\x28\xb3\x83\xea\x6f\xf7\xe1\x78\x5c\xe3\xec\x38\x4f\x8e\xb9\x53\xfa\xd9\x98\xba\xc2\x65\x87\x2d\xf8\x83\xb0\x09\xa1\x04\xdd\x37\xb7\xa5\x48\xdc\x17\x97\xe9\xc1\xc6\x28\x09\xc4\x00\xc6\x28\xe0\x78\x45\xf6\x2b\xa7\x52\xed\x1c\x98\x5f\x73\x59\x2e\x76\x00\x42\xe1\xc3\xee\xb0\x39\x1c\x7e\xcc\x43\x09\xed\x92\xbd\x49\x5d\x32\xf9\x16\x25\xc2\x97\x6b\xab\x13\xde\xce\x6d\x6c\xf0\xfd\xe5\x60\xea\xb9\xb1\xf2\x4c\xf5\x65\x99\x30\xc1\x14\x33\x35\xc5\x24\x63\xcf\xbc\xa6\xbb\x9c\x3d\xae\x66\xda\xd7\x4a\x84\x94\xc7\x5e\x19\x94\xd9\xed\x7e\x20\xba\x01\xc1\x54\x1c\xec\xad\x91\x9c\x92\x1d\x86\xd8\x65\xf5\xad\x83\x7b\x4b\x63\x58\xb7\x18\x1a\x16\xae\xad\xe4\xf7\x2a\xcc\x6b\x8f\x75\x65\x3c\x37\x67\x6b\xdf\xb9\xd7\x70\x77\xbd\x58\x6f\x01\x7b\xd9\xb5\x9f\x9d\x17\xca\x83\xec\x9a\xcf\xf3\xf6\x41\x96\x0c\x55\x5b\xbe\x35\x96\xce\x96\x8a\xaa\x97\x02\x1f\x18\x6e\xac\x9d\x8f\xff\x72\x4a\x63\x9e\x5c\xde\x52\xcc\x1f\x4e\x07\xd5\x57\x7b\x6e\x30\x76\x8b\x00\xab\x0e\xbd\x1e\x20\xd5\x42\x91\x35\x76\x37\x33\xee\x5c\x23\x58\xaa\x3a\x5b\x0b\x89\xf9\xa0\x91\x95\x7d\xaa\x92\x80\xd5\xfd\xf1\x42\x0e\xf4\x05\x5c\x79\x01\x49\x2d\x53\xa3\x39\x9f\x61\x37\x62\x79\xc9\x40\x65\x53\xdb\x42\x0c\x42\x07\x10\x23\xe1\x57\xd5\x6f\x51\xa9\x6a\xee\x19\x6a\x1c\x8c\xbd\xcc\xea\x4d\x61\xe5\xfb\xc0\xd0\x04\x03\xa1\x1e\xbe\xa9\x41\x37\xbd\xa5\x15\xb0\xac\xef\xa5\x57\x77\x32\xa7\x28\x48\x8c\xe8\xdd\xdc\xc2\xd4\x48\x1b\x3b\xae\x0b\x91\x3e\x2c\xee\x46\x91\x4b\x2e\xfd\x2d\x8c\x5c\xdf\x9c\xf4\x3d\x4e\xa3\xba\xd5\x9a\x4f\xa3\xdd\xd6\x13\x49\xcf\xfd\x5a\x75\xf4\xff\x37\xf3\x9e\xc3\xf4\xca\x3f\x1d\x49\xa8\x4e\x30\x9a\x81\xcb\x88\xc0\x8c\xa0\x96\xe2\x10\x3e\xa3\x02\xdd\xe4\x21\x46\xce\x6a\x40\xb8\x81\x50\x48\x02\xa4\x72\xab\xa2\xd2\x05\xc3\x59\x06\xf8\x0c\xdc\x40\xdd\x2a\x23\xe3\x21\x0a\xc3\xcf\xef\xd5\x4e\x1b\x0e\x31\x15\xc5\x69\xf3\x7d\x49\x37\x45\xe8\xec\x62\x19\xd5\x5f\x5f\x2c\x83\xe8\x2c\x03\x7b\x69\x64\xd0\x39\x8c\x09\x0e\x3c\x5e\x40\x79\x5b\x04\x07\x1b\x72\x35\x34\x0e\x22\x02\x96\x9e\xa3\x93\x42\x08\x69\xe2\xdc\x18\x40\xf8\x98\xb0\x54\xfd\xe6\xc0\xce\xcf\xcf\xf9\x55\x50\xf2\x13\x00\x71\xd7\xfc\x5e\x34\xfe\xc8\x6e\x37\x1e\x9c\x21\xea\x9d\xe5\x1e\xbb\x87\xc4\x0a\x28\x6c\x18\xbb\x69\xf3\x51\x3a\xd0\x14\x33\x97\x8e\x3a\x22\x8b\x45\xbd\x0d\x88\x18\x10\xdd\x46\xb1\x92\x8a\x75\xc2\x58\xcc\x36\xe4\xbb\x22\x70\xd1\x5e\x19\x4f\x02\xc1\xf3\x1c\xae\x31\x17\x89\x54\x2b\x67\xda\x38\x88\x3c\x5c\xaa\x5d\xaf\x33\x72\x85\x4f\x4d\x5e\xce\x66\xd8\x98\x23\x7e\x5a\x3e\x53\x00\x77\x15\x31\x2e\x66\x01\x1e\xa8\xcd\x42\xad\x08\xd4\x29\x78\xbb\xf8\x14\xd2\xa3\x1a\x15\xd2\x52\xd0\x7b\x89\xd8\x2c\x11\x17\x95\x35\x2f\xcb\x4a\x31\x66\x49\x66\x20\xbd\x2e\x2a\xe5\x7e\xc9\x77\xda\x87\x51\x88\x21\x86\x07\x95\x2d\x9c\x0d\x45\xb6\x0d\xbd\x33\xbc\x91\x45\xb6\x88\x7a\xe9\x96\x77\x01\x51\x9a\x1f\xc4\x90\x88\x18\x97\x80\xe0\xfc\xbf\xff\xe7\x7c\x03\xce\x5f\x9d\x4b\x5e\x39\x7f\x7f\xf0\xfb\xfe\x79\x21\x6c\x59\xa7\x8b\x88\xd0\xb4\xf9\xee\xe1\xde\xb9\x02\x7c\xfe\xf1\xe8\x7c\x03\xfc\xe8\x1a\x4f\x75\x3c\xcd\x14\x7b\x21\x08\x48\x48\xd4\x96\x48\x88\x6e\x60\x1c\x25\x2c\xed\x5e\x9e\x84\x5a\x72\x83\xb4\xfb\x35\x1e\x2a\xa8\x6e\x08\x9b\xe5\x7e\x53\xc5\x50\xf8\x2a\x41\x01\x9c\x87\xb3\xa6\x6a\xa1\x51\x4c\xc9\x90\x7e\x43\xd7\xfc\x7c\xc3\x90\x05\xee\x47\x49\xe0\xc1\xa8\x22\x8d\x65\x51\x7c\x05\x19\x48\xbd\xd1\x52\xa2\x3a\xbc\x02\x74\xcd\xed\xa2\xb9\x04\x69\x22\xb8\x1e\x40\x6d\x05\x11\x3a\xd1\xaf\xcf\xc3\xd9\xfa\x38\x06\xe4\x12\x43\x38\xfb\x7b\x77\xeb\xbb\x68\x09\xa5\xe0\x72\xc5\xcc\x0d\xa5\x81\x44\x71\x07\x99\x8f\x38\xc4\x98\x85\x84\x73\x49\x73\x11\x01\xc7\x38\x57\x26\x4c\xdd\x9c\x64\xde\x12\x96\xff\x71\x18\x09\xdc\xca\x70\xd4\xe6\xa1\x28\x8f\xdd\x90\xf6\x44\xd7\x3f\x56\x41\x2d\xd0\x47\xa9\x89\x57\x8c\x36\x47\xcb\xd8\x35\x8a\xc5\x22\x97\x14\x06\x54\xf5\xd8\x0a\xec\xd1\xb8\xad\xb6\xca\x0a\xf3\x54\xf4\x9c\xe1\x94\x56\xe6\x99\x30\xf1\x00\x46\xea\x6d\xfa\x52\x3f\xbc\x4d\xeb\x23\x7e\x3b\x3d\x2e\x05\xcb\xbe\x10\xf1\x8b\xea\x4c\x2b\x57\x85\x66\xe0\xf5\x54\x73\x8c\xd3\xfd\x41\x68\x24\xbc\x89\x11\x17\x4d\x63\x13\xa4\xb2\xa3\x0c\x0d\x63\xe6\xd9\x82\x34\xb2\x32\xd1\x98\x88\x7c\xbf\xbc\x72\x9f\xe8\xb2\xa1\x71\xd2\xbc\xc6\xf7\x34\xb4\xa5\xe0\x60\xce\xf0\x44\x86\x38\x1d\x32\xfc\xb2\x7d\xf4\xb9\xf7\xdb\xef\x07\x2f\x3f\xb7\x3f\x1e\x87\x17\x9f\xdf\x7a\xbd\xc8\x7d\x7b\x34\x29\x86\x4b\xc3\x21\xc5\x12\xc5\x5b\x5f\x3b\xff\xab\xfc\x24\xcb\x4a\xa3\xa4\x75\x4c\xd0\x60\x18\x79\xb3\x55\x49\x91\xef\xb8\x9b\xb2\xb2\x78\x59\x75\xa5\x11\x40\x03\xc5\xe4\x4c\x61\x78\x96\xe2\xbd\x80\xc2\xc5\x27\x5b\x35\x91\xd9\xb2\xd9\x21\x7c\xb6\xcd\xae\x7a\x17\x97\xe4\xe5\x55\x3b\x12\xe1\xc5\xd5\x58\xce\x76\xcc\x26\x2d\x14\xc7\xbc\x15\xf2\xe6\x48\x88\x49\xfb\x82\x76\x76\xda\x7e\xdc\xba\xd9\x4a\x5e\xb6\x78\xa7\xe5\xe1\xa9\xfe\xe1\x9d\x88\x19\x74\x31\x2a\x90\xd4\x3d\xb3\xed\x66\xa7\xdd\x6c\x6f\x1d\x77\xba\x83\xad\xce\xa0\xdb\x6f\xb5\xb7\x7a\x9d\x7e\xf7\x6b\xd1\xc3\x28\x30\xaa\xf5\xd8\x1e\xf4\xb6\x5b\xbd\xed\x6e\xb7\xfd\xf2\x6b\x9d\x63\x56\xf8\xd1\x96\xa7\xc2\x45\xba\x24\xea\x99\x8d\x7e\x14\x1b\x41\xad\x0c\x0d\x1a\x28\xad\x36\x2d\x2c\x6d\xee\x30\xb8\x29\x8f\x81\xb9\x4e\xb6\x02\x81\x39\x7c\x67\xab\x72\x68\x94\xb9\xd2\xa6\x32\x4b\xef\x4a\x5b\x3c\xd0\xd8\x0d\xd1\xb7\x88\xc2\x29\x1e\x65\x79\x25\xa3\x6d\xb6\x09\x53\xf0\x47\x7d\xb7\x7b\x05\x54\xcd\xad\xe6\x1c\x51\x0b\x73\x55\x50\x3b\x19\xc2\x3e\xe2\x62\x03\x8c\x9d\xa3\x45\xb8\x2d\xca\x93\xce\xc1\x32\x1d\x29\x9c\x35\x51\x1c\x37\xb9\x01\xbe\x14\x51\x34\xaa\xc7\xc4\xc6\x11\x83\x70\x06\x28\x8e\x6d\x09\xd8\x55\x94\x47\x4d\x45\x94\x41\xac\xa7\x2b\xaa\xa7\x36\x3a\x35\x16\xb8\xf3\x0c\xc1\xcc\xa0\x42\x63\xb8\xdb\xec\x74\xe5\x7f\xd5\xaf\xe9\x36\x86\x04\x28\xff\x58\x94\x9f\xbe\x07\x32\x99\x19\xb6\x27\x44\xae\xf9\x77\x6a\x5b\x89\xd0\xee\xd5\xc8\x60\x54\x65\xaf\x3a\x6b\x7d\x9f\xf6\x66\x09\x9a\x2a\xa2\x87\xc6\xc7\x37\x1f\x9a\xfb\x7f\x36\x4b\x9f\x72\x7d\x36\x34\xee\xf8\x4f\x6f\xff\xff\x29\xbf\xc1\xb5\xa4\xdf\xf8\xcf\xd2\xe1\x4f\x68\x7e\xed\xd1\x86\x8c\x28\xf4\xaf\x04\xbc\x52\x85\x7f\x79\xe4\x62\x58\x14\xb3\xd6\x5e\x5a\xd5\xd3\x03\x12\x5e\xbd\x73\xd9\x5e\xf2\x7e\xbb\x83\x4e\x6e\x0e\xbe\x5e\xbd\x3e\xbe\x3a\x3c\x42\x39\xed\x16\x5c\xd8\x69\xa3\x5f\xf7\x5e\xe9\xd7\xad\x93\xef\xc3\xbb\xbd\xe6\x70\xff\xe8\x8f\xe6\xee\xa7\x83\x66\xd7\x46\x43\xed\x62\xc8\x58\x2a\x46\x8c\x97\x43\xe7\x01\x9c\x28\x55\x26\xbf\xaa\xd3\xcb\x65\x9a\xea\x7d\xdf\xaa\xb5\x1d\x40\x65\xd0\x01\x2c\x1b\xc3\xb8\x8c\x2a\x0a\x92\x90\xea\x90\x53\x42\x4f\x43\x23\x70\x88\xe7\xb4\x60\x68\x6b\xc7\x75\xae\x42\xbb\x06\x59\x8e\xc2\x9e\xb9\xc8\xf2\x14\x9f\x75\x04\xa8\xd7\x6a\x00\xc4\x83\x57\xd0\x31\xa9\x53\x5d\xf9\xe0\x74\xef\x5d\x32\x1b\x1d\xb0\x7d\x7a\xc3\x76\x71\xb8\xd3\xed\x4f\xae\x2e\x2f\xc9\xde\x34\x5b\xf9\xea\x2d\xcf\xb6\xd5\xee\xb7\xfb\xeb\xac\xf6\x04\x09\x7c\x8d\x66\xc6\x02\x97\x00\xa4\x2b\xfc\x6e\xf7\x78\xff\x74\xf7\x4b\xb3\xf4\x2d\x5f\xdd\x55\x2e\x88\xce\xa7\x60\xfd\xf1\x13\xcb\x44\x9c\x1d\xa7\x32\x0d\x35\x8b\xdb\xb0\xec\x4e\x75\x3e\xa9\xbc\xef\xd4\xa6\x92\xba\xc5\xe9\x04\x74\x7e\xc3\x7b\xe5\x74\xc8\xef\x3d\x2f\xf9\xe3\xcb\xc1\x74\xba\xf5\x65\xfa\x3e\x98\x7d\xeb\x84\xef\x8e\x7a\xbf\xcd\xae\x0e\x1d\x55\xae\xaa\x7e\x5d\x70\xce\xc2\x76\xc8\x97\x8f\x3b\x93\xee\x64\xfb\xd7\x63\xef\xe4\xf7\x13\xd4\xbd\xe4\xbf\xbe\xec\x5e\x7e\xde\xeb\xcd\x52\x9a\x54\x8f\xa1\x5a\x95\x60\xdd\x14\xdc\x41\x07\x76\xe6\xaa\xc0\x8e\x65\x7d\x0b\xf9\x9c\x62\x46\xc6\x33\x19\x8a\xeb\xf3\xbf\x03\x75\xd3\x34\x61\xd8\xcb\x8f\xb9\x6a\x07\x4f\x9f\x0e\xce\x49\xb3\x40\xdd\x7d\xe9\x9d\xf8\xfb\xfe\x75\xf8\xe7\xeb\xf8\xf4\xd3\xf8\xa0\x1b\x1c\xe2\xcb\xd8\xeb\x7f\xdd\xcb\x38\xa6\x7a\x5b\x82\x8d\x57\xfa\xab\xf0\xca\x2a\x94\xe9\x57\xe8\x52\xd2\x32\xfd\x1a\x69\xd4\x21\x5f\x67\x1c\x45\xcd\x11\x62\xce\xaa\xd7\xd7\xb7\xe6\x71\x4a\xf0\xa5\x77\x42\xf6\xfd\x6f\xd4\x20\xc6\x45\xec\xf5\xbf\xbc\xc9\x68\xb1\xea\xa5\xfe\x36\x22\x6d\xdd\x17\x91\xb6\x16\x11\x69\x6b\x19\x91\x7c\xc4\xb3\xab\xf6\x01\x59\xae\xee\xdf\x06\x94\x26\x6b\xf3\x4b\xfc\x97\x10\xec\xf2\x46\x12\xec\x8f\x4f\xf8\xa0\x1b\x1d\xe2\x0b\xaf\xf7\xe7\x6b\xd5\xa1\x7e\xae\xd9\x46\x96\x5f\xee\x4d\xcf\xfc\x52\x95\xaa\x12\x61\x7e\xa9\x13\x86\x16\xbf\x68\x85\x4b\x83\x56\x35\x08\xde\x7f\x3f\x7d\xfb\xcb\xc5\x87\xcf\x5f\xb6\xbf\x4c\xfc\xf1\x87\x5f\x26\xef\x8e\xf8\xaf\xd3\xfd\xd3\x6c\x9e\x2b\x6b\xd5\xef\x38\xdb\x54\x85\xd4\xe7\xe9\xe8\xbc\xa7\x3a\xb6\xad\x7e\x45\x78\x46\x5d\x2e\x9d\xe4\xac\xc7\x20\x0d\x08\xa5\xb8\x78\xe9\xc1\xed\xac\x0d\xbe\x11\xcd\x52\x80\x7c\xd3\xee\x05\xd4\x0b\xc2\xab\xf6\xd5\xd8\xdd\xe1\x44\xa0\x2d\x1e\x5c\x4c\x5f\x9a\xe7\xa6\xa4\x55\xcf\x4a\xf9\x15\x05\x27\x5b\xde\xcb\x97\x57\xed\x80\xb9\xde\xb4\x3f\xd9\x41\xc1\x68\x87\x07\xe3\x09\xbd\xe8\x79\xfe\x88\x5f\xfc\xfd\x3f\x7e\xda\xff\xf3\xf8\x68\x17\xfe\x4b\x4f\xae\xa5\xa8\xf2\xaa\x28\x24\x34\x60\x13\x0e\x8e\xd3\x6f\xf7\x1d\x67\x43\x4d\x5c\xbf\x78\xf3\xfe\x64\x78\xbc\x7f\x34\x6c\x7e\x78\xf7\xe1\xb8\xa9\x3e\xab\x44\x6a\xbe\x92\x66\x55\xa2\xea\xd1\x99\x6c\x45\x6c\xab\x3d\x25\x49\x7b\x27\xc2\x72\xa9\x7c\x76\xe9\x76\xb7\xbd\xc9\x58\x5c\x74\x90\xeb\x94\x0e\x7f\x67\x55\x70\xaa\xdf\xc2\xb9\x38\x86\x49\xfa\xd9\x99\x6f\x94\x8e\xf9\x29\x9b\x6d\x53\x7e\x35\xea\xf2\xc3\xf0\xed\xc5\xd6\xe8\xcf\x78\x6f\xe7\x0d\x7a\xf1\xaf\x00\x00\x00\xff\xff\x6d\x87\xa7\xd3\xe3\x7a\x00\x00")

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

	info := bindataFileInfo{name: "managed-services-api.yaml", size: 31459, mode: os.FileMode(436), modTime: time.Unix(1610014073, 0)}
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
