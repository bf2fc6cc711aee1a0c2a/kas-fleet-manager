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

var _kasFleetManagerYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x7d\x7b\x73\xdb\xb6\xb2\xf8\xff\xfe\x14\xfb\x53\x7e\x67\x74\x4e\xaf\x25\x53\xb2\xec\x24\x9a\xdb\x3b\xe3\x38\x4e\xea\x36\x71\x12\x3f\x9a\xa6\x9d\x8e\x0c\x91\x90\x04\x9b\x04\x68\x02\xb2\xad\x9c\x7b\xbe\xfb\x1d\x3c\x48\x82\x4f\x51\xb6\x13\xdb\xad\x7d\xe6\x4c\x23\x12\x00\x17\xbb\x8b\xdd\xc5\x62\x77\xc1\x42\x4c\x51\x48\x86\xb0\xd9\x75\xba\x0e\x3c\x03\x8a\xb1\x07\x62\x46\x38\x20\x0e\x13\x12\x71\x01\x3e\xa1\x18\x04\x03\xe4\xfb\xec\x0a\x38\x0b\x30\xec\xbf\xde\xe3\xf2\xd1\x39\x65\x57\xba\xb5\xec\x40\xc1\x0c\x07\x1e\x73\xe7\x01\xa6\xa2\xbb\xf6\x0c\x76\x7c\x1f\x30\xf5\x42\x46\xa8\xe0\xe0\xe1\x09\xa1\xd8\x83\x19\x8e\x30\x5c\x11\xdf\x87\x31\x06\x8f\x70\x97\x5d\xe2\x08\x8d\x7d\x0c\xe3\x85\xfc\x12\xcc\x39\x8e\x78\x17\xf6\x27\x20\x54\x5b\xf9\x01\x03\x1d\x83\x73\x8c\x43\x0d\x49\x3a\x72\x2b\x8c\xc8\x25\x12\xb8\xb5\x0e\xc8\x93\x73\xc0\x81\x6c\x2a\x66\x18\x5a\x01\xa2\x68\x8a\xbd\x0e\xc7\xd1\x25\x71\x31\xef\xa0\x90\x74\x4c\xfb\xee\x02\x05\x7e\x0b\x26\xc4\xc7\x6b\x84\x4e\xd8\x70\x0d\x40\x10\xe1\xe3\x21\xfc\x82\x26\xe7\x08\x8e\x74\x27\x78\xe3\x63\x2c\xe0\xbd\x1a\x2a\x5a\x03\xb8\xc4\x11\x27\x8c\x0e\xa1\xd7\xed\x77\x9d\x35\x00\x0f\x73\x37\x22\xa1\x50\x0f\x6b\xfa\xea\xb9\x1c\x62\x2e\x60\xe7\xe3\xbe\x04\x52\xc3\x67\xfa\x10\xca\x05\xa2\x2e\xe6\xdd\x35\x09\x2f\x8e\xb8\x04\xa9\x03\xf3\xc8\x1f\xc2\x4c\x88\x90\x0f\x37\x36\x50\x48\xba\x12\xdb\x7c\x46\x26\xa2\xeb\xb2\x60\x0d\x20\x07\xc1\x7b\x44\x28\xfc\x33\x8c\x98\x37\x77\xe5\x93\x7f\x81\x1e\xae\x7c\x30\x2e\xd0\x14\x2f\x1b\xf2\x48\xa0\x29\xa1\xd3\xd2\x81\x86\x1b\x1b\x3e\x73\x91\x3f\x63\x5c\x0c\x5f\x38\x8e\x53\xec\x9e\xbc\x4f\x7b\x6e\x14\x5b\xb9\xf3\x28\xc2\x54\x80\xc7\x02\x44\xe8\x5a\x88\xc4\x4c\x61\x40\x82\xb9\x71\x2e\x51\xc4\x47\xc1\x34\x10\x1b\x97\xbd\xa1\xea\x3d\xc5\x42\xff\x03\x24\x03\x46\x48\x0e\xb3\xef\x0d\xe5\xf3\x5f\x35\x8d\xde\x63\x81\x3c\x24\x90\x69\x15\x61\x1e\x32\xca\x31\x8f\xbb\x01\xb4\xfa\x8e\xd3\x4a\x7f\x02\xb8\x8c\x0a\x4c\x85\xfd\x08\x00\x85\xa1\x4f\x5c\xf5\x81\x8d\x33\xce\x68\xf6\x2d\x00\x77\x67\x38\x40\xf9\xa7\x00\xff\x3f\xc2\x93\x21\xb4\x9f\x6d\xb8\x2c\x08\x19\xc5\x54\xf0\x0d\xdd\x96\x6f\xe4\x40\x6c\x5b\x9d\x33\x68\x31\xed\x20\xc8\xce\x85\xcf\x83\x00\x45\x8b\x21\x1c\x62\x31\x8f\x28\x57\x0c\x7f\x99\x6f\x5b\x8e\xbe\x0d\x1c\x45\x2c\xe2\x1b\xff\x26\xde\x7f\x96\xa2\x72\x4f\xb6\x7d\xb5\xd8\xf7\x1e\x22\x12\x15\x70\x95\xa8\x7b\x8b\x05\xa8\xa9\x4a\xe1\x92\x4c\xa0\x14\x73\x49\x33\x12\x37\x13\x68\x6a\x4d\xb1\xa3\x5b\x70\xf3\x20\x44\x11\x0a\xb0\x30\x6b\x34\x6e\xa2\x21\x6d\x65\x20\x4d\x5b\x6e\x10\xaf\x55\x4f\x90\x66\xb4\xe0\x0f\x96\x10\xef\x08\x17\x95\xc4\x90\x2f\x81\x4d\x20\x64\x9c\x13\x29\xf0\x33\x08\x2d\x25\x8a\x9f\xef\x22\xc5\x66\xa6\x5b\x25\x91\xca\x90\xac\x7f\x36\xe3\x7a\x25\x92\x1f\x2a\xd7\x2b\xe0\x0e\xf1\xc5\x1c\x67\xf1\x2d\xff\xf0\x35\x0a\x42\xdf\x86\x33\xfe\xb3\x7b\xbd\xc5\xe2\xd0\xcc\x68\x4f\x77\x28\xb6\x2f\x87\x21\x1e\x3f\x03\x84\x19\x23\x0f\x4b\xe5\x37\x3f\x13\x31\x7b\x83\x88\x8f\xbd\xdd\x08\x2b\xdc\x1c\x09\x24\xe6\xfc\x2e\x60\xa9\x19\xb7\x92\x37\xb5\x02\x8e\xf4\x00\x30\x61\x73\xea\x29\x91\xf1\x3a\x25\xf6\xc0\xe9\x3d\x10\x11\x57\x4f\xe5\x81\xd3\xbb\x29\x16\xd3\xae\x95\x88\xda\x99\x8b\x19\x08\x76\x8e\xa9\x34\x66\x08\xbd\x44\x7e\x22\x30\x15\x92\x36\x1f\x09\x92\x36\x6f\x8e\xa4\xcd\x65\x48\x3a\xe1\x38\x02\xca\x04\xa0\xb9\x98\xb1\x88\x7c\xd5\xc6\x2b\x72\x5d\xcc\xb5\x60\x33\xf6\xa8\x8d\xb8\xc1\x23\x41\xdc\xe0\xe6\x88\x1b\x2c\x43\xdc\x01\xcb\xad\xc4\x2b\x22\x66\xc0\x43\xec\x92\x09\xc1\x1e\xec\xbf\x06\x7c\x4d\xb8\xe0\x29\xe2\xb6\x1e\x8c\xe5\x51\x8f\xb8\x2d\xc7\xb9\x29\xe2\xd2\xae\xd5\x1c\x47\xf1\x75\x88\x5d\x81\x3d\x63\xc8\x30\x57\x59\xd3\x89\xc9\x83\xdd\x79\x44\xc4\xc2\x56\x95\xaf\x30\x8a\x70\x34\x84\x3f\xe0\xcf\x2a\x1d\x8c\x72\xe4\x48\x45\xa2\x87\x7d\x2c\x70\xa9\xf2\xd4\xaf\xf2\xfa\xb3\xdc\x60\x22\x74\x08\x17\x73\x1c\x2d\xac\x89\x51\x14\xe0\x21\x20\xbe\xa0\x6e\xd5\x74\x3f\xe2\x68\xc2\xa2\x40\x2d\x25\xa4\xf6\x38\x40\xa8\xdc\x87\xaa\x5e\xb3\x88\x51\x36\xe7\x72\x73\x45\xd5\x66\xa5\x8e\xcc\x62\x11\xe2\x21\x8c\x19\xf3\x31\xa2\xd6\x1b\x39\x65\x12\x61\x6f\x08\x22\x9a\xe3\x5a\x23\xa0\xff\xf0\x18\x30\x3f\xd2\xb3\x03\x06\xbb\x1a\xb0\x2a\x9c\xbe\x56\x64\xcb\xc8\xf2\xc7\xb1\xb2\x06\x8e\xa3\x60\x27\x8c\xde\x5c\x34\xe5\x87\xa8\xde\x8d\x49\x85\xa7\xe6\x6b\x6c\xcd\xfc\x52\x7b\x32\x15\x9e\x4c\x85\x27\x53\x41\x9b\x0a\x5a\xa6\xdc\xc2\x60\xc8\x0c\xf0\x37\x35\x1b\x6e\x87\xc4\xfc\x00\x37\x37\x21\x62\xe3\x40\x0f\x57\x67\x1c\x34\xb3\x37\x42\x24\xdc\xd9\x30\x3f\xfa\x49\xe8\x21\x81\x93\xc1\x63\x9f\x68\xc6\x33\xd3\xcc\x9a\xc9\x18\x25\x73\x35\x6c\x71\x53\xaf\x40\x7f\xc5\x3c\x6b\xac\x2c\x56\x34\x38\xec\x8a\xe2\x08\xd8\x04\x94\x0b\x61\xad\x86\x6b\xea\x79\xa6\x9c\x63\x96\x6e\xf5\x35\x14\x85\x0d\xff\x0a\x36\x4a\x96\xdb\x4b\xf6\xbe\x1a\x41\xf9\x5d\xef\xa3\xf2\x69\x7c\x64\xfc\xdb\x3a\x35\x0a\x26\x51\x06\x8f\xaf\x90\x17\x33\xd4\x03\x10\x2c\x05\x23\x64\x05\xdd\x7c\xaf\x50\x6f\x56\x43\xad\x94\x25\xe1\xb6\xbe\xe4\x4b\xf5\xe5\xbd\x4e\x66\x50\x3d\x99\x44\x69\x69\x6f\x93\x52\x59\x0a\x7e\x4b\x6d\x3d\x84\x49\xe4\x35\xe5\x4a\x4a\xe3\xbe\x20\x2f\xee\x36\x1b\x39\xe7\xeb\xdc\xc6\x7a\xa0\x90\xf1\x72\x97\xb1\x1b\xe1\x58\xc1\xfc\xb5\x76\xbc\xcb\x34\xa4\x66\x62\xeb\x60\xea\xfb\xa9\xc5\x58\xee\xa3\x85\xcf\x90\x97\x55\x1a\x55\x2a\xe3\xe4\xe8\x10\x4f\x49\x91\xe1\x96\xe8\x85\xb8\x5b\x85\xa3\x7b\xef\xe4\x46\xa3\xc6\xdd\x0a\xa3\x3e\x7c\xef\xc3\x23\x50\xd7\x79\x9d\xe7\xba\x38\x7c\xac\x1e\x8e\xf8\x38\xe3\x16\x1e\x8e\xdc\x10\x4f\x1e\x8e\x27\x0f\x47\x8c\xa4\x3b\xf6\x70\x24\xc3\xbe\x47\xd7\x3b\xbe\xcf\xae\xb0\xb7\x6f\xf6\x71\x87\x18\xb9\x33\xec\xdd\xe2\x7b\xcb\xc6\x2c\x05\xe4\x18\x47\x01\x3f\x60\x22\x96\x01\xb7\xf8\x7e\xc5\x50\xf5\x1e\x9e\x09\x8b\xc6\xc4\xf3\x30\x05\x4c\xc4\x0c\x47\x30\xc6\x2e\x9a\x73\xac\xf4\xf9\xbc\x68\xd6\x56\xba\x81\x80\x65\xfb\x06\xe8\x9a\x04\xf3\x00\xe8\x3c\x18\xeb\x1d\x6a\x12\x45\x04\x62\x86\x04\xb8\x88\xc2\x18\x1b\xf3\x44\x6d\xef\x54\xd8\x96\xfa\xe6\x0c\x71\x18\x63\x4c\x21\xd2\x18\xec\x56\x9b\xae\x0f\x97\x77\xbf\xe5\x79\xd4\xf1\x0c\xc7\x16\x10\x96\x9b\x3b\xce\xe6\x91\x8b\xc1\x63\x98\xd3\xb6\xd0\x4e\x25\x1b\x67\x2f\x1f\x09\xce\x5e\x1e\xa0\x00\xef\x32\x3a\xf1\x89\x2b\x6e\x8e\xbf\xb2\x61\xaa\x85\xa5\xc4\x87\x6a\x99\xf2\x9d\x87\x85\xde\x3c\x10\xaa\xb8\xd9\x35\x2a\x4a\xf2\xb1\x62\xd3\x18\xe5\xd5\xdb\x91\x87\x8a\xe4\x6f\x7b\xde\xb7\x43\x61\x5e\xb5\xf5\x82\xab\x19\xf1\x63\x5c\xd2\xa9\x42\x6c\xc6\x53\x67\x06\x5d\xf1\x4c\x50\x99\x0f\x45\xb7\x9f\x6a\x66\xc5\xd1\x94\x9c\x21\xc6\x51\x3c\x99\x7e\xbc\x6c\x13\x15\xc7\xdd\xf0\x95\x40\x5c\xd9\xe3\xb5\x53\x0f\xd2\x77\x65\x2b\xdb\x82\xcd\x86\x4f\x3d\x26\x6f\x93\xfe\xab\x5e\x0d\xfb\xda\x36\xfa\x24\x37\xbe\xb7\x30\x61\x4b\x86\x79\xec\x1e\xaf\x65\x98\xbb\x63\x0b\xf6\xef\x6b\x94\xde\xf6\xd8\xed\x51\xfa\xc1\x9a\x60\xfa\x4e\x35\x55\x5d\x5c\x6c\xbb\xca\xf5\x16\xa2\xa9\x45\xaa\xa5\xcd\x39\xf9\xba\x4a\x73\x16\x79\x38\x7a\xb5\x58\xe5\x03\x18\x45\xee\xac\x5d\xe1\x0e\x74\x7d\x36\xf7\x46\x61\xc4\x2e\x89\x87\x4b\x62\x76\x6b\x23\x59\xf9\x3c\x0c\x59\x24\xf9\x44\x0d\x03\xc9\x30\x15\xea\x70\x57\xb6\xfa\x98\x6b\x74\x63\xb5\xd8\xee\x3b\x4e\xbb\x92\x89\x35\xbc\xd8\x6b\x0c\x2c\x7c\x4f\xae\xce\x60\x22\xab\x29\xdb\x03\xa7\x57\x3d\xad\x27\xc9\xaf\x91\xb4\x55\x47\xfb\x27\x01\x76\x0f\x02\xac\x81\x74\x51\xc1\xea\x1b\x91\xf2\x12\xdf\x58\xd4\x98\xee\x7a\x57\x85\x2b\x97\x75\x13\x11\xa4\xfd\xd5\x0f\x45\x10\xc5\x33\xbb\x37\x79\xa4\xd1\xf1\x24\x8d\x9e\xa4\x51\xf2\xf7\xdd\xa4\xd1\x92\x93\xcc\x6c\xe3\x7b\xb2\xbd\x62\x67\xe4\x48\x2c\xc2\x4a\x91\x67\x4c\xed\x11\x72\x5d\x36\xa7\xa2\x28\xe6\x9a\x08\x90\xef\xc6\x0e\x26\xb3\x73\x47\x03\x5b\x9b\xf1\x54\x94\x63\xc6\x89\x1b\xcf\xb4\x5a\x66\x3c\x54\xee\xbe\xcf\x03\x95\xf6\xc0\xd9\x7c\x24\x48\x7a\x58\x7b\xd7\x82\xb0\x7d\xa8\x88\x7b\x0c\x49\x12\xf9\x8c\xc3\xb8\x57\x85\xf5\x94\x15\x17\x95\xd9\x8e\xa8\x5e\x46\xd8\x61\x27\xcb\x43\x32\x8e\xb2\x43\x14\xfc\x84\xdf\x21\x3e\x23\x3b\xed\xd2\x38\x81\x2a\x36\xe0\x0d\x79\x2c\x21\x7d\xe9\xb7\x6e\x1c\x52\xf1\x50\x34\x4b\xf3\x55\x63\x38\xc6\x50\x7b\xe5\x95\x93\xfd\xec\xb2\x45\x94\xe7\x2d\x73\xb0\xf8\xa4\xc9\x9e\x34\xd9\xca\x9a\xec\xdd\x52\xb3\xe8\x49\x71\xdd\x9d\xe2\x2a\x09\x57\xcc\x2e\xfd\x66\x0a\xae\xe4\x40\x30\x47\xbf\x86\x66\x7e\x79\x1e\xfe\x2d\x77\x3c\x7f\x0d\x81\x5e\xf2\x9d\x95\x84\xf8\xab\xc5\xfe\xd2\xb8\x94\xd4\xf2\xc8\x91\x6f\xe5\x44\x8e\x65\x46\x8f\x95\x70\xd1\x94\xb7\x92\x9d\x53\x35\x6c\x49\xdb\xb7\x58\x94\x35\x33\xe2\xb6\x50\x0f\xc4\x0e\xa6\x89\x9b\x27\xc1\xd7\x53\x72\x29\x25\x76\xdc\xd5\xce\x71\xfd\x26\x8c\x39\x78\x20\xd2\xad\x36\x13\xf4\x49\xa5\xff\xb5\x54\xfa\x2d\x90\xf4\xd0\x37\xa7\xf0\x6f\xf8\xcf\x5f\x57\x69\x6b\x81\x74\x6b\xe1\x9a\x26\xf0\x55\x49\xd7\xc6\xea\x7b\x23\xc2\x1c\x8b\x91\x1b\x61\x0f\x53\x41\x90\x5f\x92\x26\xf1\xa4\xd1\xa5\x46\xef\x28\x4c\x7d\xe3\xcd\xd9\xa1\xfc\x06\x58\xd4\x78\x92\xe1\x4f\x32\xfc\x49\x86\x3f\x24\x19\xae\xc4\x40\x76\x55\xef\x46\xd8\xab\xaa\x67\x56\x6d\x20\x73\x2c\x78\x1c\x34\x1b\x2f\x77\x98\xb0\xa8\x46\xac\x3f\x93\xff\x87\xe3\x19\xe6\x18\x50\x94\x06\x9f\x77\x26\xc8\x25\x74\x0a\x11\xf6\x55\x90\x78\x52\x5b\xd3\xf4\x59\x52\x4b\x6d\x23\xc0\x22\x22\x2e\xdf\x50\x79\x6d\xa3\x08\xd1\x29\x2e\xec\xeb\x0a\x1e\x4f\xd3\xc9\xd8\xde\x24\xc0\x1c\x47\x04\x73\x50\xdd\x75\x8a\x9c\x04\x5c\x47\x68\x26\xfb\x91\xfc\x4e\xe3\xbd\x1e\xe5\xd5\xe2\x50\x76\xfb\x64\x25\xd6\x7d\xeb\xb3\xe9\x9f\x8f\x3e\x1c\x00\x8a\x22\xb4\x00\x36\x81\x8f\x11\x0b\xb0\x98\xe1\x79\x3a\x31\x36\x3e\xc3\xae\xe0\x30\x89\x58\x00\x6c\x2c\x89\x82\x04\x8b\xc8\x3c\xb8\x0f\x09\x63\x10\x95\xa2\xe9\xe9\xd0\xfa\xe9\xd0\x3a\xf9\x7b\x98\x87\xd6\x95\x8d\xbd\xb9\x16\x02\x2b\x74\x21\x54\xc8\x05\xe8\xaf\xd0\x65\x42\x7c\xf9\xdf\xfa\xb4\xe0\x12\x09\xb8\xa2\xec\xd3\x67\xe4\x62\x75\x91\xa7\xf3\x9f\xc4\x93\xd0\x5b\x26\xf4\x6c\x44\x3d\x89\xbd\x27\xb1\x97\xfc\x3d\x32\xb1\x77\x03\x81\x34\xc1\x9e\x94\x1e\x0d\xec\x31\xe4\xfb\xc9\x2a\x26\x14\xb8\x1b\xa1\x10\xab\xc2\xec\x13\x16\x05\x48\x18\xdb\x52\x7b\x48\xcf\x75\x71\x1e\xaf\x4c\x44\xc5\x9f\x34\x8b\xef\x3b\x49\x26\x2d\x34\xad\x09\x20\x5b\x3c\x09\x7c\x2d\xcc\x3c\x96\xb1\xa5\x6c\xba\x11\xfa\x88\x34\x66\x48\x5d\x50\x81\x8b\x88\xd0\xa9\x2d\x59\x6a\xc0\x7e\x5c\xd9\x3b\xef\x09\xe7\x84\x4e\x3f\xc6\x9c\x78\x8b\x0c\x9e\x8a\xa1\x9e\x24\xf2\x6a\x12\x79\x90\x3b\x39\x28\xa9\xc8\x41\x3c\xb5\x87\x57\xd5\x65\x1e\x05\x86\xee\x34\x91\xf7\x49\x67\x7d\x5b\x9d\xb5\x96\xbe\x92\x3d\xcd\x5c\xf4\x20\x1f\x94\x0d\x78\x88\x27\x38\xc2\xd4\x4d\xc0\xd4\x62\x52\x1b\x88\xf1\xe7\x23\xa9\x39\x04\xb1\xe7\x49\x3c\x7b\x5e\xa5\xb2\xf5\x9c\xd0\xe5\x8d\x66\x72\x12\x75\x8d\xa4\x25\x38\x4c\xf4\x8e\x09\x0e\xb2\xb0\x20\xbf\x62\xfd\x0c\xd1\x14\x5b\x3f\x39\xf9\x6a\xff\x14\x4c\x20\xdf\xfa\x4d\x04\x0e\xf8\x6a\x13\x6f\x34\x2b\x09\x45\xb1\x91\xdc\xdc\x4c\xad\xc2\x3f\x12\xb8\xe5\xad\x14\xcc\xf5\xcd\x14\x6f\xc6\x4d\x90\xef\x7f\x98\x2c\xe3\x93\x98\xab\x73\x4c\x60\x5b\x39\x25\xf8\xa8\xc2\x09\xa8\x85\xe8\x15\x58\xbd\x14\x37\xa0\x08\x89\x4a\x96\x65\x65\xf3\xc4\x70\x19\x65\xd9\xae\xb4\x53\x72\xc5\xc1\x8d\x10\x22\x3b\xde\x02\x0b\x8a\xa1\xca\x41\x54\xfb\xb1\xdc\x9b\xd2\xe6\x50\x0b\xa0\x9a\x9e\x86\xd0\x4e\x4a\xfe\x4e\xd4\x2f\xae\x40\xdd\x3c\xc2\x68\x2e\x66\x98\x0a\x23\x76\x47\x98\x4a\xa3\xd4\xcb\x35\x0b\xe6\xbe\x20\x23\xf4\xb5\x01\x26\xb9\xba\x11\x20\x8f\x9b\x8c\x7e\x68\xfd\x8a\xfc\x39\xe6\x43\xf8\x03\x99\x2a\x1f\xeb\x10\x46\x38\x44\x92\x17\xd6\x75\xfa\x09\x27\x8c\xaa\x5f\x11\x46\xde\x62\x1d\x26\xea\xda\x81\x75\xf0\x70\xf2\x7a\x5d\x9f\xd8\x11\x3a\xfd\x13\x5a\x4d\x59\x32\x9b\x00\x54\x0f\xe6\x01\x0a\xb0\xdc\x88\xab\x5c\x14\x98\x9b\x62\x78\x1e\x0e\x7d\xb6\xe8\xc2\x1b\x16\xc5\x7a\x04\x76\x3e\x1f\x35\x86\x20\xc6\x65\x39\xb7\x15\x0b\x87\x81\x49\xc3\x69\x82\xd2\xe4\xea\x27\x2b\x27\xc9\x54\xbc\x73\x73\xc9\x3d\x99\x09\x0c\x61\xce\x3b\x18\x71\xd1\xe9\xa9\x8d\xc8\x2a\xf3\x51\xc5\x3b\x1b\x8b\x04\x55\x8c\xad\x69\xe3\x31\x63\x82\x8b\x08\x85\x23\x7d\x33\xd2\x68\x66\x1d\x7c\x2e\xed\x6d\x62\x27\x47\xa8\xd0\x45\x6f\x55\x86\xe0\x21\x81\x3b\x82\x04\xb8\xe9\x90\xa6\x8c\xe7\x5d\x0e\xa9\x19\x7b\xb4\xa2\x64\x8d\x2f\xc9\x6a\xda\x3e\x93\x29\xb2\x8a\xb8\x2f\x95\x0e\xb5\xac\x0b\x60\xbd\x2d\x2d\x98\x5e\x26\xda\xea\x6a\x8e\x15\xa5\xe6\xb7\x56\x13\xa5\x60\x2b\x0b\x02\x5a\x79\x38\xb2\x0b\x45\x59\x10\xd0\xea\x65\x9f\x2a\x8b\xa1\xf0\x54\x5b\x08\x85\xc7\x52\xb9\xe4\x31\x7c\x9b\x32\x6d\xdf\x56\xe7\xe5\xd0\x9f\xfe\xd5\x13\xc2\x86\x59\x4f\x3f\x77\x61\xd7\x77\x52\x8c\x75\x94\xde\xf9\xb8\x6f\x80\xca\x11\x48\xbe\xbc\xcc\x51\x6d\xa6\xc1\x2a\xf1\x1c\xb5\x72\xf6\x96\xef\x63\x55\x62\xb2\x80\xcc\x8e\x1e\x59\xf7\xce\xcb\xdf\xba\x2f\x6c\x54\x75\xb1\x59\x36\xcf\xab\xd5\x06\x61\x25\x80\xdf\x8b\x39\x4a\xc9\x68\x4e\x75\x8f\x32\xf6\x45\x36\x78\x5e\x75\x57\x0a\xcc\x0e\x43\xd4\x16\x49\xe2\xfc\x82\x31\xf3\x62\xf0\x0b\xd4\xb7\x8b\x91\xea\xbf\x00\x5d\x8f\x5c\x14\x22\x97\x88\xc5\xc8\x54\xf2\xca\x24\x39\x34\xdd\x7f\x14\x46\x2e\xd4\xc2\x8a\xcb\xe5\x98\x3a\x58\x28\x24\x06\xf6\xc2\x26\xa2\xb1\x7d\x57\x06\x7d\x03\x1e\x28\x9d\x74\x9d\xfd\xb1\x4f\x3d\xa9\x23\xa4\x05\x32\xc3\xaa\x04\xdb\x15\x86\x19\xba\xc4\x71\xf1\xb3\xd8\x83\x68\x0a\xaa\xc5\x83\x2f\xb5\x81\x4a\x2a\x91\x36\xa1\x7d\x52\x35\x9d\x79\x0b\xe0\x98\x0a\x69\xb9\x99\x65\x02\x1f\x3f\x1c\x1d\xaf\x55\xe1\xad\xa3\x4c\x94\xd5\x68\x5b\x6d\x54\x16\x68\x9c\xcb\xb0\xbe\x52\x97\x70\xa6\x35\xa4\x5c\x7f\xce\x85\x7c\x6e\xec\xb8\xb8\xb0\x1c\xa1\xcb\xb6\xae\x65\x66\x65\x2e\xb5\x44\xe8\xaa\x5f\x82\x29\xf6\x95\xff\x75\x19\x9d\x90\xe9\xbc\x14\x04\xc1\x24\x00\x6a\xd8\x9d\xdf\x0b\x5f\xcf\xdb\xa9\x79\xbb\x2e\xf3\xe9\xb6\x9c\x39\x35\xd6\x74\xe1\x4b\x5d\xd8\x17\x10\xcc\xb9\x90\xe0\x70\x93\xb4\xe0\xb3\x2b\x1c\x75\x5c\xc4\x31\x20\x3f\x9c\x21\x3a\x0f\x70\x24\x8d\xd8\x19\x8a\x90\x2b\x70\xc4\x81\x45\xd0\x6e\x77\xda\xed\x75\xb9\x4a\x22\x13\x66\x8c\xa8\x6e\x3f\xc6\xc2\x6e\xbd\x0e\x88\xaa\xc8\x8b\x6c\xab\xc2\xa8\xba\x9d\x8b\xa8\xf2\xee\x8d\x31\xf8\x8c\x4e\x25\x32\x66\x88\xc2\x66\xdf\xfa\x7c\xb7\xbd\x8c\x22\x45\xb3\xbd\xa4\xfa\x9d\x6c\x72\x87\x5c\xd0\xc4\x62\xcb\x40\xf1\xd9\x2c\x57\x97\x51\xaa\xa5\x7e\x61\x0c\x20\x1c\xcc\x30\x12\xe7\x94\x09\x75\x7f\x2c\xc7\x22\x66\xa5\xf5\xda\xee\x8c\x5a\x53\x4b\x6e\x1c\x48\x77\x2a\x7a\x05\x02\xbe\xc4\xd1\x02\xb6\x20\x20\x74\x2e\x30\xef\x2a\x04\x79\x78\x82\xe6\xbe\x80\x4b\xb9\xbd\x91\x80\x28\xce\x5d\xca\x8d\x00\x74\xee\xfb\x12\x64\x2d\xaa\x8d\x4d\x5a\x28\x72\x72\x5f\x36\x64\x01\x90\xfb\x37\x22\x33\x20\x3d\x16\x2b\x32\x03\x74\x2b\xa5\x71\x5a\x38\xe2\x5e\x29\x9c\x82\xf1\x40\xe8\x5b\x59\x65\xfb\xe1\x52\x57\x83\xdc\x2a\xae\xdf\x52\x33\xa0\xbd\x9b\x75\x72\xb4\xeb\x2c\xb2\x9c\x47\x38\x3b\x50\x6a\xd1\x48\xe1\x25\xa7\x9c\x94\xcd\xd4\x8c\xd0\x85\xcf\x46\x84\xb5\xdb\x19\xc0\xda\x6d\xf0\x09\x3d\x5f\xae\x20\x48\xcd\xe7\x4f\x28\xb9\x90\x12\x4f\xc5\x1e\x4e\x88\x2e\x3e\x2b\x21\x31\x1f\x5f\x3a\xb8\x47\x78\xe8\xa3\xc5\xa8\x5e\x31\x1f\x58\x4a\x39\x67\x9a\x48\x53\xca\x0c\x02\xe1\x3c\x0a\x19\xc7\x0d\x94\x5e\xfd\xe7\x7e\x9a\x07\x88\xc2\x24\x22\x98\x7a\xfe\xa2\x64\x76\x59\x18\xd6\x15\x10\xb1\x93\xed\x14\x5d\xf1\xd3\xe5\x10\x2c\xd3\x78\xed\x58\xe5\x95\xcc\xd9\xd2\x74\x6a\xfa\xca\xd5\x47\xe8\x54\x1a\x0c\x1f\x8e\x5e\x27\x16\x4b\x11\x88\xac\x06\x2a\x33\x2b\x6d\xcf\xaa\xc5\xd9\xe5\x6c\xfc\x3a\xfd\x25\x51\x83\x62\x4b\x41\xfd\xdb\xbd\x3f\x1e\xd7\x30\xb7\xdb\x8f\x8e\xb9\x0d\xfe\xca\x98\x3a\xc7\x65\x07\x5d\xf8\x95\x44\x53\x42\x09\xba\x6b\x6e\x33\x40\xdc\x15\x97\xe9\x8f\x29\x03\x69\x08\x13\xe4\xf3\xd4\xe9\x98\x94\x74\x1a\x65\x3c\x7f\xd5\xfb\xcf\xf6\x71\xd1\x44\x53\x3d\xac\xea\x50\x71\x75\x6b\x3d\x8d\x12\xf0\xf2\x2a\x41\xab\x03\xc8\xea\xb3\x32\x2c\xc6\xdb\xc1\x26\xac\x7a\x95\x22\x34\x52\x26\x61\xdc\x19\x7c\x3c\x11\x10\xca\x65\x6c\xcf\xa0\x29\x98\x19\x28\x4b\x35\x56\xbd\xb6\xd2\x4b\x63\xd7\x00\x23\x95\xfe\xbe\xc0\x41\xab\xa1\x44\xd0\x4f\xaa\xc8\x66\x35\xc9\xec\x9c\xb3\x91\xee\xe5\xa2\x24\x2e\x29\xb0\x93\x2d\x29\x00\x84\xc2\xfb\x9d\xa3\xce\xd1\xd1\x87\x64\xd7\xac\xe9\xbf\x6b\x76\x1f\x2a\x22\x29\x63\xca\xb7\x6f\x62\x4c\xdd\xdd\x51\x65\xf1\x10\x31\x3b\x53\x7d\x48\x00\x53\x4c\x55\x84\x94\x07\xf3\x58\xce\x24\xb5\xd9\xb2\xa1\xfb\xf9\xb0\x80\x95\x4e\x2d\xb2\xdf\x6e\x3c\x94\xdd\xed\x6e\x46\x74\x7d\x82\xa9\x68\x72\xc4\x9a\xeb\xc1\xb1\x1b\x15\x93\xa6\xee\xe8\xa0\xe7\xce\x4f\x5f\x56\x3f\x4a\x28\xcd\xee\x6a\x95\x2c\x9c\xdc\x59\x6c\x6e\xfd\x94\x7b\x96\x04\x33\x53\x2c\x66\x84\xb4\x6b\xd6\xfc\xea\xce\xa5\xd5\x3c\x2b\x35\x1c\x5e\xae\x49\xcb\xd9\x31\xfb\x91\x1d\xfb\x77\xc1\xbf\xda\xec\x53\x05\xf2\xad\x40\xba\xb2\xe3\xa0\x72\x71\x5b\x4e\x42\x9e\x92\x10\xc5\xc1\x95\xb6\x8a\x48\x55\x08\xa1\x46\xbb\xb5\x57\x23\x52\xe5\xf9\x5a\x16\x90\x92\x6f\x2f\xa5\xd0\x32\x5f\x6c\xf6\x0b\x13\x1f\x4d\x81\x68\x65\x29\x4d\x8a\x2b\xdb\xd8\x8d\x67\x19\x53\x30\x8b\x04\x73\x4b\x40\x6a\xa4\x98\x8f\xdd\xc4\xd8\xad\xf4\x3b\x17\xeb\xbb\x55\x93\xed\x6f\xab\x6e\x2a\x25\x7a\x16\x00\xdd\xec\xbb\xa8\xb7\x86\x22\xa6\xf6\x2b\xa5\xfa\x23\xfb\x99\xe4\x22\xd0\x5b\x61\xef\xc6\x9a\xa7\x48\xdf\x42\x2d\x26\xb9\x34\x54\x3a\x9d\x40\x41\x78\x17\x4a\xbf\xb2\x4f\x1e\x1c\x2f\xbb\x25\xac\xc4\x50\x71\x85\x55\xfa\xc0\x6e\xe0\xd7\x2a\x8e\x5e\xf4\x4b\x95\x1c\x6c\xae\x90\x85\x1d\xcb\x84\x15\xbc\x54\xf9\x5d\x6e\x2d\x5e\xef\xd5\xa5\x55\x3e\x55\x1b\x85\x55\x27\x79\x99\x78\x49\xc8\x45\x41\x3e\xcb\x24\x9a\xc6\x61\xfa\x71\xc2\xe9\x33\xd5\xa6\x34\x45\xf1\x2e\x59\xa3\xf4\x03\x25\x27\xe7\x3d\x3a\x0e\x8f\x9e\x3b\x3f\x79\xf3\x8f\x78\xe0\x3b\x82\xbd\x38\x3b\x9a\xf6\x77\xdf\x7d\x9d\xcc\x1b\xf0\x52\x2d\x27\x15\x40\xf8\x66\x4c\xf4\x48\xf8\x2d\xc5\x84\xb1\x9a\x92\xdf\xc3\xd5\x0c\x1c\xcd\x53\xc3\x82\x29\x50\xe0\x10\xe4\x79\x44\xca\x28\xe4\x7f\xac\x40\x74\x29\xa6\x2e\x75\x48\x60\x61\xfc\x06\x7b\xf5\xba\xe8\x6f\x3d\xac\x26\x7f\xf6\x13\x0d\xe7\x9d\xc8\xfa\x22\x68\xf9\x80\xdf\x54\xbf\x10\x2a\xb6\x07\xd9\xa9\x15\xbb\xeb\x3b\xc5\x4a\x7a\x7b\x6c\x3e\xf6\x71\x8d\x71\xa5\x06\xb4\xd7\x74\x3e\x03\xef\x1b\xac\xea\xfc\x27\xee\x65\x5d\xdb\x40\xfc\xdd\x57\xb6\x8d\x8b\x96\xcd\x0c\x6f\x74\x82\x18\x61\xf4\x10\xf3\xb9\x2f\xb2\x0c\x6f\x4d\xc3\x1e\xe1\x81\x49\x83\x87\xbd\xea\x8a\x97\xc6\xaf\x88\xbe\x67\x6a\x07\x46\xd9\x95\xb9\xf9\x9e\x68\x07\x3c\xa3\xfe\xc2\x72\xb7\x4e\x08\xf6\xb5\x87\x58\x87\x9d\x26\xdd\x0b\x86\x74\x05\x87\x66\xcf\xc0\x93\x17\x7f\xa1\x18\x81\x02\x0e\x96\x07\x02\xc0\xda\x5a\x31\x31\x27\x5d\xf4\xfa\xb2\xe8\x24\xeb\xad\x10\xb1\xb1\xff\x5a\x1a\xdf\x11\x76\x59\x94\x94\x29\xc9\x65\x25\x95\x50\x83\xd0\x21\x84\x48\xcc\xf2\xec\x95\x12\x26\xce\xb9\xcf\xc2\x11\x3f\xb5\x86\xb1\x6f\xb7\x2e\x40\xe7\x63\x3a\x15\x33\xb5\x3d\x20\x81\xda\xd1\x1b\x4c\x29\x36\xba\x9a\x11\x77\x26\xe9\x11\xa9\xb4\x4e\x7d\xc3\x65\x26\x8d\xb4\xb4\x8a\x6f\xf9\xfc\xf2\xeb\xb0\x7c\x15\x26\xc7\x13\x5b\xa9\xec\x20\x94\x04\xf3\x60\x08\xbd\xf4\x91\x8e\x0a\x1b\xc2\x60\xb3\xef\x98\xa7\xc5\x14\xad\x3c\x8a\x20\x59\xe5\x66\xf4\xb8\x08\x41\x8e\x96\xe6\x69\x53\x1c\xc6\xed\x55\x9a\x2e\x76\x19\xf5\x38\x8c\xb1\xb8\x52\x37\x2a\x22\x81\x20\x29\xde\xf2\x6d\x31\xb6\xe9\x34\x42\x59\xcf\x79\xe1\x54\xe3\x2c\x8f\x12\x0b\x67\x66\x7c\x93\xf5\x9c\xc5\x99\x79\xd8\x04\x65\x71\x95\xd9\x78\xd3\x21\x18\x4c\xb0\x70\x67\x5d\x78\x23\xff\x93\x49\x7c\xbe\x9a\x61\x0a\x38\x08\xc5\xa2\xab\xfb\x61\x2a\x54\x55\x1a\x14\xa5\x6b\x5f\xe0\x88\xa2\xb8\x8f\x82\x27\x59\xe7\xe5\x78\xcd\xea\xda\x82\x96\xad\x70\x7b\x1a\x2c\xc7\xc9\xd1\x76\xe6\x97\xc6\x81\x95\x91\x56\x8b\x80\x8f\x68\x2a\x99\xc6\xc3\xd7\x05\x96\xb0\xcf\xe4\x1a\x48\x89\x22\xf9\xf2\xf9\x68\x86\x74\x71\x30\x88\x9d\x88\xa6\x81\xb6\xf2\xe6\x6a\x81\x3e\x48\x2f\xb4\x95\xf8\x92\xbc\x8e\x91\x3b\xb3\x27\x7d\x87\xd3\xc8\x27\xcc\x25\xd3\x70\x1c\x3d\x11\x73\x89\x58\xa9\x1b\xf0\x7f\x3b\x49\xcf\x23\x9d\xca\x62\xce\xab\x55\x27\x18\x2f\xc0\x8d\x88\xc0\x11\x41\x3a\x64\x8c\x2f\xa8\x40\xd7\xc9\x41\x76\x22\xea\x81\x70\x0b\xa0\x80\xf8\x48\xc5\x38\x8a\x5c\x17\x0c\xa7\xf1\xc0\xa7\xe0\xfa\xea\x2a\x60\x36\x01\x44\xe1\xe8\xd3\x3b\x15\x8f\x8b\x03\x4c\x45\xaa\x7a\xf6\x24\xde\x74\x75\x11\x73\x1b\xb0\xea\xaf\xcf\x4b\x11\x5d\xc4\xc3\x4e\x98\xef\xb3\x2b\xb9\x3f\x3f\x3d\xb7\x82\x5a\xf9\xa9\x56\xf4\x7c\xb8\x96\x0c\xf9\x43\x79\xe6\x8b\xf5\x3e\x1b\x71\x9a\x79\xa1\x0e\xef\x46\x56\xde\xf6\x0f\x96\x47\xcc\x7a\x38\x8b\xf0\xc4\xfa\x99\xe9\x90\x71\x67\x5b\xcf\x0b\x79\x60\x3f\xd8\x07\x1a\xf2\x27\x8b\xa6\x88\x12\x1e\x67\xfd\xd9\x6f\xa4\xd5\x62\xfd\x5e\x9a\x7a\xf6\x83\x71\x45\x5b\x0f\x72\xe1\xd0\x3f\x58\x09\x39\xd6\x43\x93\x1c\x93\xe2\xd3\xca\x74\x5a\xb7\xf4\x9f\x14\x4d\x59\x8b\x83\xdb\xb4\x13\x33\x4c\x22\x35\xbf\x75\x88\x2f\x84\x4e\x89\xa8\x79\xc6\x22\xda\xe9\xe9\x29\xbf\x48\x93\x54\x95\xc7\x14\x71\xd7\x7e\x9f\x36\x3e\x5e\x1d\x08\x18\x21\xea\x8d\x92\xe3\x5b\x39\xef\xdb\xc0\xb5\x6e\x71\x45\x35\x9c\xfb\x9a\x77\xed\x45\x44\xdb\x22\x8e\x3d\xf1\xd6\xa5\xb1\x47\x74\x9b\x24\x44\x53\x09\xf8\x75\xf9\x2c\x25\x9d\x3e\x58\x90\xdb\x11\x2d\xec\xad\x19\x4a\x80\xba\x89\xe8\x08\x7d\xe6\x65\x0d\xd6\xa2\x38\xc9\x49\x0b\xb0\x24\x4a\x3c\xbb\x56\x85\x10\xd4\x52\xd2\x0c\x70\x5b\x41\xc7\xc5\x42\xda\x95\x52\x8f\x6b\x71\xac\x2e\x36\x2c\x17\x62\xa9\x0c\x53\x8d\x52\x99\x65\xf1\x44\xbd\xf0\x5a\x22\xb4\x54\x0c\x71\x56\x62\xa5\xdf\xcc\x48\x2e\x30\x17\xbc\x1b\xb9\x13\x9f\xfb\x68\xe8\x15\x75\x4e\xb3\xe2\xe5\x74\x1d\x4e\x25\xe2\xe4\x7f\xd5\x2a\x96\xff\xd0\x6b\xf3\x54\x07\x4c\x9f\xea\x85\x79\x9a\x8e\x2d\x77\xac\x28\x42\x82\x45\x9a\xe0\xa7\xff\xfd\x3f\xb2\xd7\x8f\xa7\x8a\x65\x4e\xdf\xed\xff\xb2\x77\x9a\xca\xd0\xb8\xd7\x19\x23\xd4\xb4\xdf\x39\x78\x7d\xaa\xc7\xfe\x70\x78\xda\x85\x9f\xd8\x95\x34\xfe\xd7\x61\xc1\xe6\x4a\xce\xca\x59\xa6\xf9\x04\x6c\x02\x3d\xc7\x74\x57\xd5\x49\xcc\x6c\x14\xed\x2d\x1c\xef\x25\xcc\x54\xb6\x14\x8b\xfb\x0f\x53\xb8\x5a\xb1\xd5\x69\xb0\xe8\x28\xc9\xad\xe1\xb2\xce\xca\x54\x64\x5a\xd3\xc5\x98\x5d\x89\x3f\x42\x3c\xaa\x8e\x3c\xcf\x20\x1e\x7e\x04\x74\xc5\xed\xce\x7f\x84\x9d\x3f\x9b\x83\x8e\xf4\x37\xd4\x4d\xf5\x2a\x46\xde\xd4\xc4\x3a\x0d\x16\x37\x04\xd7\x27\xe7\x18\x82\xc5\x3f\xfa\x5b\xdf\x44\x5e\x28\x69\x58\xdc\x08\x72\x4b\x8e\x20\x91\xde\xea\x3f\x43\x1c\x42\x1c\x05\x84\x73\x75\x2e\xc3\x80\x63\x5d\x7b\x31\x32\x85\x6b\x2c\xd2\x1f\x30\x81\xbb\x31\x80\x5a\x5f\xa7\x45\x4e\x24\x1b\x9b\x62\x15\xea\xe0\x33\xee\x5d\x2d\x96\x8c\xbd\xa5\xd8\xac\x42\xd8\x94\x0b\x96\x12\xf3\x28\x23\x37\x20\x2f\xce\x1a\xb0\x48\xeb\xe6\x42\xab\xf4\xe4\x3a\xde\x39\x15\xad\x80\x8a\xa4\xa5\xec\x49\xb2\xdc\x03\xa8\x1d\x44\x46\xee\x8f\x17\x15\x78\x6a\x00\x75\x53\x54\xe2\x4b\xe4\x8f\x2a\x0f\xe3\x63\xb4\xe2\x4c\xa5\x3a\xd9\xd8\x43\x91\xb7\xbc\x5f\xdc\x52\xf6\x8d\x2b\x2e\xa9\xf0\x90\x18\x04\x53\x72\xc9\x9e\x17\x1e\xc2\x58\x3d\x35\x0f\xf5\x8f\x37\x66\xef\xf7\xf3\xe7\x38\x13\x49\x4f\x7a\x26\x44\xb8\x96\x9f\xd8\xc9\x51\x26\x6e\x3b\x1e\x3e\xe7\xe1\x32\xe9\x26\xd0\x4a\x52\xb9\xd3\x29\xe6\x12\x94\xa0\x65\xf1\x4c\x4c\xed\x56\x7c\xd1\x4d\x48\x44\x92\x94\xb9\x77\xb2\xd2\xa7\xf1\xbc\x73\x85\xef\xe8\xd3\x25\x59\xad\x15\x9f\xd7\xde\x67\x72\xf4\x65\xfb\xf0\xd3\xe6\xcf\xbf\xec\xbf\xf8\xe4\x7c\x38\x0e\xce\x3e\xbd\xf1\x36\x99\xfb\xe6\x70\x9a\x7e\xce\xf8\xb4\xd5\x62\x4a\x9f\x2e\x4d\xac\xdc\x68\x34\xb8\xa9\x7c\x00\x2d\x55\xb2\xa0\x29\x06\x92\xbc\xad\xbc\x93\xae\x9a\x9a\xda\xff\x07\x2d\x14\x92\x91\x02\x70\x64\xf0\x57\x83\xd7\xf4\x55\x79\x52\xbd\xdd\xb6\xd3\x23\x7c\xb1\x1d\x5d\x6c\x9e\x9d\x93\x17\x17\x0e\x13\xc1\xd9\xc5\x44\x4e\x77\x12\x4d\xbb\x28\x0c\x79\x37\x38\xef\x8c\x85\x98\x3a\x67\xb4\xf7\xdc\x99\x85\xdd\xeb\xad\xf9\x8b\x2e\xef\x75\x3d\x7c\xc9\x67\x64\x22\xba\x2c\xb2\x10\x63\x1d\xc8\x43\xab\xef\xf4\x9d\x4e\xcf\xe9\x38\x5b\xc7\xbd\xfe\x70\xab\x37\xec\x0f\xba\xce\xd6\x66\x6f\xd0\xff\x3d\xed\x61\xe5\xd9\x17\x7a\x6c\x0f\x37\xb7\xbb\x9b\xdb\xfd\xbe\xf3\xc2\xea\x11\x27\xc4\x43\xab\xdf\xdd\xee\x3a\xe9\x8b\xec\xa2\x4e\x16\xbb\x85\xe8\x0a\x6f\x68\x4a\x0f\x9b\x13\xdf\xa8\x74\xfd\x5d\x13\x0a\xa0\x93\x51\x1f\x17\x77\xea\x82\x03\x4f\xec\xf9\x5d\xd9\x33\x5b\xe5\x01\x5a\xc8\x94\xd2\xb1\x8c\x9d\x38\xb0\x30\x09\x33\xc9\x13\xea\x0e\x38\xb9\x2c\xc9\xab\x82\x6d\xcb\x32\xd5\x5a\x59\xa6\x2e\x93\xe4\x99\x67\x99\x30\x7d\x68\xed\x04\xe8\x2b\xa3\xf0\x19\x8f\xe3\x20\x15\xab\x6d\x05\xb0\x4d\xd4\x4f\x31\xe5\x2a\x07\x68\x09\x93\xe6\x40\x3b\x39\x82\x3d\xc4\xc5\x3a\x58\xd1\xff\x75\xb0\x41\x5d\x8c\x3d\xfc\x91\x5a\x0a\x7f\xa6\x6c\x16\xc7\xb8\xc3\x1f\x96\x69\xf1\x6f\xeb\xdf\x25\x44\x4e\x07\x5a\xcf\x35\x5c\x9a\x97\x2e\xff\xd2\x4a\xef\x1a\x8e\xba\xb0\xca\x0a\xe4\x1a\x04\x05\x8b\x0e\x0a\xc3\x0e\xb7\xb0\x92\x2d\x40\x93\x8f\x96\x9a\xb0\x08\x82\x05\xa0\x30\x2c\x8b\xb8\x6d\x22\x32\x0b\x82\x31\x3b\x44\x23\x09\x99\xbd\xbc\x8f\x6f\xf4\x0a\x0c\x7b\xeb\x89\x41\x26\x60\x0f\x5a\x47\x3b\x9d\x5e\x5f\xfe\xaf\xf0\xda\xc4\x5b\xcb\x21\xe5\x3f\x8a\x12\x53\x1a\x3f\x1d\xb9\xb3\xa9\x16\x4e\xbd\x8e\x33\xe8\x38\xcf\x8f\x7b\xdb\xc3\xfe\x60\xe8\xf4\xfe\xcb\xd9\x1a\x6e\x3a\x65\x28\xb6\x6e\xa9\xfa\x9b\xa0\xf9\x9b\xa0\x31\x17\x8e\x76\x1b\x54\x16\xc3\xbd\xfe\x2e\x28\xad\x8a\xcb\xaa\xc0\x66\x31\xbc\x60\xa4\x04\xf5\x68\x34\x84\xd4\xa2\xc0\xd1\x68\x1c\xb1\x73\x1c\x09\x16\x12\xd7\x9c\x31\x8d\xc6\x0b\x81\xf9\x88\xd0\x51\xb6\x66\x20\xa8\xfd\x64\xf0\x95\x8c\x08\x1b\x19\x27\xb9\x19\xac\x93\xbf\x6f\x03\x40\x8d\x38\x84\xd1\xc8\x65\x94\xcf\x03\x1c\x8d\xd8\x64\xc2\xb1\x75\x77\x62\x31\x5e\xa9\x63\x45\x2d\x40\x6f\xbb\xd7\xdb\x7e\xee\xf4\x37\x1d\xc7\x71\x32\x92\xdb\xec\x25\x5f\x0c\x7a\x5b\x83\x65\xbd\xb7\x2b\x7b\x6f\xbd\x78\xf1\x62\x59\xef\x97\x95\xbd\x9f\x6f\xf7\xfb\x36\x5d\x4a\xe2\x6a\x1e\x2f\x65\x96\x52\xa1\x40\x81\x81\xe3\xa8\x4b\x9b\x96\x1a\x1a\x7a\x95\x3b\x9b\x85\x75\x6e\x15\xf7\x83\xfa\x65\xad\x7c\x4c\x7c\x23\x33\x88\x2a\xc1\x08\xad\x5f\x76\xde\xfc\xb2\x73\xd4\x79\xff\xf6\xfd\x71\x27\xf3\x3e\xb1\x1a\x8f\x16\xd4\x9d\x45\x8c\xb2\x39\x07\xe4\xc6\x71\x17\x94\x89\xd4\x16\xd1\x6e\x3d\xc4\x17\xd4\xfd\x51\x05\x3c\x24\xae\x38\x6b\x51\xdb\x65\x19\xe5\xde\xe4\xf3\x3e\x09\x2e\xde\xba\xd1\xeb\xf9\xbb\xed\x1e\x3a\xb9\xde\xff\xfd\xe2\xd5\xf1\xc5\xc1\xa1\x91\x2c\x03\xc7\x89\x37\x3c\x4f\xf8\x29\xc7\xcf\xbe\x76\x23\x36\x58\x41\x6a\xc8\xfe\x1d\xa0\xa8\x5f\x8f\xa1\x7e\x19\x82\xf4\xee\x15\x04\x93\xd3\xe6\x38\xe3\x25\x1f\xc2\x89\x32\x73\xe5\x5b\x75\x5b\x76\x66\x5b\x62\x2e\xbf\xc9\x6f\xe9\x86\x90\xfd\xe6\x10\x96\x7d\x22\x8d\x6f\x72\x99\x3f\x0f\xa8\xf6\x2b\xcb\xc1\x8d\x1b\x14\xda\xc4\x6b\x77\xe1\xa8\xac\x9d\x3a\x1b\x18\x9a\xdd\xe7\xba\x39\x9b\xcb\x6e\x60\xe3\xa7\x7a\xbf\xdb\x85\x4f\xda\xd3\xab\xe9\x33\x04\xe2\xc1\x8f\xd0\xb3\x91\x93\xa7\xb6\xff\xf9\xf5\xdb\xf9\x62\xbc\x1f\xed\xd1\xeb\x68\x07\x07\xcf\xfb\x83\xe9\xc5\xf9\x39\x79\x7d\x99\x50\x7b\x49\xc5\xee\x52\x8a\x17\x8d\x83\xd5\x29\xde\xab\xa7\x78\xaf\x84\xe2\x81\x06\x55\x45\x1f\xa5\xbc\x3e\x4c\x4a\xcc\xdf\x06\x0f\xf9\x92\xd2\x65\xf3\x7e\x7e\xfb\x69\x3f\xaf\x9d\xf5\xf3\x92\x49\x1f\xa7\x79\x7a\xd8\x4b\x0b\x6a\x79\x0c\xab\xd3\x08\x7c\x9d\x04\xb0\x0e\x9c\x81\xbe\xaf\xef\xa1\x4e\xc5\xf8\x9e\xcc\x0c\xf4\x8d\x26\xde\x8f\xed\x1e\xf9\x65\xd3\x9b\xff\xfa\x65\xff\xf2\x72\xeb\xcb\xe5\x3b\x7f\xf1\xb5\x17\xbc\x3d\xdc\xfc\x79\x71\x71\xd0\x4e\x0b\x93\xd7\x88\xb4\x2f\x1f\x9e\x4f\xfb\xd3\xed\x9f\x8e\xbd\x93\x5f\x4e\x50\xff\x9c\xff\xf4\xa2\x7f\xfe\xe9\xf5\xe6\x22\xc6\x4b\xbe\xa2\x7a\xa9\xa8\xbf\x03\xa6\xee\xd5\x33\x75\xaf\x8c\xa9\x53\x41\x75\x89\x23\x32\x59\xc0\xcf\x9f\x8f\x75\xdd\xfa\x21\x1c\xc6\xb1\x82\xf1\xf5\x6a\x26\x67\x47\x55\xb5\x6f\x84\x99\xcd\x93\xd9\xde\xec\x2a\xf8\xed\x55\xf8\xf9\xe3\x64\xbf\xef\x1f\xe0\xf3\xd0\x1b\xfc\xfe\x3a\xc6\x4c\xfe\xce\xee\x32\xcc\x0c\x6e\x8f\x98\x41\x2d\x5e\x06\x65\x68\xe1\x38\x82\xf6\x84\xb1\xce\x18\x45\xed\x58\xf5\x2d\xbb\x66\xae\x5b\x23\x02\xbe\x6c\x9e\x90\xbd\xd9\x57\x6a\xe1\xe2\x2c\xf4\x06\x5f\x76\x13\x5c\xbc\x47\xd7\xe6\xf0\x76\xdf\x78\x2e\x0e\xb5\x2f\xa2\x01\x92\xb6\x6e\x8f\xa4\xad\x5a\x24\x6d\x2d\x47\xd2\x0c\x25\x79\x8e\xd6\x71\x32\x4d\xc2\xa3\xb6\x01\x99\xb3\xe9\xe4\x30\x72\x29\xc2\xce\xaf\x25\xc2\x7e\xfd\x88\xf7\xfb\xec\x00\x9f\x79\x9b\xbf\xbd\x4a\xf0\x75\x8c\xa3\x80\x1f\x30\xb1\x63\x2a\x1f\x37\x59\x65\xfd\x3b\x58\x65\xfd\xfa\x55\xd6\x2f\xc1\x54\xb2\x92\x84\x84\x59\x57\xf3\xd3\x55\xd1\x30\x85\xb8\x72\x73\x25\x2e\xce\x7f\xdb\xfd\xfa\x59\xa1\x20\xc6\xc5\xbb\xcb\x37\x2f\xcf\xde\x7f\xfa\x12\xe3\xe2\xe5\x01\x0a\xf0\x2e\xa3\x13\x9f\xb8\x4d\x1c\x42\x9b\xdb\xb7\xc7\x83\x3d\x46\x09\x1e\xec\xd7\x59\x11\x9c\xd4\x64\x53\xe6\x0a\xe1\x80\x7c\x75\xcc\xa3\xca\x42\x57\x22\x61\xfb\xfc\x8b\x23\x19\xe2\x6b\x8a\x8d\x2f\x78\xe6\x6d\xee\x19\x61\x52\xbc\x6d\xa0\x6c\xe2\x2f\x6f\x3f\xef\x97\xb5\xd3\x7e\x59\x2a\x63\x4d\xe1\xe8\xf8\x16\x87\x1a\x91\x89\xf7\x62\xda\x6e\x7f\x99\xce\x26\xef\x5f\x4e\xdf\x1e\xf2\x9f\x2e\xf7\x3e\x27\xb3\x6c\xac\x64\xef\x65\xae\xfa\xe0\x3f\xae\x26\x0e\x72\x73\xc0\xb1\x18\xc2\x87\xdd\xf7\x9d\xbd\xdf\x3a\x2f\x87\xc6\x17\xaf\xcb\x7f\xcb\x99\xa4\x6d\xf0\xb5\xe8\x64\xce\x26\xae\x9d\x4d\x9f\x7a\x7e\x70\xe1\x5c\x4c\xdc\xe7\x9c\x08\xb4\xc5\xfd\xb3\xcb\x17\xf6\x2e\x56\x9a\xbb\x31\x43\xc9\x69\xf7\xa6\x5b\xde\x8b\x17\x17\x8e\x1f\xb9\xde\xe5\x60\xfa\x1c\xf9\xe3\xe7\xdc\x9f\x4c\xe9\xd9\xa6\x37\x1b\xf3\xb3\x7f\xfc\xbf\x7f\xee\xfd\x76\x7c\xb8\x03\x3f\xe8\x39\x76\x15\x52\x7e\x4c\x0b\xe8\x58\x63\x13\xae\x2f\x30\x59\x57\xb3\x57\x3f\x77\xdf\x9d\x1c\x1d\xef\x1d\xc6\xaa\xc3\x19\xb4\x55\x24\x41\x42\x47\xbb\x12\x8f\x6c\xdf\x9b\x6e\xb1\x68\xcb\xb9\x24\x73\xe7\x39\xc3\x92\x4a\xb3\xe8\xdc\xed\x6f\x7b\xd3\x89\x38\xeb\x21\x37\x73\xf5\x47\x5c\x2e\xa4\xbd\x6c\x12\x96\x61\xf2\xaf\x3a\xfd\x7b\xcc\x3f\x47\x8b\x6d\xca\x2f\xc6\x7d\x7e\x10\xbc\x39\xdb\x1a\xff\x16\xbe\x7e\xbe\x8b\x5a\x6b\xff\x17\x00\x00\xff\xff\x1b\x05\xa5\x5c\x57\xc7\x00\x00")

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

	info := bindataFileInfo{name: "kas-fleet-manager.yaml", size: 51031, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
