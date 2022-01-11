// Code generated by go-bindata. (@generated) DO NOT EDIT.

//Package generated generated by go-bindata.// sources:
// .generate/openapi/connector_mgmt.yaml
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

var _connector_mgmtYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x7d\x7d\x73\xdb\x38\x92\xf7\xff\xfa\x14\xfd\x28\xcf\x96\x77\xef\xac\x17\xbf\x24\x4e\x54\x97\xa9\x72\x6c\x27\xe3\x49\xe2\x24\xb6\x33\x49\x76\x6b\xca\x82\xc8\x96\x04\x8b\x04\x68\x00\x94\xad\xec\xed\x77\xbf\x02\x40\x8a\xef\x12\xe5\x78\x62\x4f\x22\x57\xa5\x2a\x22\x1b\xcd\xee\x46\xe3\x87\xee\x06\x08\xf2\x00\x19\x09\x68\x0f\x76\xda\xdd\x76\x17\x1e\x01\x43\x74\x41\x8d\xa9\x04\x22\x61\x48\x85\x54\xe0\x51\x86\xa0\x38\x10\xcf\xe3\xd7\x20\xb9\x8f\x70\x7c\x78\x24\xf5\xa5\x09\xe3\xd7\x96\x5a\x37\x60\x10\xb1\x03\x97\x3b\xa1\x8f\x4c\xb5\x1b\x8f\x60\xdf\xf3\x00\x99\x1b\x70\xca\x94\x04\x17\x87\x94\xa1\x0b\x63\x14\x08\xd7\xd4\xf3\x60\x80\xe0\x52\xe9\xf0\x29\x0a\x32\xf0\x10\x06\x33\xfd\x24\x08\x25\x0a\xd9\x86\xe3\x21\x28\x43\xab\x1f\x10\x49\xc7\x61\x82\x18\x58\x49\x12\xce\xcd\x40\xd0\x29\x51\xd8\xdc\x04\xe2\x6a\x1d\xd0\xd7\xa4\x6a\x8c\xd0\x9c\x10\xd9\x1a\x7a\x88\xaa\xe5\x13\x46\x46\x28\x5a\x11\x71\x7b\x46\x7c\xaf\x09\x43\xea\x61\x83\xb2\x21\xef\x35\x00\x14\x55\x1e\xf6\xe0\x80\x33\x86\x8e\xe2\x02\xce\x50\x4c\xa9\x83\xf0\x52\x73\x80\xb7\x96\x43\x03\x60\x8a\x42\x52\xce\x7a\xd0\x6d\x77\xdb\x3b\x0d\x00\x17\xa5\x23\x68\xa0\xcc\xc5\x25\xed\xad\x42\xa7\x28\x15\xec\xbf\x3f\xd6\x92\x5a\xd1\xc0\x89\xdb\xc9\x76\x43\xa2\xd0\x0f\xd1\x52\xb5\x20\x14\x5e\x0f\xc6\x4a\x05\xb2\xd7\xe9\x90\x80\xb6\xb5\xb5\xe5\x98\x0e\x55\xdb\xe1\x7e\x03\x20\x27\xc0\x5b\x42\x19\xfc\x3d\x10\xdc\x0d\x1d\x7d\xe5\x1f\x60\xd9\x95\x33\x93\x8a\x8c\x70\x19\xcb\x33\x45\x46\x94\x8d\x4a\x19\xf5\x3a\x1d\x8f\x3b\xc4\x1b\x73\xa9\x7a\x4f\xbb\xdd\x6e\xb1\xf9\xfc\x7e\xd2\xb2\x53\xa4\x72\x42\x21\x90\x29\x70\xb9\x4f\x28\x6b\x28\x32\x8a\x0c\xc0\x88\x9f\xe9\x97\xf3\x59\x80\xb2\xd8\xbe\xd9\x2c\xa3\xae\x4d\x08\x07\x5e\x28\x15\x56\x34\x68\x04\x44\x8d\x8d\x3c\x8f\xf4\x3f\x38\x1f\xa3\x44\x20\x02\x8d\xa3\xcd\xfb\x0e\x04\x7a\x44\xa1\xab\xfb\x56\x46\xc4\x4d\x6d\xe7\xce\x9c\xe4\xc2\x1f\xf9\xaa\x33\xdd\xea\x4c\xc8\x70\x42\x2e\x92\xeb\x4a\xab\xd5\xf9\x77\xf6\xc2\x05\x75\xff\xd3\xec\x19\x91\x02\x22\x88\x8f\x2a\xf2\x0b\xfd\x17\xeb\x50\x68\x12\xdd\xcf\xa9\x71\x3e\x46\xa0\x2e\xf0\x61\x4e\x66\xdd\x68\xde\x42\x3a\x63\xf4\x49\x6f\xfe\x1b\xcc\xed\x1e\x48\x25\x28\x1b\xcd\x2f\x53\xd6\x03\x6d\x92\xf9\x05\x81\x57\x21\x15\xe8\xf6\x40\x89\xd0\xb2\x1b\xa1\x8a\xf9\xc4\x9d\x19\xcb\x5d\xd6\x99\x9a\x87\x0c\x38\x93\x98\x22\x6d\x6e\x77\xbb\xcd\xb4\x34\x0e\x67\x0a\x99\x4a\x5f\x02\x20\x41\xe0\x51\x87\x68\x35\x3b\x97\x92\xb3\xec\xdd\x32\xa5\xec\xdf\xff\x17\x38\xec\x41\xf3\x51\xc7\xe1\x7e\xc0\x19\x32\x25\x3b\x96\x56\x76\xe6\x22\x6a\x09\x9b\xa9\xa6\x05\x9b\x66\x2d\x09\x3e\x51\xce\x58\x8f\x16\x6d\x65\x6d\x16\x34\x8e\x1f\xe9\xb3\xdb\xdd\xba\x1f\x7d\x8e\x84\xe0\xa2\x99\x6b\x82\x37\xc4\x0f\xbc\xb4\xc1\xe3\xbf\xdd\xee\xd6\x91\xbd\x59\xbc\x57\xfe\xa0\x98\x57\x27\x69\x5a\x69\xb6\xfd\x50\x8d\x41\xf1\x09\x32\x8d\x87\x94\x4d\x89\x97\xf2\xda\xe6\x6e\x77\xf7\x2f\x62\xa4\xdd\xdb\x1b\x69\x77\x99\x91\x4e\x78\xe2\x4b\x39\x1f\xc3\x1b\x2a\x95\x4c\x0c\xf6\xf8\xbe\x46\xc9\x8a\x06\x7b\xdc\xed\xde\xd6\x60\x49\xd3\x4a\x83\x7d\x64\x78\x13\xa0\xa3\xf1\x17\xb5\x5c\xc0\x1d\x33\xa9\xc4\x9e\x25\xd1\x09\x05\x55\xb3\x34\x12\xbd\x40\x22\x50\xf4\xe0\x5f\xf0\x47\x74\x95\x07\x28\x8c\x91\x8e\xdd\x9e\xc6\xb0\x0c\x10\xbc\x98\x1d\x1f\xc6\xdc\x42\xdf\x27\x62\xd6\x83\x57\xa8\x80\xe4\x7b\x68\x30\x03\xea\x36\x1a\x00\xab\xa0\x7f\xef\x76\xb8\x79\x0b\xbd\x3c\x2a\xb3\x8a\xc9\xbc\x56\xa7\xa8\x42\xc1\x74\xb0\xa2\x69\xf5\xac\x91\xd5\x30\x6e\x50\x9c\x96\xf4\xe3\xcb\xba\x32\xa1\xec\x04\x64\x94\xea\xc6\xa5\xe4\x92\x7e\x5d\x85\x9c\x0b\x17\xc5\x8b\xd9\x2a\x0f\x40\x22\x9c\x71\xf3\xc1\xcf\x43\x6f\xa8\x54\xd5\xa0\xba\xa4\xa7\xd6\xb3\x4f\xbd\xd9\x67\x0d\xa6\xcb\xc0\xb4\x1e\xa6\x45\x22\x07\x3a\x31\x58\x86\x67\xb2\x0c\xa4\x1c\x81\x44\xe1\x9c\x66\x21\xe0\xe8\x80\xf4\x2a\x44\x31\x4b\xe9\x63\xa3\x63\x22\x67\xcc\xa9\xd2\xf2\x3d\x8a\x21\x17\xbe\x89\xd5\x88\x49\x97\x80\x32\x9d\xd2\x9a\x56\x63\xc1\x19\x0f\xa5\x4e\xd1\x18\x8a\xc6\xe2\xde\xb5\x71\xf2\x80\x73\x0f\x09\x4b\xdd\x29\x89\x8c\x21\x8e\x0b\x5f\x70\x37\x85\xda\x15\x79\xa4\x4b\x14\x99\xd3\x94\x38\xe3\x62\x57\x2c\x77\xc4\x5a\x88\xd3\x5c\x14\xdd\x57\xc1\xe4\xf6\x3d\xc3\x64\xf5\xa8\x77\x1c\x0c\x14\x66\xc2\xcc\xbf\xc6\x40\xdf\xed\x76\x0f\xf4\x50\xa0\x9c\xdd\x1e\x15\xf3\x2c\x2a\xed\xf4\xbb\x46\x43\x43\x69\x07\xbe\xcc\x87\x51\xeb\x79\x64\x9d\xc5\xd4\xcf\x62\xce\x93\x2c\x18\x5d\x8d\x19\x3c\x14\x0e\x82\xcb\x51\xb2\x0d\x65\x33\x99\xf5\xdc\x9b\x73\x2c\x06\x61\xd5\xf4\x6b\x67\xc5\xb8\xbe\xe0\xe4\x26\xc7\x7a\xa9\xc0\x3c\xc8\x37\x98\x80\xa6\xc2\x7a\x9d\xe2\x55\x3b\x0d\xb9\xcf\x24\xa0\x7a\xc6\xb7\x71\x48\x6a\x28\x96\xb8\xa4\xa1\x01\xc7\xd6\xfc\x20\x43\x5b\x3d\xb9\x67\x8a\x60\x0f\x33\x4b\x58\x35\x43\x58\x27\x07\xeb\xe4\xe0\x41\x54\x5a\x32\x15\x89\x5b\x55\x23\x6a\x16\xda\x65\xe7\xdf\x8b\x8b\xea\x4b\x70\x88\xba\xcd\x39\x69\x11\x83\x2a\x10\xa8\x3e\xfe\xd4\x28\xc0\xaf\x82\xcc\x0f\x13\xa5\x6a\xd6\xd3\xd7\xa5\xf4\x75\x10\x5a\xc7\x48\xeb\x52\xfa\x4a\x06\xbb\xff\x52\x7a\x1e\xde\x73\x25\xf4\x86\x95\xc7\x43\x85\xdf\x08\x74\x0f\xc5\xf9\x33\xc6\x3d\x34\x9a\xad\xd3\xe9\x1f\x16\xc9\x6c\x07\x7f\x03\x9e\x65\x18\x2c\x42\x35\x1b\x46\x44\x53\x23\x5c\x53\x35\x06\x19\xa0\x43\x87\x14\x5d\x38\x3e\xfc\x2b\xa3\xdb\xb7\x19\x31\xcf\xe0\x7b\x22\x9d\x45\xae\x4a\xb0\xb3\x72\x15\xf0\x2e\xd0\x13\xd4\x37\xc2\xdd\x83\x2d\xbe\xae\xe3\xba\x9f\x17\x0d\xd7\x71\xdd\x0f\x1d\xd7\x19\xdc\xaa\x04\x3b\x73\xb7\x80\x75\x75\x16\xc0\x0e\x89\x22\xa0\x78\xc4\x21\xbb\x65\x4d\x4f\x74\x8d\x05\x3d\xf9\x1d\x96\xc4\xb2\x0f\xf1\x51\x8c\xb0\x65\x44\xfd\xef\xef\xf2\x40\xfd\x90\x15\x9f\x97\x2b\x05\xfe\x76\xf6\xee\x04\xde\x6b\x0e\x9b\x70\xfa\xf2\x00\x9e\x3c\xeb\x6e\x43\x6b\xbe\x39\x54\x71\xee\xc9\x36\x45\x35\x6c\x73\x31\xea\x8c\x95\xef\x75\xc4\xd0\xd1\x54\x39\xbe\xb6\x3e\xc1\x07\x97\xe8\x24\x28\x9e\x5b\x31\x34\x9b\x26\x8f\xe6\x9b\x76\x87\x66\x12\x60\x76\x57\x69\xd2\xad\x51\x39\x44\x02\x61\x6e\xea\x32\x19\x21\x53\x10\x2d\xe3\xd6\xdc\x4f\x13\xb3\x8a\xca\x3c\x75\xd6\xa0\xb3\x3b\x40\x17\xaf\x45\x47\xa4\x11\xe5\x4f\xb7\x24\x1d\xd7\xad\xee\x6d\x69\x3a\xb2\xff\x5f\x72\x85\xba\x20\xfb\x7a\xa1\x7a\xbd\x50\xbd\x8e\x25\xeb\x18\x69\xbd\x50\xfd\xb0\xc2\xc9\x5b\x2c\x54\xc7\x73\xc7\x4a\x01\xe7\x92\x05\xeb\x0c\xcf\x5a\xdb\x67\x73\x53\xfd\xf7\x5e\xbf\x7e\x98\xab\x32\x91\x51\x56\xde\x64\xea\x64\x8d\xb9\x86\xdd\xf5\x52\xf2\xc3\x59\x4a\xce\x8d\xf4\x5a\x2b\xca\x29\x87\x5e\x31\xdd\x48\xbf\xbf\x15\x5d\xbb\xa0\xee\x7f\xa2\x2c\xa4\xc6\x1b\x5c\x49\xa3\xf2\x18\xbc\xea\x25\xae\x2c\xaa\x7e\xff\xf7\xb8\x72\x66\x7e\xd0\x00\x57\xb3\x4a\x19\x27\x39\xeb\x6a\xe5\x3a\xc2\xac\x63\xa4\x5b\x56\x2b\x63\x37\x5b\x57\x2d\xef\x6b\x35\x3a\x5b\xcc\xa9\x7c\xaf\x2b\x86\x58\x8b\xe5\xe1\x5d\x81\x62\x76\x64\x64\x0b\x2e\x91\x6b\x48\x45\x54\x68\x5e\xb6\x0f\x03\x97\xac\x97\x8e\x0b\x86\x5a\xc3\xcf\x37\xc3\xcf\x1a\x79\xca\x6c\xf6\x27\x20\x8f\x1d\xc2\x79\xf0\x79\x31\x3b\x76\xf3\x00\x14\xba\x01\xc9\x2e\x14\xe7\xc2\xbc\xfa\x8b\x28\x11\x6e\x3c\x84\x25\x94\x65\xa5\xdb\x06\xd4\xdf\xfc\x73\x1b\x84\x5d\x6f\x02\x5a\x23\xf9\x8f\xb5\x09\x68\x5e\xab\x5c\xef\xff\x79\x88\xfb\x7f\x2a\xc2\xcb\xe2\x36\xa0\x74\x84\x79\x37\x75\x87\x0e\x71\x5d\xce\x2e\xf2\x85\x87\x75\x21\xe2\x7e\x06\xc1\xbe\xee\x8d\xf7\xb1\xf1\x17\xd6\x59\xb5\x79\x93\x6e\x02\x35\x26\x0a\xe4\x98\x87\x9e\x0b\x03\x84\x50\xda\x03\xb1\x1c\xce\x86\x74\x14\x46\xc7\x10\xd9\x93\xa4\x32\x2b\xe7\xfa\x81\xc0\x99\xed\x23\x6b\x99\xf6\x7a\xca\xf9\x51\xa7\x9c\x75\xed\xe2\xa7\xc8\x20\x4a\x6a\x17\x59\x64\x29\x14\xbb\x2b\x4a\x19\x1b\x32\x42\x88\x04\x69\x1a\x8d\x44\x4f\x2d\x52\x64\x53\x2b\x5d\xe6\x2c\x92\x58\xe0\x8c\xb6\xbf\xb4\xe6\x6a\x9c\x62\x20\x50\x6a\x3e\xc5\xd3\x71\x64\x18\x04\x5c\x68\x9b\x0c\x66\x06\x9b\xf6\xdf\x1f\xc7\x9a\x32\x7c\x37\x4c\x1b\x63\x3e\x11\xa4\x2c\x6c\xa7\xab\xcc\x85\xe8\x54\xbe\xcc\x35\x2b\xfc\x12\x5e\x25\xdc\xca\xf9\xe9\xab\xda\x45\x2f\x32\x6c\x89\xe7\x65\xe5\x5d\xe4\x99\xef\xcc\x26\xa9\x53\x1c\xa2\x40\xe6\x64\x56\x0f\x4b\x77\x51\x01\x04\x42\x77\xbd\xa2\x79\xc7\x35\xd3\x75\xce\x5d\xb3\x03\x96\xf8\x58\x7e\xd6\x5b\x3b\xd7\xac\x74\x3e\xd6\x7f\xf1\x41\x87\x8b\x1e\xf3\xbb\xa5\xf9\xc6\x27\x39\x63\xc2\x18\x7a\x85\xc1\x99\xad\x86\x45\x44\xab\x3c\x8b\x08\x41\x66\xb9\x3b\x54\xa1\x5f\x02\x03\x95\xc2\xa5\x85\x58\x24\xdf\x7e\xfa\x67\x41\xc8\xda\xb6\xa0\x0e\x67\x17\x63\xed\x44\x8b\x1e\xf6\xf1\xf4\x8d\x39\x9c\x93\x19\xfa\xdb\x3f\xcd\x23\x83\x65\x76\x7f\x63\x48\x92\x98\x83\x28\x1c\x71\x41\xbf\x62\xe9\x1b\xe8\x77\x6f\x7f\xfd\x87\x2c\xf4\x35\x1e\x4a\xca\x26\x9b\x10\xa5\x3b\x7f\x64\xc8\x6a\x6c\x7a\x4c\xa1\x53\xfc\xb7\x6f\x86\x74\xd4\xd8\x86\x59\x0e\x61\xe9\x18\x6b\x6a\xb7\x19\x65\x22\x75\x59\xe0\x93\x04\xce\x3a\xff\x82\x21\x45\xcf\x2d\xef\x04\x3b\xc4\xe1\x11\x28\xee\xf2\x1e\x08\x0c\x3c\x12\xe7\x6e\x03\x54\xf1\x74\x9a\x69\x9b\x82\x9d\x1f\x4b\xc1\xc2\xe9\x56\xbd\xdb\xc0\x6a\x36\x92\x5e\x19\x4b\x4b\x1d\x72\x65\xff\x5d\xe1\x30\xc9\xac\xea\x3f\x81\xda\x79\x95\x4b\x63\x87\xfd\x14\xa2\x8f\xb9\xe7\xca\x18\x5f\x4c\x8a\x63\x37\xfa\xd9\x9c\x47\x13\x01\x81\xd7\xe6\xe5\x27\xc5\x03\xea\x58\x2c\xe4\x6a\x8c\x02\xe4\x4c\x2a\xf4\xdb\xf7\x3b\x3f\xfb\xa8\x88\x4b\x54\x61\xb8\x56\xb0\x59\xc4\x4a\xff\xc5\x2f\x94\x97\x45\xa8\x0b\x81\x93\x5f\x33\x14\x2b\xb7\x2a\x8b\x2e\x96\x36\xb2\x5b\xa4\xdd\x0b\xa2\xca\x9a\x0e\xb9\xf0\x89\xea\x81\x46\x9a\x96\xa2\xb9\x78\xab\x06\xfb\xa8\x84\xfd\x67\xb1\x8f\x8b\x68\x17\x15\x51\x4f\xc2\x80\x32\x85\x23\xcc\xcf\x77\x69\x21\x28\x53\x4f\x76\x73\x31\x44\xe0\xf1\x99\x8f\x4c\x5d\x78\xdc\xa6\x3b\x2b\x15\xcf\x6d\xbc\x7e\x4e\xc4\x08\x55\x36\xb3\x31\x9e\xb1\x0a\x2f\x33\x6a\xa2\x91\x48\x39\x3b\x43\xa5\x28\x1b\xc9\x2c\xd7\xc2\x91\xc2\xe5\x7e\x5c\x16\xca\x65\xe6\x89\xda\xee\x1f\x45\x80\xb5\x1f\xe3\xa2\xd4\x81\xfc\x85\x54\x44\x15\x5c\xb5\xb2\x95\x5d\x48\xad\x41\x7e\xd3\xf2\x28\x9b\xa4\x28\xb3\x06\x49\x73\xa8\x7b\x5e\x28\x54\xec\x33\x2c\x37\x37\x6c\x14\xae\x6d\x54\xa5\x98\xbf\x2c\xe0\xa5\x25\x92\x25\xe1\xb2\x4e\xbc\x42\x19\xed\x6a\xc9\xb4\xd7\xb4\xfd\xc2\xc3\xfb\x89\xec\x40\xd9\xfc\x8d\x53\xc5\x33\x6d\xfb\xaf\x8e\xce\x57\x3a\xf1\xb4\xec\xbc\xeb\x7e\x3b\x9a\x2d\xd2\x4e\x3f\x9f\x31\xa8\x56\xdc\xa7\x8c\xa4\xa6\x91\x39\x76\xce\x4e\xec\xf1\x1b\x94\x25\x05\x1c\x9f\x04\x01\x65\xa3\xcc\xda\x96\x4e\x7a\x17\x16\xe9\x2a\x07\x9c\xe3\xf1\xd0\xbd\x08\x04\x9f\x52\x57\xa7\xe8\x55\xe3\x95\x87\xee\xfb\x88\xa8\x94\x57\x21\xcd\x5d\x5a\x37\xac\x10\x69\x71\xcb\x45\x82\x18\x16\x45\xd6\xa5\x73\x73\xd3\xde\x93\x70\xcd\xc5\xc4\xe3\xc4\x95\x51\xf6\x61\xeb\x07\x4e\x76\x4d\xaf\x64\x94\x17\x33\xef\x56\xba\x9b\xca\xe6\x3e\x7d\x7b\x69\xdd\x38\x29\x53\x57\x92\x46\xce\x54\x65\x88\x55\xf4\xb5\xdd\x0f\x71\xf7\xdf\x8b\xbe\x19\xff\x5b\x46\x2e\x70\x94\x9b\x6a\x4a\xc9\xfc\xd0\x53\xf4\x82\x7c\x2d\x12\xc6\x2f\x0f\x25\xce\x92\xa9\x5f\x97\x1b\x2f\xd9\x24\x9c\x2f\x2f\x65\x2d\x96\x0e\x33\x73\xe1\x65\xfd\x32\x7a\xb3\x4c\xb6\x2a\xb9\x72\xf2\x2c\xe8\xc0\xb2\x1e\x5a\xe0\x64\xf1\xc5\x29\xf1\x42\x5c\xe2\x8a\xb9\xda\x5d\xb9\xac\x67\x36\x61\x1b\x6a\xbc\x4e\xf6\x1d\x26\x6b\xf9\x40\xcc\xdb\x57\x10\x78\x84\x61\xaa\xa0\x67\xa7\xb8\xe6\x0f\x15\xfd\xae\x83\xd8\x1a\xec\x6b\xc4\x36\xa5\xee\xf7\x13\xa4\x9f\xf3\x2d\x27\xa6\x79\x45\x00\xdc\x5b\x30\xf0\x07\x9c\x2b\xa9\x04\x09\x2e\xec\x27\x50\x6a\xc0\x34\xd5\xb1\x7e\x0d\xc0\x88\x28\x25\x3a\x02\xd5\x62\xe0\xc8\x8d\xcf\xde\x9d\x83\x57\xad\x19\x28\x5f\x9a\x2c\xca\x99\xf6\xa8\x25\xf3\xa0\xfe\x19\x90\x11\xa6\x7e\x4a\xfa\x35\xfd\x53\x71\x45\xbc\xd4\x6f\xe3\x07\xab\x69\x5e\x4b\x2d\x2d\x45\x91\x28\x9f\xeb\x69\xe1\x96\x53\x19\x99\xab\xc9\xcc\x0d\xb3\x48\x75\xab\x71\x77\x77\x30\xed\x70\xb7\x7e\xee\x24\x90\x94\xac\xcf\x55\x92\xcf\xd3\xa2\x65\xc9\x63\xa3\x98\x16\x25\x2d\xec\xae\x84\xf9\x72\x6c\x61\x89\xfc\xf8\x50\x47\x19\x02\x1d\x2e\xe6\x0b\x6a\xb9\x32\x69\x89\x84\xb9\xcd\x06\x25\x5b\x0d\xd2\xde\x60\x65\x48\x79\x69\xfe\xc5\xee\xec\xeb\xdb\x64\x84\x40\x99\x8b\x37\x05\xee\x43\xe2\x49\xac\x2f\x65\x71\xc5\x32\xef\xa3\x36\xd8\x80\xe6\x96\xf5\x81\xb4\x73\x5a\xa1\x53\x63\x69\xa1\xd0\x27\xa1\x3f\x40\xa1\x4d\x69\x86\x97\xce\xf0\x90\x38\xe3\xb4\xd2\x77\xa8\x46\x7e\x10\xcd\xd5\xe8\x76\xad\x22\xd1\x07\x1c\x4a\x03\xa3\xff\x4d\x4a\xdc\x67\xd1\x4e\x27\x9b\xdf\x9a\x46\x3a\xad\x75\x04\x55\x28\x28\x69\x1b\x0f\x91\x33\xa6\xc8\x8d\x5d\x20\xa1\x32\x9d\xc5\xca\x94\x40\x3e\xf5\x88\x88\xbf\x1b\x96\x6e\x82\xd0\x8f\x19\xf7\xc1\xf1\x48\x28\xcd\xaa\x1a\x61\x70\xf6\xe1\x8d\x99\x73\xd1\x7e\xf1\x2c\xe6\x75\xa4\xed\x66\x0c\x1d\xd7\xd9\x4d\x7b\xbb\xd4\x49\xd8\x6c\xce\x36\x53\x27\xe8\xdb\x82\xba\x4c\xf8\xbc\xe4\x22\x36\xdd\xa6\x16\x4c\x98\xf7\x95\xcc\x37\xd2\x0e\xb2\x47\x5f\xa6\x1f\xa0\xc6\x48\x85\xe9\xfc\x4d\xd0\xa2\xea\x27\x0d\xb9\xe7\xf1\x6b\xf3\xfd\x2e\xa3\x58\xaf\x31\x7f\x48\xbf\xdf\x97\x57\x09\xba\xea\x76\x40\xa4\x93\xbe\x9f\x10\x9f\xaf\x2e\x04\x5c\x10\xe6\x5e\xc4\x0b\x89\xdf\x22\xd2\x66\xcc\xa4\x5a\x3e\xfb\xcd\xb8\x4c\x0f\xb3\x0d\x15\x27\x6b\xee\x26\x70\x01\xd4\xd2\x18\x8f\x03\x2a\x01\xfd\x40\xcd\x36\xf5\xb5\x64\xa5\xd7\x86\xdb\x32\xf4\x94\x34\x5f\xf8\x4a\x69\xa6\xa5\x69\xcf\xfd\x3a\xf0\x34\x7e\xa6\x0f\x1c\x28\xfa\x7a\xce\x95\xd3\xee\x1e\xab\xd6\xac\x18\xa1\x76\x08\x47\x0c\xbe\x75\x14\x4a\x35\xf3\xb0\x67\x22\x4c\x8b\x15\xe6\x8b\x27\xe5\x23\x2c\x19\x60\x86\x28\x19\x50\x29\x5f\x58\x3c\xb2\x96\x8c\xa8\xeb\x31\x0a\xcc\x0c\xa7\xe4\x91\x99\x51\x05\xfb\xda\x4f\xd0\x8d\x46\x87\xc6\x25\xc3\xce\xca\xa5\x3b\xa7\xaf\xad\xd4\xdf\x84\x7e\x4a\x05\xfd\x33\xf2\x16\xfd\x5f\xb3\xc4\xd9\xdf\x34\x87\x8a\xf4\xa3\x3a\x63\x3f\x61\x6d\x27\x2a\x2e\x6c\x67\xf7\xff\xe7\x17\xdd\xe6\x79\xdf\xb8\x4b\xff\xcd\xf1\xeb\xa3\x7e\x32\x28\xe3\x36\x0e\x67\x97\x21\x73\x14\x9d\x62\xbe\xfd\xfe\xc9\x61\xdf\x3e\xea\xdd\x69\xbf\x0d\xbf\xf2\x6b\x9c\xa2\xd8\x84\x19\x0f\x0d\x20\x68\x8d\x09\xf8\xe4\x86\xfa\xa1\xaf\x75\xdf\xea\x26\xec\x38\x33\x3a\x92\x58\x43\xe3\x0e\x29\xb3\x1f\xcd\xfd\xab\x6c\x54\xe6\x3e\x0f\x64\x57\xdc\xb4\xbd\x8c\xa7\xf5\xc9\xb5\x6c\xc9\x2b\xd9\xb2\x75\x6e\x2b\xa4\x29\x0c\x5a\x93\x40\x5f\x2a\x32\xf0\xb4\x31\xeb\x0d\xd3\xec\x18\x7d\x0e\x59\xfe\xf6\x0c\x97\x88\xf5\x73\xb0\xbc\xd3\xcd\xff\x15\xb4\xfe\x28\x57\xc3\xae\xeb\x50\x26\x15\x61\xf1\xc2\x21\xb1\x4f\xb1\xfb\xe0\x14\x11\x4a\xda\xeb\x5a\xab\x5b\x4a\xec\xd1\x09\x6a\xa1\xff\xb6\xfd\xf8\x4f\x01\x14\x03\x93\xfa\x66\xb6\x5b\x52\x38\x43\x94\xb9\x1f\x4a\x14\x30\x26\x12\x02\x14\x3e\x95\x32\x5a\xe8\x92\x68\xbf\x30\x68\xed\x82\x6e\xca\x0f\x4e\xb8\xc2\xf8\x23\x99\xd1\x64\x93\x6c\x5d\xd3\x9e\x1e\xed\x35\xa2\x32\xd5\xba\x1a\xb6\xa2\x60\xc1\xf8\x5c\x05\x18\x95\x03\x4f\xc9\xdc\x9e\xc1\x15\xc8\xc3\x5d\x2d\x2f\x69\xde\x0e\xd6\x1a\xc9\x66\x2a\x53\x3c\x89\xc5\x8a\x76\x53\xa5\x99\x62\x0f\x06\xe6\x6a\x74\xd1\xfe\x78\x19\xa5\xdf\xbf\x7d\x3a\xcf\x24\x18\x63\xa5\x02\xcd\x3d\xab\x6d\xf5\x61\x27\xb9\x1a\x10\x75\x4d\x6c\xb3\xd3\xcc\x26\x24\xd0\xcc\xed\x53\x1b\xdb\x98\xbf\xa2\x7c\x6e\x8f\x38\xe9\x64\xf8\x98\xf8\x1d\x9a\x07\xef\x4e\x4e\x8e\x0e\xce\xdf\x9d\xb6\xde\xbe\x7a\x7b\xde\xca\x90\x44\x51\x3b\x34\xcf\x52\x87\x07\xc5\xc7\x0a\x49\x60\x5c\x25\x9b\xb1\xec\x30\x32\xc7\x0c\x3d\xd7\xde\x51\xac\x93\xe5\xc2\x7a\x68\x6e\xd1\x4f\xc7\xd4\xbf\x7a\xe5\x88\xc3\xf0\xcd\x93\x2d\xf2\xf1\xe6\xf8\x9f\x57\x2f\xce\xaf\x4e\x4e\x49\x33\xb6\xd2\xb1\x75\xcb\x0f\xda\x9b\x6a\x58\x6a\xfb\x8e\x2c\xb5\xbd\xd4\x50\xdb\x65\x76\x7a\x49\xa8\x67\xf7\x45\x04\x44\x48\xcc\x20\x71\x0f\x3e\x32\xf3\xb1\x5a\xc5\x6d\x85\xf3\x75\xfa\xf8\x4d\x7b\xac\x15\x09\xe8\x85\x5d\xea\x90\xf6\xfb\xaf\x3d\x28\x3c\xb6\x07\xcb\x9e\x92\x6c\x90\x73\xb8\x17\xfa\xcc\x22\x96\xe6\x1f\x0d\x30\xd8\xa0\xee\x46\x1b\xce\xca\xe8\xcc\x4c\xd4\x8b\x6a\xbf\x9b\x51\x34\x98\x2d\x1c\xc7\x57\x6d\xd1\xa8\x0d\x1f\x2c\x86\xd8\x9e\xd2\x09\x17\x3c\x87\xad\xb4\x7d\xf2\xfd\xee\x7d\x3a\x7c\x15\xce\x06\xc7\xe2\x88\xdd\x88\x7d\xf4\xf7\xb6\x77\x47\x57\x93\x09\x3d\x9c\xc6\xfd\x9e\xdf\x8d\x5a\xd6\xd7\xbb\xdd\xdd\x3b\xe9\xeb\xbd\x65\x5d\xbd\x57\xd2\xd3\x75\x0e\x7c\x99\x2b\x53\xfa\x36\x42\x99\x4a\x7b\xf7\xa7\x90\xf1\xc6\xd3\xcc\xcb\x20\xd4\x7d\xbe\xb1\x45\x5f\xef\xb8\xe1\xef\x5f\x8e\xa7\xd3\xc7\x5f\xa6\x6f\xbc\xd9\xd7\x2d\xff\xd5\xe9\xce\x6f\xb3\xab\x93\x0d\x03\x00\x43\x1e\x32\x77\xc1\x10\xff\xf2\x6e\x6f\xb4\x3d\x7a\xf2\xeb\xb9\xfb\xf1\xf5\x47\xb2\x3d\x91\xbf\x3e\xdd\x9e\x7c\x38\xdc\x99\xc5\xd6\xc9\xef\xce\x2e\x05\xc0\xad\xbb\xc1\xbf\xad\xa5\xf0\xb7\x55\x62\x9a\x64\xdc\x4e\x51\xd0\xe1\x4c\x63\xbc\xdd\xf3\xdd\x83\xd3\x68\x6a\x03\x12\xaa\x31\x17\xf4\x6b\xbc\xd5\x64\x82\xac\x9e\x7d\x76\x3e\x8e\x8f\xc6\xd7\xfe\xe7\x17\xc1\xa7\xf7\xc3\xe3\x6d\xef\x04\x27\x81\xbb\xfb\xcf\xc3\xd8\x3e\xcf\x4e\x88\x8f\x07\x9c\x0d\x3d\xea\xa8\x1a\xb6\xda\x79\x72\x27\xb6\x4a\xb3\x29\xb7\x55\x9a\x22\xeb\x46\xf3\xad\xdd\x06\x78\xa8\x04\xe2\x09\x24\xee\xcc\xec\x1a\xab\xb4\xc5\x93\xc9\x97\xee\x47\x7a\x34\xf9\x3a\xf9\x7c\xf0\xf5\xd3\x7b\x3c\xde\xe6\x5f\x70\xec\xee\x1c\x45\xa6\x28\x6e\xb5\x2e\x53\xff\xd9\x9d\x68\xff\x6c\x99\xf2\xcf\x4a\xfd\x24\x79\x7d\x0a\xb3\x0f\x2d\x74\x3b\x1e\xbd\x99\xbe\x7c\x76\xf9\xf6\xc3\x97\x27\x5f\x46\xe3\xe1\xdb\x67\xa3\x57\xa7\xf2\xd7\xe9\xd1\xa7\xb9\xae\xb5\x41\xe3\xde\x34\x4e\xc5\x69\x4d\x1b\x32\x9a\x97\x97\xa2\x00\xda\x91\xa8\x7a\xf0\xee\xe0\x6d\xeb\xe8\x73\xeb\x99\x0e\xfd\xe2\x59\xcb\xbe\xe2\x94\xd0\xe0\x8d\x6a\x45\xf3\x1d\x09\x68\x6b\x8b\xde\x74\x77\x3c\xe6\x7a\xfe\x55\xf7\x6a\xe8\xec\x49\xaa\xc8\x63\xe9\x5d\x4e\x9f\xa6\x97\x34\x86\xa9\x37\x08\xb4\x19\xb6\x46\x8f\xdd\xa7\x4f\xaf\xba\x9e\x70\xdc\xe9\xee\x68\x8f\x78\x83\x3d\xe9\x0d\x47\xec\x72\xc7\x1d\x0f\xe4\xe5\xdf\xfe\xdf\xdf\x8f\x3e\x9f\x9f\xee\xc3\x7f\x59\x85\xdb\xc6\x48\xcf\xa9\x8b\x4c\xe9\x2e\x4b\x6f\xc9\xa1\x12\x36\x76\xbb\xbb\x1b\x9b\xc6\x14\xe6\xe7\xc1\x9b\x8f\x67\xe7\x47\xa7\x67\xd6\x16\xfa\xa6\x89\x3e\xe7\xfd\x0a\x09\x23\x43\xbf\x35\x7a\xcc\xc5\xe3\xee\x94\x86\xdd\x3d\x8e\xba\xd7\xc6\x62\xe2\x6c\x3f\x71\x47\x43\x75\xb9\x45\x9c\x8d\xb4\xf5\xe2\x57\xf5\x37\x96\x29\x91\x82\xdc\x7f\x2c\xc2\x94\x73\xf9\x49\xcc\x9e\x30\x79\x35\xd8\x96\x27\xfe\xcb\xcb\xc7\x83\xcf\xc1\xe1\xde\x01\x69\x36\xfe\x2f\x00\x00\xff\xff\xf1\xec\x66\x6b\xbd\x7f\x00\x00")

func connector_mgmtYamlBytes() ([]byte, error) {
	return bindataRead(
		_connector_mgmtYaml,
		"connector_mgmt.yaml",
	)
}

func connector_mgmtYaml() (*asset, error) {
	bytes, err := connector_mgmtYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "connector_mgmt.yaml", size: 32701, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
	"connector_mgmt.yaml": connector_mgmtYaml,
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
	"connector_mgmt.yaml": &bintree{connector_mgmtYaml, map[string]*bintree{}},
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
