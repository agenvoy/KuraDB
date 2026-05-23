package filesystem

import (
	"bytes"
	"io"
	"os"
	"unicode/utf8"
)

const sniffSize = 8192

var (
	excludedNames = map[string]struct{}{
		".DS_Store": {},
	}

	skippedExts = map[string]struct{}{
		".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".bmp": {},
		".heic": {}, ".heif": {}, ".svg": {}, ".ico": {}, ".tiff": {}, ".tif": {}, ".avif": {},
		".mp4": {}, ".mov": {}, ".avi": {}, ".mkv": {}, ".webm": {}, ".flv": {}, ".wmv": {},
		".m4v": {}, ".mpg": {}, ".mpeg": {}, ".3gp": {},
		".mp3": {}, ".wav": {}, ".flac": {}, ".ogg": {}, ".m4a": {}, ".aac": {}, ".wma": {}, ".opus": {},
		".zip": {}, ".tar": {}, ".gz": {}, ".bz2": {}, ".xz": {}, ".7z": {}, ".rar": {}, ".tgz": {},
		".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".o": {}, ".a": {}, ".bin": {},
		".iso": {}, ".dmg": {}, ".pkg": {}, ".deb": {}, ".rpm": {}, ".apk": {},
		".class": {}, ".jar": {}, ".pyc": {}, ".pyo": {}, ".wasm": {},
		".db": {}, ".sqlite": {}, ".sqlite3": {},
		".psd": {}, ".ai": {}, ".sketch": {}, ".fig": {},
		".ttf": {}, ".otf": {}, ".woff": {}, ".woff2": {}, ".eot": {},
	}
)

func shouldSkipByName(name string) bool {
	_, ok := excludedNames[name]
	return ok
}

func shouldSkipByExt(ext string) bool {
	_, ok := skippedExts[ext]
	return ok
}

func looksLikeText(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, sniffSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false
	}
	if n == 0 {
		return false
	}
	sample := buf[:n]
	if bytes.IndexByte(sample, 0) >= 0 {
		return false
	}
	return utf8.Valid(sample)
}
