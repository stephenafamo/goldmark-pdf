package pdf

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

func getFileMime(f http.File) string {
	if typed, ok := f.(interface{ MimeType() string }); ok {
		return typed.MimeType()
	}

	info, err := f.Stat()
	if err != nil {
		return ""
	}

	if ext := filepath.Ext(info.Name()); ext != "" {
		return mime.TypeByExtension(ext)
	}

	sniff := make([]byte, 512)
	if _, err := io.ReadFull(f, sniff); err != nil {
		return ""
	}

	_, err = f.Seek(0, io.SeekStart)
	return mime.TypeByExtension(http.DetectContentType(sniff))
}

func getImageMime(m string) string {
	if strings.HasPrefix(m, "image/") {
		return strings.TrimPrefix(m, "image/")
	}
	return ""
}

type multiFs struct {
	wrapped []http.FileSystem
}

func (f multiFs) Open(name string) (http.File, error) {
	for k := range f.wrapped {
		if file, err := f.wrapped[k].Open(name); err == nil {
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}

func mergeFs(f ...http.FileSystem) http.FileSystem {
	return multiFs{wrapped: f}
}

type inlineFs struct{}

func (f inlineFs) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, "data:image/") {
		return nil, fs.ErrNotExist
	}

	dataDiv := strings.Index(name, ":")
	mimeDiv := strings.Index(name, ";")
	encDiv := strings.Index(name, ",")

	if dataDiv < 0 || mimeDiv < 0 || encDiv < 0 {
		return nil, fs.ErrInvalid
	}

	mimeType := name[dataDiv+1 : mimeDiv]
	encoding := name[mimeDiv+1 : encDiv]
	ext, _ := mime.ExtensionsByType(mimeType)

	if encoding != "base64" {
		return nil, fs.ErrInvalid
	}

	if len(ext) == 0 {
		return nil, fs.ErrInvalid
	}

	data, err := base64.StdEncoding.DecodeString(name[encDiv+1:])
	if err != nil {
		return nil, fs.ErrInvalid
	}

	h := sha256.New()
	h.Write(data)
	fName := hex.EncodeToString(h.Sum(nil)) + ext[0]

	return inlineFile{
		data: bytes.NewReader(data),
		mime: mimeType,
		size: int64(len(data)),
		name: fName,
	}, nil
}

type inlineFile struct {
	data *bytes.Reader
	mime string
	size int64
	name string
}

func (f inlineFile) MimeType() string {
	return f.mime
}

func (f inlineFile) Stat() (fs.FileInfo, error) {
	i := inlineInfo{
		s: f.size,
		n: f.name,
	}
	return i, nil
}

func (f inlineFile) Read(p []byte) (int, error) {
	return f.data.Read(p)
}

func (f inlineFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}

func (f inlineFile) Close() error {
	return nil
}

func (f inlineFile) Seek(offset int64, whence int) (absoluteOffset int64, err error) {
	return f.data.Seek(offset, whence)
}

type inlineInfo struct {
	s int64
	n string
}

func (i inlineInfo) Name() string {
	return i.n
}

func (i inlineInfo) Size() int64 {
	return i.s
}

func (i inlineInfo) Mode() fs.FileMode {
	return fs.ModeIrregular
}

func (i inlineInfo) ModTime() time.Time {
	return time.Time{}
}

func (i inlineInfo) IsDir() bool {
	return false
}

func (i inlineInfo) Sys() interface{} {
	return nil
}

// ----------------------

type webFs struct{}

func (f webFs) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, "http://") && !strings.HasPrefix(name, "https://") {
		return nil, fs.ErrNotExist
	}

	if _, err := url.Parse(name); err != nil {
		return nil, fs.ErrInvalid
	}

	res, err := http.Get(name)
	if err != nil {
		return nil, err
	}

	return &webFile{r: res}, nil
}

type webFile struct {
	r         *http.Response
	data      *bytes.Reader // @todo configurable so that we can use tmp directory instead of just memory
	closed    bool
	closedErr error
}

func (f webFile) MimeType() string {
	return f.r.Header.Get("Content-Type")
}

func (f webFile) Stat() (fs.FileInfo, error) {
	// if this errors, we get empty time struct which is desired
	mod, _ := time.Parse(time.RFC1123, f.r.Header.Get("Last-Modified"))

	fileName := strings.TrimPrefix(f.r.Request.URL.Path, "/")
	if fileName == "" {
		_, params, err := mime.ParseMediaType(f.r.Header.Get("Content-Disposition"))
		if err == nil {
			fileName = params["filename"]
		}
	}

	if filepath.Ext(fileName) == "" {
		mt, _, _ := mime.ParseMediaType(f.r.Header.Get("Content-Type"))
		if spl := strings.Split(mt, "/"); len(spl) > 0 {
			if fileName == "" {
				fileName = spl[0]
			}
			fileName += "." + spl[len(spl)-1]
		}
	}

	fileName = filepath.Base(fileName)

	i := webInfo{
		s: f.r.ContentLength,
		t: mod,
		n: fileName,
	}

	return i, nil
}

func (f *webFile) setData() error {
	if f.data != nil {
		return nil
	}

	data, err := io.ReadAll(f.r.Body)
	f.closedErr = f.r.Body.Close()
	f.closed = true
	if err != nil {
		f.closedErr = err // this will have priority
		return err
	}

	f.data = bytes.NewReader(data)
	return nil
}

func (f *webFile) Read(p []byte) (int, error) {
	// if there is no data and we're closed, it means there was previously some kind of error
	if f.data == nil && f.closed {
		return 0, f.closedErr
	}

	if err := f.setData(); err != nil {
		return 0, err
	}

	return f.data.Read(p)
}

func (f *webFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}

func (f *webFile) Close() error {
	if f.closed == false {
		f.closedErr = f.r.Body.Close()
		f.closed = true
	}
	return f.closedErr
}

func (f *webFile) Seek(offset int64, whence int) (absoluteOffset int64, err error) {
	// if there is no data and we're closed, it means there was previously some kind of error
	if f.data == nil && f.closed {
		return 0, f.closedErr
	}

	if err := f.setData(); err != nil {
		return 0, err
	}

	return f.data.Seek(offset, whence)
}

type webInfo struct {
	s int64
	t time.Time
	n string
}

func (i webInfo) Name() string {
	return i.n
}

func (i webInfo) Size() int64 {
	return i.s
}

func (i webInfo) Mode() fs.FileMode {
	return fs.ModeIrregular
}

func (i webInfo) ModTime() time.Time {
	return i.t
}

func (i webInfo) IsDir() bool {
	return false
}

func (i webInfo) Sys() interface{} {
	return nil // we can pass the response if needed
}
