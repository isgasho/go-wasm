package tarfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/spf13/afero"
)

const (
	pathSeparatorRune = '/'
	pathSeparator     = string(pathSeparatorRune)
)

type Fs struct {
	compressedData []byte
	// pathSegments holds directory paths and their children's base names. Non-nil means directory exists with no children.
	directories     map[string]map[string]bool
	compressedFiles map[string]compressedFile
}

type compressedFile struct {
	start  int64
	header *tar.Header
}

var _ afero.Fs = &Fs{}

func New(r io.Reader) (*Fs, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	fs := &Fs{
		compressedData:  data,
		compressedFiles: make(map[string]compressedFile),
		directories:     make(map[string]map[string]bool),
	}
	return fs, fs.init()
}

func (fs *Fs) init() error {
	r := bytes.NewReader(fs.compressedData)
	compressor, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer compressor.Close()

	archive := tar.NewReader(compressor)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		path := header.Name
		pos := r.Size() - int64(r.Len())
		fs.compressedFiles[path] = compressedFile{
			start:  pos,
			header: header,
		}
		for _, segment := range dirsFromPath(path) {
			if fs.directories[segment] == nil {
				fs.directories[segment] = make(map[string]bool, 1)
			}
		}
	}

	// register dirs and files with parent directories
	for dir := range fs.directories {
		base := filepath.Base(dir)
		fs.directories[base][dir] = true
	}
	for path := range fs.compressedFiles {
		base := filepath.Base(path)
		fs.directories[base][path] = true
	}
	return nil
}

// dirsFromPath returns all directory segments in 'path'. Assumes 'path' is a raw header name from a tar.
func dirsFromPath(path string) []string {
	var dirs []string
	if strings.HasSuffix(path, pathSeparator) {
		// denotes a tar directory path, so clean it and add it before pop
		path = fsutil.NormalizePath(path)
		dirs = append(dirs, path)
	}
	path = fsutil.NormalizePath(path) // make absolute + clean
	path, _ = filepath.Split(path)    // pop normal files from the end
	for ; path != pathSeparator; path, _ = filepath.Split(path) {
		dirs = append(dirs, path)
	}
	return dirs
}

func (fs *Fs) Open(path string) (afero.File, error) {
	path = fsutil.NormalizePath(path)
	_, isDir := fs.directories[path]
	if isDir {
		return &file{
			compressedFile: compressedFile{
				header: &tar.Header{Name: path}, // just enough to look up more dir info in fs
			},
			fs:    fs,
			isDir: true,
		}, nil
	}
	f, isCompressed := fs.compressedFiles[path]
	if !isCompressed {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}

	return &file{
		compressedFile: f,
		fs:             fs,
		isDir:          f.header.FileInfo().IsDir(),
	}, nil
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag != os.O_RDONLY {
		return nil, syscall.EPERM
	}
	return fs.Open(name)
}

func (fs *Fs) Stat(path string) (os.FileInfo, error) {
	path = fsutil.NormalizePath(path)
	f, isCompressed := fs.compressedFiles[path]
	if isCompressed {
		return f.header.FileInfo(), nil
	}
	_, isDir := fs.directories[path]
	if !isDir {
		return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}

	return &genericDirInfo{name: filepath.Base(path)}, nil
}

func (fs *Fs) Name() string {
	return "tarfs.Fs"
}

func (fs *Fs) Create(name string) (afero.File, error)                      { return nil, syscall.EPERM }
func (fs *Fs) Mkdir(name string, perm os.FileMode) error                   { return syscall.EPERM }
func (fs *Fs) MkdirAll(path string, perm os.FileMode) error                { return syscall.EPERM }
func (fs *Fs) Remove(name string) error                                    { return syscall.EPERM }
func (fs *Fs) RemoveAll(path string) error                                 { return syscall.EPERM }
func (fs *Fs) Rename(oldname, newname string) error                        { return syscall.EPERM }
func (fs *Fs) Chmod(name string, mode os.FileMode) error                   { return syscall.EPERM }
func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error { return syscall.EPERM }