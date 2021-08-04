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

var _kasFleetManagerYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x7d\x79\x73\x1b\x39\xb2\xe7\xff\xfa\x14\xb9\xf4\x4e\x70\x66\x57\xa4\x8a\x87\x0e\x33\x9e\x5f\x84\x2c\xc9\x6e\xb5\x6d\xd9\xd6\xd1\x6e\xf7\xc4\x04\x05\x56\x81\x24\xa4\x2a\xa0\x04\xa0\x24\xd1\xf3\xe6\xbb\xbf\x00\x50\xf7\x45\x52\x87\x45\x75\x4b\x13\x13\x6d\x92\x40\x22\xf1\x43\x22\x33\x91\x48\x00\xcc\xc7\x14\xf9\x64\x00\xbd\xb6\xd5\xb6\xe0\x15\x50\x8c\x1d\x90\x53\x22\x00\x09\x18\x13\x2e\x24\xb8\x84\x62\x90\x0c\x90\xeb\xb2\x1b\x10\xcc\xc3\x70\xb8\x7f\x20\xd4\x57\x97\x94\xdd\x98\xd2\xaa\x02\x85\x90\x1c\x38\xcc\x0e\x3c\x4c\x65\x7b\xed\x15\xec\xba\x2e\x60\xea\xf8\x8c\x50\x29\xc0\xc1\x63\x42\xb1\x03\x53\xcc\x31\xdc\x10\xd7\x85\x11\x06\x87\x08\x9b\x5d\x63\x8e\x46\x2e\x86\xd1\x4c\xb5\x04\x81\xc0\x5c\xb4\xe1\x70\x0c\x52\x97\x55\x0d\x84\xdc\x31\xb8\xc4\xd8\x37\x9c\x24\x94\x1b\x3e\x27\xd7\x48\xe2\xc6\x3a\x20\x47\xf5\x01\x7b\xaa\xa8\x9c\x62\x68\x78\x88\xa2\x09\x76\x5a\x02\xf3\x6b\x62\x63\xd1\x42\x3e\x69\x85\xe5\xdb\x33\xe4\xb9\x0d\x18\x13\x17\xaf\x11\x3a\x66\x83\x35\x00\x49\xa4\x8b\x07\xf0\x01\x8d\x2f\x11\x9c\x98\x4a\xf0\xce\xc5\x58\xc2\x27\x4d\x8a\xaf\x01\x5c\x63\x2e\x08\xa3\x03\xe8\xb4\x3b\xed\xee\x1a\x80\x83\x85\xcd\x89\x2f\xf5\x97\x35\x75\x4d\x5f\x8e\xb1\x90\xb0\xfb\xe5\x50\x31\x69\xf8\x0b\xeb\x10\x2a\x24\xa2\x36\x16\xed\x35\xc5\x2f\xe6\x42\xb1\xd4\x82\x80\xbb\x03\x98\x4a\xe9\x8b\xc1\xc6\x06\xf2\x49\x5b\xa1\x2d\xa6\x64\x2c\xdb\x36\xf3\xd6\x00\x72\x1c\x7c\x42\x84\xc2\xdf\x7d\xce\x9c\xc0\x56\xdf\xfc\x03\x0c\xb9\x72\x62\x42\xa2\x09\x9e\x47\xf2\x44\xa2\x09\xa1\x93\x52\x42\x83\x8d\x0d\x97\xd9\xc8\x9d\x32\x21\x07\x3b\x96\x65\x15\xab\xc7\xbf\x27\x35\x37\x8a\xa5\xec\x80\x73\x4c\x25\x38\xcc\x43\x84\xae\xf9\x48\x4e\x35\x02\x8a\xcd\x8d\x4b\x05\x91\x18\x7a\x13\x4f\x6e\x5c\x77\x06\xba\xf6\x04\x4b\xf3\x0f\x50\x02\xc8\x91\x22\x73\xe8\x0c\xd4\xf7\xbf\x99\x31\xfa\x84\x25\x72\x90\x44\x61\x29\x8e\x85\xcf\xa8\xc0\x22\xaa\x06\xd0\xe8\x5a\x56\x23\xf9\x08\x60\x33\x2a\x31\x95\xe9\xaf\x00\x90\xef\xbb\xc4\xd6\x0d\x6c\x5c\x08\x46\xb3\xbf\x02\x08\x7b\x8a\x3d\x94\xff\x16\xe0\xff\x72\x3c\x1e\x40\xf3\xd5\x86\xcd\x3c\x9f\x51\x4c\xa5\xd8\x30\x65\xc5\x46\x8e\xc5\x66\xaa\x72\x06\x96\xb0\x1c\x78\xd9\xbe\x88\xc0\xf3\x10\x9f\x0d\xe0\x18\xcb\x80\x53\xa1\x05\xfe\xba\x58\xb6\x04\xbd\x0d\x21\x91\x0c\xc4\x5c\x10\x43\x21\x3e\xd1\xa5\x57\x11\xc2\x0c\x83\x95\x00\x7e\xbe\x4c\x58\xdd\xcc\xb1\x9a\x29\x78\x46\xf1\xad\x8f\x6d\x89\x1d\xc0\x9c\x33\x0e\xcc\xd6\x32\xe9\x3c\x45\xdf\x0e\x14\x07\xcd\x5c\x15\x7c\x8b\x3c\xdf\x4d\x83\x1f\xfd\x6d\x5a\xd6\x81\xf9\xb1\xf8\x5b\x79\x43\x11\xad\x8d\xa4\x6a\xb3\x4e\xb6\x8c\xd0\x00\x1b\x2b\x19\x60\x01\xb7\xb1\x58\x07\x11\xd8\x53\x65\x31\x6e\xa6\x58\xa9\x6b\xf0\xd0\x2d\xf1\x02\x0f\x42\x85\x0b\x36\xf2\x91\x4d\xe4\x0c\xa6\x48\xc0\x08\x63\x0a\x1c\x23\x7b\x1a\x43\x2a\xb0\x1d\x70\x22\x67\x09\xd3\x2d\x78\x8b\x11\xc7\x7c\x00\xff\xfc\x57\x85\xf8\x9a\x8f\x1b\xff\x26\xce\x7f\xe6\xca\xb0\x56\xad\x6f\x67\x87\xce\x2a\xca\xaf\x66\xee\x18\x5f\x05\x58\xc8\xc5\x87\x3a\x5d\xeb\x3d\x96\xc7\x61\x8f\xee\x3a\xfc\x69\x72\x39\x39\x98\xdb\xe6\x37\x22\xa7\xef\x10\x71\xb1\xb3\xc7\xb1\xc6\xc6\xcc\xc5\x87\xe0\xa5\x86\x6e\xe5\x54\x37\x86\x94\x1b\x02\x30\x66\x01\x75\x94\x5f\x71\xb8\x9f\x0c\x76\xdf\xea\x3c\xcd\x60\x2f\x39\xa1\xfb\x56\xe7\xae\x28\x26\x55\x2b\x81\xda\x0d\xe4\x14\x24\xbb\xc4\x54\x39\x25\x84\x5e\x23\x97\x38\x69\x90\x7a\xcf\x04\xa4\xde\xdd\x41\xea\xcd\x03\xe9\x4c\x60\x0e\x94\x49\x40\x81\x9c\x32\x4e\x7e\x18\x27\x14\xd9\x36\x16\xa1\x4e\x34\x6a\x2e\x0d\x5c\xff\x99\x00\xd7\xbf\x3b\x70\xfd\x79\xc0\x1d\xb1\xdc\x4c\xbc\x21\x72\x0a\xc2\xc7\x36\x19\x13\xec\xc0\xe1\x3e\xe0\x5b\x22\xa4\xa8\x36\xcc\xab\x0a\xdc\x83\xda\xd9\x02\x70\xf3\x3c\x90\xb9\xe6\x12\xca\xac\x37\xca\x8d\x46\xa2\x11\x1d\xec\x62\x89\x4b\x6d\xa7\xf9\x29\x6f\x3e\x7d\xc4\x91\x87\x65\xb8\x36\x89\x58\x20\x74\x00\x57\x01\xe6\xb3\x54\xbf\x28\xf2\xf0\x00\x90\x98\x51\xbb\xaa\xb7\x5f\x30\x1f\x33\xee\xe9\x99\x84\xf4\x52\x05\x08\x55\xcb\x49\x5d\x6b\xca\x19\x65\x81\x50\x6b\x24\xaa\xd7\x1c\x75\xa3\x2c\x67\x3e\x1e\xc0\x88\x31\x17\x23\x9a\xfa\x45\x75\x99\x70\xec\x0c\x40\xf2\x00\xd7\xfa\x00\xdd\xd5\x93\xbf\x3c\xa5\x57\x47\x0c\xf6\x0c\x63\x55\x98\xee\xeb\x61\xcb\xa8\xf2\xe7\x31\xb1\xfa\x96\xa5\x79\x27\x8c\xde\x5d\x33\xe5\x49\x54\x2f\xaa\x94\xbd\xd3\xfd\x35\x13\x4d\x14\x7d\xfd\x17\x4f\xe1\xc5\x53\x78\xf1\x14\x94\xa7\x60\x74\xca\x3d\xfc\x85\x0c\x81\xbf\xa8\xd7\x70\x3f\x10\xf3\x04\xee\xee\x41\x44\xce\x81\x21\x57\xe7\x1c\x2c\xe4\x6e\xf8\x48\xda\xd3\x41\x9e\xf8\x99\xef\x20\x89\x63\xda\x51\x64\x53\x11\x27\x4b\xf9\x32\x19\x97\x24\xd0\x54\x8b\x2b\x7a\xcd\xf8\x5b\xe6\xa4\x48\x65\x31\x31\xdc\xb0\x1b\x8a\x39\xb0\x31\xe8\xf8\xc1\x5a\x8d\xcc\xd4\x4b\x4c\xb9\xbc\xcc\x5d\xe7\x1b\x2e\x0a\xab\xfd\x25\x3c\x94\x9a\xd0\x95\xc1\xd9\x00\x94\x5f\xf2\x3e\xab\x80\xc6\x17\x26\x1e\x37\xa2\x51\x70\x88\x32\x38\xbe\x45\x4e\x24\x50\x8f\x8c\xdf\x25\x12\xad\xb1\x8b\xb1\x6c\x99\x4d\x00\xae\xf7\x23\x16\x51\x36\x05\xc7\x64\x09\x7b\xbd\x82\x7d\xe9\xd5\x04\x64\x85\xd9\x34\x49\x59\x56\x31\xd7\xb2\xae\x60\x17\xfb\xd5\x5d\x8c\x8d\x9e\x09\x56\x69\x93\xa7\x7b\x95\x32\x7b\xab\xdb\xb5\x95\x0c\xa7\xdf\xad\x3f\xc5\x95\x6d\x2b\x24\xd8\xc8\x54\x4d\xca\x6d\x10\xa7\x51\x1b\xa1\x36\x84\x7c\x26\xca\xa3\xd3\x36\xc7\x91\x39\xfb\x73\xad\xae\xe7\xd9\x63\x23\xf0\xa9\xbd\xac\x9f\x67\x84\x23\x2b\x83\x66\x2e\x43\x4e\xd6\x44\x55\x19\xa8\xb3\x93\x63\x3c\x21\x45\x31\x9c\x63\x85\xa2\x6a\x15\x31\xf5\x83\xb3\x3b\x51\x8d\xaa\x15\xa8\xae\x7e\xa4\xe3\x19\x38\x07\x79\x5b\x6a\xdb\xd8\x7f\xae\xd1\x94\x68\xe7\xe4\x1e\xd1\x94\x1c\x89\x97\x68\xca\x4b\x34\x25\x02\xe9\x81\xa3\x29\x31\xd9\x4f\xe8\x76\xd7\x75\xd9\x0d\x76\x0e\xc3\x45\xe3\xb1\xd9\x30\xbe\x47\x7b\xf3\x68\x96\x32\x72\x8a\xb9\x27\x8e\x98\x8c\x74\xc0\x3d\xda\xaf\x20\x55\x1f\x4d\x1a\x33\x3e\x22\x8e\x83\x29\x60\xa2\xb7\xd6\x47\xd8\x46\x81\xc0\xda\x9e\x07\x45\xc7\xb8\x32\xe4\x04\x2c\x5b\x37\xda\xa2\xa7\x81\x37\x32\xeb\xe1\x38\xf1\x08\xe4\x14\x49\xb0\x11\x85\x11\x0e\xdd\x13\xbd\x98\xd4\x99\x5e\xba\xcd\xfc\x36\x7e\xbb\xda\xcd\x5d\x5d\xd9\x7d\xcc\xad\xaf\xd3\x29\x8e\x3c\x20\xec\xc4\x99\x12\xe0\x30\x2c\x68\x53\x9a\x00\x56\x1a\xb3\xd7\xcf\x04\xb3\xd7\x47\xc8\xc3\x7b\x8c\x8e\x5d\x62\xcb\xbb\xe3\x57\x46\xa6\x5a\x59\x2a\x3c\x74\xc9\x44\xee\x1c\x2c\xcd\x92\x82\x50\x2d\xcd\x76\x68\xa2\x94\x1c\x6b\x31\x8d\x20\xaf\x5e\xa4\xac\x2a\xc8\x8f\xbb\xb5\xb8\x4b\x21\xa8\x5a\x90\xc1\xcd\x94\xb8\x11\x96\x74\xa2\x81\xcd\x44\x05\x43\xa2\xcb\x6d\x3f\x6a\xef\xa1\x18\x61\xd4\xc5\x52\x19\x3b\x25\xdb\x95\x2e\x11\x52\x0d\x68\xa6\x9e\x28\x5b\x43\x45\x19\x3e\x62\x19\x0e\x97\x8e\xae\xed\xd6\x73\xf4\x53\x85\x2a\xed\xbf\x7e\x24\x69\x4f\x7a\x55\x22\x5b\x0f\x31\x17\x0e\x8d\x67\xf4\x55\x2d\x7b\xef\xe1\xc0\x96\x90\x59\xe5\x38\xda\x0a\xfa\xaf\x7f\x5d\x97\xf4\xbe\x1b\x7c\x2b\x19\x1b\x5b\x39\x3b\x55\x1e\xf2\x2a\x23\x92\x0a\xbc\xf9\x68\x92\x1a\xaa\xb9\xc5\x05\xf9\xb1\x4c\x71\xc6\x1d\xcc\xdf\xce\x96\x69\x00\x23\x6e\x4f\x9b\x15\xc1\x40\xdb\x65\x81\x33\xf4\x39\xbb\x26\x4e\xdc\xcf\x3a\x03\xa8\xc4\x29\x32\x38\x22\xf0\x7d\xc6\x95\x9c\x68\x32\x10\x93\xa9\xb0\x86\x7b\xaa\xd4\x97\x5c\xa1\xbb\x5a\xc5\x66\xd7\xb2\x9a\x95\x32\x6c\xd8\xc5\xce\xc2\xbc\xc2\xcf\x14\xea\x0c\x10\x59\x43\xd9\xec\x5b\x9d\xea\x6e\xbd\x28\x7e\x03\xd2\x66\xdd\xd8\xbf\xe8\xaf\x27\xd0\x5f\x0b\x28\x17\x9d\x14\xbf\xc1\x75\x88\xf8\xce\x9a\x26\xac\x6e\x96\x54\xb8\x72\x5a\x2f\xa2\x81\x4c\xb0\x7a\x45\xf4\x50\xd4\xb1\x27\x53\x47\x06\x8d\x17\x65\xf4\xa2\x8c\xe2\xbf\x9f\xa6\x8c\xe6\xec\x62\x66\x0b\xff\x0c\xcd\x15\x3a\xcc\x43\x64\xdb\x2c\xa0\xb2\xa8\xad\x16\x51\x04\x3f\xfb\xa8\xd9\xae\x61\x36\x3b\x85\xe7\xea\xa3\x30\x10\x1b\xf5\xb4\x7a\xee\xaf\xaa\x94\x3e\xe5\xa6\x48\xb3\x6f\xf5\x9e\x09\x48\xab\xb5\x02\x2d\x28\xcd\x55\x05\xee\x19\x9c\xa9\x90\x68\x92\xd1\xa9\x51\xa5\x0a\x1f\x28\xab\x2d\x44\x95\xfb\x85\xea\x55\x44\x3a\x73\x64\x7e\x56\xc5\x49\x96\x44\x21\xd8\xf7\x13\x52\x2c\xb2\xdd\x2e\xdd\xea\xaf\x92\x02\xb1\xa0\x88\xc5\x23\x5f\xda\xd6\x9d\xb3\x22\x56\xc5\xb0\x2c\x3e\x69\x42\x89\x09\x47\x7b\xe9\x89\x93\x6d\x76\xde\x1c\xca\xcb\x56\xb8\x37\xf8\x62\xc8\x5e\x0c\xd9\xd2\x86\xec\xe3\x5c\xaf\xe8\xc5\x6e\x3d\x98\xdd\x2a\x49\x38\xcc\xce\xfc\xc5\xec\x5b\xc9\x9e\x5e\x6e\xf8\xd6\x00\x9a\x8b\x78\xf9\x3a\x40\xd1\x2c\xf8\xfa\xf7\x5c\xb9\xfc\x39\x34\x7a\x49\x3b\x4b\x69\xf1\xb7\xb3\xc3\xb9\xb9\x25\x89\xeb\x91\x1b\xc0\x65\x0f\x7e\xcc\x73\x7a\x52\x27\x34\x16\x15\xae\x78\xe1\x54\xcd\x5a\x5c\xf6\x3d\x96\x65\xc5\x42\x75\x9b\xe9\xb2\x2a\x9a\xce\x87\x89\x8a\xc7\xb9\xd6\x13\x72\xad\x34\x76\x54\x35\x7d\x24\xf6\x51\xe4\xb2\xbf\x22\xda\xad\xf6\xe0\xe8\x8b\x49\xff\x73\x99\xf4\x7b\x80\xb4\xea\x6b\x53\xf8\xf7\x7f\xfe\xb4\x36\xdb\xa8\xa3\x7b\xab\xd6\xe4\xb4\x5f\x95\x6e\x5d\xdc\x7a\x6f\x70\x2c\xb0\x1c\xda\x1c\x3b\x98\x4a\x82\x5c\x11\x8e\x64\x7a\xbd\xfa\x62\xd0\x95\x41\x6f\x69\xa8\x1e\x79\x71\x76\xac\xda\x80\xd4\x70\xbc\xe8\xf0\x17\x1d\xfe\xa2\xc3\x57\x47\x87\x6b\x25\x90\x9d\xd3\x7b\x1c\x3b\x62\x69\xf7\x58\x60\x29\xa2\xac\xd7\x68\xb2\xc3\x98\xf1\x1a\xb5\xfe\x4a\xfd\x1f\x4e\xa7\x58\x60\x40\x3c\xc9\x1e\x6f\x8d\x91\x4d\xe8\x04\x38\x76\x75\x96\x77\x7c\x9f\x66\x58\x67\xce\xbd\x6b\x1b\x1e\x96\x9c\xd8\x62\x43\x1f\x4c\x1b\x72\x44\x27\xb8\xb0\xa8\x2b\xc4\x3b\xc3\x4a\xa1\xe7\x4d\x3c\x2c\x30\x27\x58\x80\xae\x6e\xce\xb8\x29\xc6\x4d\x92\x65\xbc\x18\xc9\xaf\x33\x3e\x19\x2a\x6f\x67\xc7\xaa\xda\xd7\xd4\xc9\xb8\x47\xde\x5f\xfe\xf5\xe4\xf3\x11\x20\xce\xd1\x0c\xd8\x18\xbe\x70\xe6\x61\x39\xc5\x41\xd2\x2f\x36\xba\xc0\xb6\x14\x30\xe6\xcc\x03\x36\x52\x63\x82\x24\xe3\x24\xf0\x9e\x42\xbd\x84\x38\x25\x28\xbd\x6c\x3c\xbf\x6c\x3c\xc7\x7f\xab\xb9\xf1\x5c\x59\xd8\x09\x8c\x0e\x58\xa2\x0a\xa1\x52\x4d\x40\x77\x89\x2a\x63\xe2\xaa\xff\xd6\x1f\xeb\x2d\x51\x80\x4b\xaa\x3e\x73\xe0\x46\x2e\xaf\xf1\xcc\xf9\x25\xf9\xa2\xf3\xe6\xe8\xbc\x34\x4e\x2f\x5a\xef\x45\xeb\xc5\x7f\xcf\x4c\xeb\xc5\xfa\x68\x6d\x2d\x29\xa0\x1a\x0b\xbb\x6f\xda\xfd\xac\xa7\xe0\x31\x1e\x63\x8e\xa9\x1d\xf7\xcc\x9c\xda\x37\xf3\x33\xe2\x98\x2b\xd5\x22\x49\x1a\x1a\xe2\xa4\xa1\x30\x95\x84\xe4\x84\x4e\xe2\xaf\x2f\x09\x9d\x5f\x68\xaa\xba\x52\x57\x48\x4d\xc4\x41\xac\x90\xc2\x7d\xd9\x14\x16\xaa\x95\xd4\x47\x1f\x4d\x70\xda\x2b\x26\x3f\xd2\x1f\x25\x93\xc8\x4d\x7d\x26\x12\x7b\x62\xb9\x8e\x2f\xd4\x2b\xc5\x45\xb1\x90\x32\x2d\x93\xd4\xb5\x09\x8a\xb9\xf9\xa5\x34\xcf\xf5\xc5\xb4\x38\x47\x45\x90\xeb\x7e\x1e\xcf\x13\xad\x68\x22\xe4\x84\x20\x2d\x64\x25\x78\x54\x61\x02\x7a\xee\x3a\x85\xd9\x51\x8a\x0d\xe8\x81\x44\x25\x33\xb9\xb2\x78\x6c\xd9\x86\x59\xb1\x2b\xad\xa4\xc1\x48\x4b\xcd\x52\x80\xa8\x8a\xf7\x40\x41\x0b\x54\x39\x8b\xda\x1c\xe6\x7e\x29\x2d\x0e\xb5\x0c\xea\xee\x19\x0e\xd3\x87\xba\x9e\x78\xf4\xd3\x77\xc3\x27\x7f\x19\x03\xd0\xf8\x0d\xb9\x01\x16\x03\xf8\x27\x0a\x4f\x34\xaf\x83\xcf\xb1\x8f\xd4\xc8\xad\x9b\x74\x5b\x41\x18\xd5\x9f\x38\x46\xce\x6c\x1d\xc6\xfa\x36\xe7\x75\x70\x70\xfc\xf3\xba\x09\x6e\x12\x3a\xf9\x17\x34\x16\x15\xa0\x6c\xbe\x73\x3d\x9b\x47\xc8\xc3\xca\x6b\xd1\xb9\xb7\x6a\xd5\xab\x23\x1a\x0e\xf6\x5d\x36\x6b\xc3\x3b\xc6\x23\x43\x01\xbb\xdf\x4e\x16\xe6\xc0\x0b\x5c\x49\x86\xe8\x47\xb9\x6c\x14\x2f\x49\x81\x30\xed\x78\x11\x48\xe3\x97\x31\x52\x29\xd8\xe1\x4d\x40\x76\x2e\x99\x39\xd3\x81\x01\x04\xa2\x85\x91\x90\xad\x8e\x8e\x07\x2c\xd3\x1f\x7d\x2d\xda\xc2\x13\x58\x5f\x3c\xb3\x68\xe1\x11\x63\x52\x48\x8e\xfc\xa1\x79\x38\x62\x38\x4d\x45\x88\xe7\xd6\x0e\x93\x4c\x86\xa8\x50\x65\xcc\xb8\x87\xe4\x00\x1c\x24\x71\x4b\x12\x0f\x2f\x4a\x32\xbc\x20\xed\x21\x49\x1a\xc1\x1e\x2e\xa9\x07\xa3\x37\x44\x16\x2b\x5f\x7a\x1d\x6c\x99\x3a\xa8\xbb\xe5\xa4\xa8\x69\x1e\x5b\xb5\x96\xb2\xad\xad\x2e\x34\xf2\x7c\x64\xc5\x55\x5b\x5d\x68\x74\xb2\xdf\x6a\x2b\x5b\xf8\xd6\x58\xd5\xc2\xd7\x4a\x21\xe7\xe1\xbd\xcf\xc5\x30\x8f\x6b\x27\x72\xf0\x27\x7f\xf5\x03\x91\xe6\xd9\x74\x3f\xf7\xaa\xc8\x4f\x32\x26\x75\x23\xbd\xfb\xe5\x30\x64\x2a\x37\x40\xea\xc7\xeb\xdc\xa8\x4d\x0d\x5b\x25\x8b\xef\x46\xce\x47\x71\x5d\xac\x2f\xb5\x2a\x80\xd9\x32\x94\x4d\xed\xbc\x16\xac\x6b\x61\xa3\xaa\x4a\x5a\x64\xf3\xb2\x5a\xed\x44\x55\x32\xf8\xb3\x84\xa3\x74\x18\x33\x4f\xa6\x44\x34\xb3\xb9\x7e\xba\xba\x36\x23\xe9\xac\x89\xf0\xf9\x8f\x28\x92\x00\x23\xe6\x44\xec\x17\x46\x3f\x7d\xfd\x99\xf9\xf3\xd0\xed\x30\x7a\x0b\x64\x18\xde\x1d\x92\xc9\xc9\x5c\xd4\x67\x2f\x50\x2e\xdc\xbe\x11\x1d\xd1\x0f\x6f\xde\x40\x3e\x09\x79\x2f\x38\xde\x05\x31\x2e\xae\x4a\x0c\xca\x65\xdc\x2f\x20\x03\xa5\x9d\xae\xf3\x02\x0e\xa9\xa3\xd6\xc7\x38\x79\x4f\xe5\x06\xc3\x14\x5d\xe3\xe8\xba\x95\xb0\x73\xd1\x15\x2e\x11\xf1\xb9\x9e\x48\xc9\xdd\x67\x8b\x8c\x7d\x7c\x27\x2c\x73\x66\x20\x30\x95\xca\x7f\x0a\xa7\x09\x7c\xf9\x7c\x72\x5a\xb3\x9a\x53\x8e\xc2\x72\x63\x5b\xed\xda\x15\xc6\x38\x77\xae\xeb\x46\xbf\x14\x96\xdc\x5a\x61\xbb\x81\x90\xea\xfb\xd0\x9b\x8a\xae\xb2\x21\x74\xde\x72\xaf\xcc\xb9\xcb\x65\xc2\x4a\x73\xcf\x88\x64\x5a\x7c\xd5\x7f\x6d\x46\xc7\x64\x12\x94\xb2\x20\x99\x62\x40\x93\xdd\xfd\xa3\xd0\x7a\xde\x5b\xcc\x7b\x57\x99\xa6\x9b\xaa\xe7\x34\xf4\x69\x0b\x2d\xb5\xe1\x50\x82\x17\x08\xa9\xd8\x11\x61\x8e\xa5\xcb\x6e\x30\x6f\xd9\x48\x60\x40\xae\x3f\x45\x34\xf0\x30\x57\xae\xe4\x14\x71\x64\x4b\xcc\x05\x30\x0e\xcd\x66\xab\xd9\x5c\x57\xb3\x84\x87\x59\x51\x88\x9a\xf2\x23\x2c\xd3\xa5\xd7\x01\x51\xbd\x55\x94\x2d\x55\xa0\x6a\xca\xd9\x88\xea\x0d\xc5\x11\x06\x97\xd1\x89\x02\x63\x8a\x28\xf4\xba\xa9\xe6\xdb\xcd\x79\x23\x52\x74\x9e\x4b\xee\xdb\x51\x45\x1e\x48\x0a\x0a\xe7\x86\x9f\xca\x43\x2a\x30\xf2\xf4\x2e\x52\x86\xa5\xe7\xe2\x23\x65\x98\x6e\x24\x63\x9c\x1c\xc6\x7c\xd2\x11\x4e\xd8\x58\x91\xf1\xad\xbc\xb5\x72\x75\x47\xd7\xb0\xdc\x28\xce\xdf\x52\x23\xd7\xdc\xcb\x2e\xa4\x9b\x75\xfe\x46\x2e\x46\x98\x25\x94\xd8\x6b\xa5\x75\x54\x97\xe3\x6b\xa8\x8c\x20\xb4\xe1\x5b\xa8\x7c\x9a\xcd\x0c\x63\xcd\x26\xb8\x84\x5e\xce\x57\x7f\xa4\xa6\xf9\x33\x4a\xae\x02\x0c\x44\xa7\x02\x8c\x89\xb9\xcc\x4d\x71\x12\x36\x3e\x97\xb8\x43\x84\xef\xa2\xd9\xb0\xde\xec\x1c\xa5\x4c\x4e\xce\xf0\x2a\x47\x21\x24\x02\x7e\xc0\x7d\x26\xf0\x02\x2a\xbd\xbe\xb9\x5f\x02\x0f\x51\x18\x73\x82\xa9\xe3\xce\x4a\x7a\x97\xe5\x61\x5d\x33\x11\x05\x72\xce\xd1\x8d\x38\x9f\xcf\x01\xa6\x68\xe4\xe2\x1a\x68\xbf\x85\xfe\x57\x49\x9f\x89\x88\xaa\x9b\xee\xeb\x70\x12\xa1\x13\x65\x0e\x3f\x9f\xec\xc7\xf6\xb8\xc8\x44\xd6\xda\x97\x39\x4d\x21\xe1\xbc\x92\x2a\x17\xe3\xfd\xe4\x93\x82\x06\x45\x76\x50\xff\xdb\x7e\x3a\x19\x37\x3c\x37\x9b\xcf\x4e\xb8\x43\xfc\xca\x84\x3a\x27\x65\x47\x6d\xf8\x8d\xf0\x09\xa1\x04\x3d\xb4\xb4\x85\x4c\x3c\x94\x94\x99\xc6\xc6\x28\x70\xe5\x00\xc6\xc8\x15\x78\x41\xf1\xcb\xe6\x2b\x95\x4b\x60\x74\x2c\x6c\x37\x7b\x2c\x0c\x08\x85\x4f\xbb\x27\xad\x93\x93\xcf\xf1\x52\xc2\xb8\x64\x7b\xa1\x4b\xa6\x73\x96\x02\x39\x55\x63\x6b\x36\x04\x9b\x3f\x27\x50\x51\x69\xc3\xea\x17\x66\x4d\x13\xbf\x84\x09\xa6\x98\xeb\x2e\x06\x91\x78\xc6\xb7\x64\x64\x13\xb0\xf2\x5b\x92\x4b\x05\x54\xb3\x6d\x2f\x4c\x2a\x5d\xed\x61\x28\xda\x2e\xc1\x54\x2e\xb2\x57\x93\xab\x21\xb0\xcd\x8b\x89\xaf\x0f\x14\x83\x7e\xf0\xc0\xf0\xf2\xf1\xd5\xd2\x0c\xdd\x46\xc9\xc4\xc9\x6d\xea\xe4\xe6\x4f\xf9\x72\x5b\x2d\x29\x75\x17\x8b\x79\x7d\xcd\x07\x5d\x71\x2f\xb7\xdc\xac\x91\xf0\x72\x05\x5c\x2e\x8e\xd9\x46\x76\xd3\x9f\x0b\x41\xa7\xc5\x9a\x2a\x0c\xdf\x12\x43\x57\x16\x23\x2f\xde\xf5\x70\x98\xf2\xa4\xcb\xa7\xd2\x5f\x4e\x6f\x55\xaa\x86\x2c\x03\xa6\xd8\x4f\xd1\x93\x0b\xca\x6a\x6d\x2b\xa5\x8a\x28\xdb\x4c\xfc\x8c\xd0\xbd\xd0\xbb\xb3\x0a\x2b\x8e\x6f\xe1\x60\xb6\x9a\x49\x3a\xbb\x56\x22\xcf\x7f\x08\xeb\x51\x59\x27\xcf\x8e\x93\x75\x49\x2b\x11\x2a\xce\xb0\xca\x35\xf8\x1d\xd6\xd5\x45\xea\xc5\x75\x71\xc9\xb6\xc1\x12\x47\x32\x22\x9d\xb0\xc4\x2a\x39\xef\x65\xd7\xe2\xfa\xa4\x4b\xea\xf2\xae\xa6\x21\xac\x8a\x93\x67\x32\x78\x20\x97\x97\xf3\x2a\x93\x77\x1e\xa5\x0f\x46\xf9\xe7\xaf\x74\x99\xd2\x94\xe5\x87\x14\x8d\xd2\x06\x4a\xf6\xa5\x3a\x74\xe4\x9f\x6c\x5b\xbf\x38\xc1\x17\xdc\x77\x2d\xc9\x76\x2e\x4e\x26\xdd\xbd\x8f\x3f\xc6\xc1\x02\xb2\x54\x2b\x49\x05\x16\x1e\x4d\x88\x9e\x89\xbc\x25\x48\x18\x68\x93\xcf\x83\xe5\xdc\x19\x23\x53\x83\x82\xb3\x50\x90\x10\xe4\x38\x44\xe9\x28\xe4\x7e\xa9\x00\xba\x14\xa9\x6b\x93\xf6\x52\xa0\x9f\x87\xa8\x04\x9e\xba\x14\x46\x43\xd6\x0c\x7f\xb6\x89\x05\xfb\x1d\xeb\xfa\x22\x6b\xf9\x14\xb4\xc4\xbe\x10\x2a\xb7\xfa\xd9\xae\x15\xab\x9b\x37\x02\x4a\x6a\x3b\x2c\x18\xb9\x75\x4b\x49\x4d\x30\x3d\xa7\xf3\x29\xb9\x8f\x30\xab\xf3\x4d\x3c\xc9\xbc\x4e\x33\xf1\x57\x9f\xd9\x69\x2c\x0c\xbc\xe9\x6f\x56\x6c\x76\xaf\xf6\x2c\x2a\x3e\x21\x39\xa8\xa9\xa7\x7d\xd4\xe5\xe0\x7d\xa5\x17\x5f\x94\xdd\x84\xef\x64\x12\x13\xf0\x63\xd4\x9d\xa5\x2e\xd4\x1c\x13\xec\x9a\x88\x94\x49\xa5\x8a\xab\x17\x1c\xe7\x0c\xc6\x6b\xc5\x64\xe9\x44\x86\xcd\x5b\x66\x71\x9e\x7d\x61\x7b\xef\x70\x5f\xf9\x92\x1c\xdb\x8c\xc7\x67\xf0\x72\x99\xe2\x25\x03\x4a\xe8\x00\x7c\x24\xa7\x79\x94\x92\xa4\x83\xe8\x48\x49\x96\x8f\xe8\xdb\x14\x99\xf4\xe3\x6b\x19\xee\x34\x7b\x2e\xa6\x13\x39\xd5\xee\x2e\xf1\x30\x10\x0a\x1e\xa1\x81\xc4\xe6\x64\xde\xcd\x94\xd8\x53\xb5\xa2\xe7\xfa\x44\x85\x79\x81\xc5\x68\x8a\x6a\xce\xaa\x3a\x98\x97\xc3\x72\x29\x8c\xc3\x7d\x9b\xc9\xdc\x21\x94\x78\x81\x37\x80\x4e\xf2\x95\xc9\x21\x18\x40\xbf\xd7\xb5\xc2\x6f\x8b\x79\xf3\x79\x8c\x20\x96\xf2\x90\x7a\x74\xc8\x26\x37\x98\xe1\xb7\x8b\x80\xa8\x30\x8c\xca\x2b\xf4\x04\xb6\x19\x75\x04\x8c\xb0\xbc\xd1\x2f\x7e\x20\x89\x20\x3e\x9b\xf8\xb8\x88\xf5\xac\x85\x20\xeb\x58\x3b\x56\x35\x66\x79\x48\x52\x98\x85\xf4\xc3\xb4\xfe\x2c\x66\xe1\x97\x8b\x40\x16\x5d\xa1\x14\x39\xd1\x92\xc1\x18\x4b\x7b\xda\x86\x77\xea\x3f\x4a\x5d\x27\x07\x8e\xa6\x98\x02\xf6\x7c\x39\x6b\x9b\x7a\x98\x4a\x7d\xe8\x12\x71\x1c\x3f\xcf\x2b\x31\xa7\x28\xaa\xa3\xf9\x11\xed\x5a\x5c\xb3\xb6\xa3\x60\x35\x2a\xe2\x41\x21\xca\xf1\xd3\xc1\x49\x6a\xbd\x81\x20\x95\xf2\x5f\xdb\xff\x2f\x68\xa2\x64\xc6\xc1\xb7\x05\x89\x48\x87\xb8\x17\xd0\x12\xc5\xd1\xcb\x27\xfc\x87\x23\x17\xed\xad\xa6\x33\xfd\x0d\xd3\xa9\x83\x09\xb5\x4c\x1f\x25\xef\x2d\x29\xb8\x94\xa8\x63\x64\x4f\xd3\x9d\x7e\xc0\x6e\xe4\x4f\x24\xc4\xdd\xb0\x2c\xd3\x91\xf0\x96\xfb\xd2\xa8\xd6\xff\xb4\xe2\x9a\x27\xe1\x3b\xa4\xa1\x35\x50\x95\x60\x34\x03\x9b\x13\x89\x39\x41\x6d\x3d\x7d\xc5\x8c\x4a\x74\x1b\xef\x0b\xc5\xaa\x1e\x88\x48\x31\xe4\x11\x17\xe9\x84\x18\x99\xab\x82\xe1\x3c\x22\x7c\x0e\xb6\xab\x5f\xaa\x62\x63\x40\x14\x4e\xbe\x7e\xd4\xc9\x5b\xd8\xc3\x54\x26\x6f\x4d\x1d\x28\xdc\xcc\xe1\xb9\xf0\xb1\x2a\x5d\xdf\x3c\x56\x85\xe8\x2c\x26\x7b\x99\xca\x7b\x12\xe7\xc6\x7a\x89\x84\x4e\x2a\x83\x7a\x3d\xa5\xa3\xd5\xf4\xc9\x3e\x5c\x2d\xd2\x0d\xc8\x29\x26\x5c\x0f\xfe\x3a\x44\x8f\x6a\x8d\x99\xeb\xb2\x1b\xb5\xb0\x35\x1d\x1b\xac\xc5\x8d\x9c\x9f\x9f\x8b\xab\xe4\xa8\x8a\x8e\x52\x21\x61\xa7\x7f\x4f\x0a\x9f\x2e\xcf\x04\x0c\x11\x75\x86\xf1\xde\x8b\xb2\xca\xf7\xe1\x6b\x3d\x15\x9b\xaa\xe6\xf3\xd0\x00\x9c\x1e\x69\xda\x94\xd1\x7e\xa3\xb3\x0e\x8c\x03\x31\x65\xb4\xe4\xe9\xfd\x2c\xa5\x84\xd6\xd5\x77\xc9\xe6\x94\x09\x7b\x8b\xc0\x95\x46\x21\xa5\x7a\xa8\x18\x6a\xc7\xf2\xed\xbb\xcc\xc1\x19\x85\x5f\x94\xf9\x9c\x48\xa7\xc5\x3e\xea\x5d\xa3\x62\xa6\x9a\xa9\x1c\x12\xb8\xef\x6c\x14\x72\xe6\x2a\x85\xce\xb8\x39\x03\x69\x9e\x87\x28\x9f\x69\xc9\x44\xd3\x85\x92\x89\x95\x92\x89\xfa\x19\x36\x67\x66\xe9\xac\xa8\xec\xb4\x4a\xda\xcc\x4c\x2f\x08\x1f\xc9\x0b\xa7\x49\xf4\xc2\x96\xe1\x5e\x8f\xce\x79\x36\x45\xef\x7c\x1d\xce\x15\x70\xea\xbf\xda\xff\x53\xff\x30\xfb\x97\xe7\x26\x05\xec\xdc\x64\x5e\x9e\x27\xb4\xd5\x2a\x01\x71\x24\x19\x37\x03\x7e\xfe\x5f\xff\xad\x6a\xbd\x39\xd7\x22\x73\xfe\xf1\xf0\xc3\xc1\x79\x32\x41\xa3\x5a\x17\x8c\xd0\xb0\xfc\xee\xd1\xfe\xb9\xa1\xfd\xf9\xf8\xbc\x0d\xbf\xb0\x1b\x7c\x8d\xf9\x3a\xcc\x58\xa0\x95\x81\xea\x65\x92\x21\xc9\xc6\xd0\xb1\xc2\xea\x84\xea\xfd\x14\xdd\x1b\x3d\xf6\x29\x8c\x0f\x62\x61\x2a\x9b\x8a\xf9\x67\xec\xe3\x9b\xc3\xb4\x58\x9d\x7b\xb3\x96\x56\x34\x86\xaf\xd4\x26\xae\xce\x46\x58\x74\x32\x66\x67\xe2\x1b\x88\xa8\x9a\x5c\xba\x0c\xf0\xf0\x06\xd0\x8d\x48\x57\xfe\xa7\xdf\xfa\xd7\xe2\xac\x23\xd3\x86\x7e\xed\x4f\x67\xfd\x85\xe7\x92\xcf\xbd\xd9\x1d\xd9\x75\xc9\x25\x06\x6f\xf6\xb7\xee\xe6\xa3\xe8\x0b\xad\x0d\x93\x6d\xe4\x58\x23\xa6\xf4\x08\x92\xc9\xcb\x88\x53\x24\xc0\xc7\xdc\x23\x42\xe8\x58\x38\x03\x81\xcd\xf5\x17\x3c\x3c\xd7\x9c\x1a\xfa\x23\x26\x71\x3b\x62\xd0\x18\x95\xe4\xec\xaf\x12\xe3\xf0\x94\xab\x7e\x5b\x2e\xaa\x5d\xad\x96\x42\xa7\x40\x8b\x59\x85\xb2\x29\x57\x2c\x25\x36\x3c\xa3\x37\x20\xaf\xce\x16\x10\x91\xc6\x5d\x95\x56\x74\xaa\x5c\xef\x50\x46\x3c\x85\xc7\xca\xd3\x34\xf1\x00\x46\xfa\xdb\xf0\x4b\xf3\xe1\x5d\xe8\x65\xff\xfa\xed\x34\xb3\x08\x9d\x4a\xe9\xaf\xe5\x7b\x9a\x7b\x27\x39\x22\x9f\x5b\x4b\x87\x69\xa0\xd0\x88\x0f\x3a\x25\xf1\x9a\x5c\xe2\x30\x34\x52\x3d\x8f\x06\xa4\x11\xdd\x97\xeb\x13\x19\x1f\x96\xc8\x3d\xa6\x3c\xaf\x69\x1c\xb4\x6e\xf0\x03\x35\x5d\x72\xda\xa4\xa2\x79\x13\xb7\x22\x27\xdf\xb7\x8e\xbf\xf6\x7e\xfd\x70\xb8\xf3\xd5\xfa\x7c\xea\x5d\x7c\x7d\xe7\xf4\x98\xfd\xee\x78\x92\x34\x17\x46\xc3\xb4\x48\x24\xdf\xce\x3d\xf0\xb0\xb1\x10\xf1\xf0\x5c\x20\x34\xf4\x81\xbe\x45\x11\x88\xf3\xa9\xd3\x53\xa4\x7e\x34\x4d\x24\x01\xa0\x81\x7c\x32\xd4\x1c\x0e\x43\x00\x6b\x80\x4d\x7e\x2a\x3f\x73\x96\x2e\xdb\xea\x10\x31\xdb\xe2\x57\xbd\x8b\x4b\xb2\x73\x65\x31\xe9\x5d\x5c\x8d\x55\x7f\xc7\x7c\xd2\x46\xbe\x2f\xda\xde\x65\x6b\x24\xe5\xc4\xba\xa0\x9d\x6d\x6b\xea\xb7\x6f\x37\x83\x9d\xb6\xe8\xb4\x1d\x7c\x2d\xa6\x64\x2c\xdb\x8c\xa7\x90\x49\xed\xe5\xe9\xd7\xb5\xad\x56\xc7\x6a\x59\x9b\xa7\x9d\xee\x60\xb3\x33\xe8\xf6\xdb\xd6\x66\xaf\xd3\xef\xfe\x91\xd4\x48\x1d\x43\x2b\xd4\xd8\x1a\xf4\xb6\xda\xbd\xad\x6e\xd7\xda\x49\xd5\x88\xce\x8b\x41\xa3\xdb\xde\x6a\x5b\x45\x19\x7a\xa7\x8f\xa1\x45\x0f\x36\x9b\xe3\x1d\xcf\x4b\xae\xcc\x41\xba\x17\xc1\xfa\xb9\x82\x95\x3d\xbe\x08\x0d\x14\x9e\xe8\x2e\xbe\x01\x9a\xec\x2d\xa7\x47\xaa\x2c\x25\xbc\x42\xe0\xca\xf2\xda\x1b\x59\x71\x2c\xd3\x9e\x99\xef\x32\x49\x7d\xd0\xd8\xf5\xd0\x0f\x46\xe1\x1b\x1e\x45\x5b\xca\xa9\xb2\x51\xda\x5d\x22\x21\xc5\xfc\xe6\x05\x58\x4d\x27\x17\xc7\x8c\x96\x88\x57\x8e\xb5\xb3\x13\x38\x40\x42\xae\x43\x2a\x57\xb0\x8e\xb7\xba\xb4\x94\x0a\x2e\xc3\x96\xbc\x59\x0b\xf9\x7e\x4b\xa4\xc8\x67\xcf\x16\xe7\x93\x04\xc6\x8c\x83\x37\x03\xe4\xfb\x65\xf9\x2e\x8b\x68\x8d\x82\x6e\xc8\x92\x58\x48\x49\x64\x2f\xb0\x17\x1b\x9d\xc2\xc8\xdf\xbb\x63\x90\xc9\x53\x81\xc6\xc9\x6e\xab\xd3\x55\xff\x2b\xfc\x1c\xe6\xab\x29\x92\xea\x1f\x05\xa5\xd1\x50\x96\xbb\xa5\x9c\xcb\xea\xe9\xd9\x69\x59\xfd\x96\xb5\x7d\xda\xd9\x1a\x74\xfb\x03\xab\xf3\xff\xad\xcd\x41\xcf\x2a\x83\x38\x75\x51\xf3\x5f\x04\xe6\x47\x81\x31\x97\x85\x71\x1f\x28\x8b\x59\x0e\x7f\x15\x48\xab\xd2\x11\x2a\xd0\x2c\xee\xc2\x0d\xb5\xc6\x1b\x0e\x07\x90\xd8\x54\xcc\x87\x23\xce\x2e\x31\x97\xcc\x27\x76\x18\x8a\x1e\x8e\x66\x12\x8b\x21\xa1\xc3\xec\xe5\x2d\xa0\x17\x22\xde\x0f\x32\x24\x6c\x18\xe6\x35\x87\xc4\x5a\xf9\x3b\x27\x01\x34\xc5\x01\x0c\x87\x36\xa3\x22\xf0\x30\x1f\xb2\xf1\x58\xe0\xd4\xfb\x01\xc5\x6d\xfa\x56\x6a\x73\x0f\x3a\x5b\x9d\xce\xd6\xb6\xd5\xed\x59\x96\x65\xa5\x0a\xc5\xab\xab\x9d\x7e\x67\xb3\x3f\xaf\xf6\x56\x65\xed\xcd\x9d\x9d\x9d\x79\xb5\x5f\x57\xd6\xde\xde\xea\x76\xd3\xe3\x52\xb2\x9d\xfc\x7c\x47\x66\xee\x28\x14\x46\xa0\x6f\x59\xfa\xea\xe2\xb9\x16\xdb\xcc\x72\xab\x57\x98\xe7\xa9\x5b\x56\xa0\x7e\x5a\xeb\x65\xbe\xd8\xc8\x10\xd1\x77\xe1\x40\xe3\xc3\xee\xbb\x0f\xbb\x27\xad\x4f\xef\x3f\x9d\xb6\x32\xbf\xc7\x7e\xd3\xc9\x8c\xda\x53\xce\x28\x0b\x04\x20\x7d\xee\x1b\x88\xd0\xc7\x1f\xe3\xcd\x53\x13\x59\x41\x62\x46\xed\x37\xfa\xe8\x68\x1c\x0d\x49\x4d\xea\xf4\xfd\x38\xca\x3d\xff\x76\x48\xbc\xab\xf7\x36\xdf\x0f\x3e\x6e\x75\xd0\xd9\xed\xe1\x1f\x57\x6f\x4f\xaf\x8e\x8e\x43\xcd\xd2\xb7\xac\xc8\xe7\x7f\xc1\xa7\x1c\x9f\x9a\xa7\xc4\xcb\x30\xea\x3e\x04\x46\xdd\x7a\x88\xba\x65\x08\x99\x15\x1c\x48\xa6\xfa\x2d\x70\x26\x52\x39\x80\x33\xed\x30\xaa\x5f\xf5\x93\x51\xd9\x77\xe8\xcd\x79\x8a\xfc\xaa\x66\x00\xd9\x36\x07\x30\xaf\x89\xd4\xc3\x99\xcc\x0d\x3c\x6a\x62\x7b\x8a\x78\x18\x8a\x82\x26\x71\x9a\x6d\x38\x29\x2b\xa7\xe3\xb3\x83\x70\x05\xb6\x1e\xee\x8f\x64\x17\x71\xd1\xb7\x66\xcd\xd7\x86\xaf\x26\xda\x66\x06\x68\x00\xc4\x81\x37\xd0\x49\x83\x93\x1f\x6e\xf7\xdb\xfe\xfb\x60\x36\x3a\xe4\x07\xf4\x96\xef\x62\x6f\xbb\xdb\x9f\x5c\x5d\x5e\x92\xfd\xeb\x68\xb8\xfb\x0b\x0c\x71\xdf\xea\xdf\x7f\x88\xb7\x6b\x47\x78\xbb\x64\x80\x4f\x93\x14\x7f\xfd\xc8\x6a\x78\x41\x81\xc3\xb0\x8e\x85\xe2\xdb\x38\x65\xa9\x6f\xf5\xcd\x85\xed\x0b\x74\x66\xfb\x29\xba\x12\x46\x1e\xc2\x1e\x98\x3b\x2d\x9d\x37\xcd\x0e\xf9\xd0\x73\x82\xdf\xbe\x1f\x5e\x5f\x6f\x7e\xbf\xfe\xe8\xce\x7e\x74\xbc\xf7\xc7\xbd\x5f\x67\x57\x47\x4d\x3d\xd7\xc7\x2c\xa0\x4e\xcd\x6c\xfe\xfe\x79\x7b\xd2\x9d\x6c\xfd\x72\xea\x9c\x7d\x38\x43\xdd\x4b\xf1\xcb\x4e\xf7\xf2\xeb\x7e\x6f\x16\xe1\x92\xbf\xe6\xb0\x54\xcb\x15\x9d\xbd\xe5\x95\x5c\xa7\x5e\xc7\x75\x4a\x40\x49\xa6\xe8\x35\xe6\x64\x3c\x83\x5f\xbf\x9d\x9a\xcb\x24\x07\x70\x1c\x86\x7c\xe3\xdb\xb5\xc3\x2c\x6d\x7d\xd5\xe4\x42\xc8\xf4\xce\xa6\x07\xd3\x1b\xef\xf7\xb7\xfe\xb7\x2f\xe3\xc3\xae\x7b\x84\x2f\x7d\xa7\xff\xc7\x7e\x84\x4c\xfe\xc9\xa6\x52\xc1\xbf\x3f\x30\xfd\x5a\x5c\xfa\x65\xb0\x08\xcc\xa1\x39\x66\xac\x35\x42\xbc\x19\x69\xfd\x79\xb7\x8c\xb7\x6b\x94\xc0\xf7\xde\x19\x39\x98\xfe\xa0\x29\x2c\x2e\x7c\xa7\xff\x7d\x2f\xc6\xe2\x13\xba\x0d\xb7\x8e\x0e\xc3\xad\x82\x63\x73\x81\xc6\x02\x20\x6d\xde\x1f\xa4\xcd\x5a\x90\x36\xe7\x83\x34\x45\x22\xbe\xf2\x23\xd9\xcc\xa2\x71\x06\xc1\x16\xa0\x70\x67\x2c\xde\x0a\x99\x0b\xd8\xe5\xad\x02\xec\xb7\x2f\xf8\xb0\xcb\x8e\xf0\x85\xd3\xfb\xfd\x6d\x8c\xd7\x29\xe6\x9e\x38\x62\x72\x37\xbc\xcf\x6d\x91\x59\xd6\x7d\x80\x59\xd6\xad\x9f\x65\xdd\x12\xa4\xe2\x99\x24\x15\xcf\xe6\x76\x14\x73\xcb\x04\xa6\x10\xdd\x47\x57\x89\xc5\xe5\xef\x7b\x3f\xbe\x69\x08\x22\x2c\x3e\x5e\xbf\x7b\x7d\xf1\xe9\xeb\xf7\x08\x8b\xd7\x47\xc8\xc3\x7b\x8c\x8e\x5d\x62\x2f\x12\x0b\xe9\x6d\xdd\x1f\x87\x34\x8d\x12\x1c\xd2\x3f\x67\x55\x70\x7c\xc7\x85\x36\xd4\x44\x00\x72\x75\x78\x5e\x5f\x76\x57\x09\xc2\xd6\xe5\x77\x4b\x09\xc4\x8f\x04\x8d\xef\x78\xea\xf4\x0e\x42\x65\x52\xbc\x24\xb5\xac\xe3\xaf\xef\xdf\xef\xd7\xb5\xdd\x7e\x5d\xaa\x63\xc3\xeb\xf0\xa2\xcb\x67\x6b\x54\x26\x3e\x88\xc6\x76\xeb\xfb\x64\x3a\xfe\xf4\x7a\xf2\xfe\x58\xfc\x72\x7d\xf0\x2d\xee\xe5\xc2\x46\xf6\x49\xfa\x6a\xb6\x1d\xa3\x3b\x12\x41\xf9\xc5\x02\xcb\x01\x7c\xde\xfb\xd4\x3a\xf8\xbd\xf5\x7a\x10\x06\x62\xcd\xa5\x86\xaa\x27\x49\x19\x7c\x2b\x5b\x99\xc0\xf4\xad\xd5\x73\xa9\xe3\x7a\x57\xd6\xd5\xd8\xde\x16\x44\xa2\x4d\xe1\x5e\x5c\xef\xa4\x17\x70\xca\xd1\x8b\x04\x4a\x75\xbb\x33\xd9\x74\x76\x76\xae\x2c\x97\xdb\xce\x75\x7f\xb2\x8d\xdc\xd1\xb6\x70\xc7\x13\x7a\xd1\x73\xa6\x23\x71\xf1\xb7\xff\xf3\xf7\x83\xdf\x4f\x8f\x77\xe1\xff\x99\x3e\xb6\x35\x28\x6f\x92\x23\xdb\x29\xda\x44\x40\xb3\x6f\xf5\x9b\xeb\xba\xf7\xfa\xe3\xde\xc7\xb3\x93\xd3\x83\xe3\xc8\x74\x58\xfd\xa6\xde\xc7\x8c\xc7\x31\x7d\xf6\x5b\x95\xef\x4c\x36\x19\xdf\xb4\xae\x49\x60\x6d\x33\xac\x46\x69\xca\x2f\xed\xee\x96\x33\x19\xcb\x8b\x0e\xb2\x33\x37\x16\x47\x27\x8d\x9b\xf3\x3a\x91\x72\x4c\xfe\x51\x67\x7f\x4f\xc5\x37\x3e\xdb\xa2\xe2\x6a\xd4\x15\x47\xde\xbb\x8b\xcd\xd1\xef\xfe\xfe\xf6\x1e\x6a\xac\xfd\x6f\x00\x00\x00\xff\xff\x70\x02\x25\xce\x4c\xb5\x00\x00")

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

	info := bindataFileInfo{name: "kas-fleet-manager.yaml", size: 46412, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
