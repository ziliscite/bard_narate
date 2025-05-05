package domain

import "io"

type File struct {
	name  string
	types string
	body  io.Reader
}

func NewFile(name, mimetype string, body io.Reader) *File {
	var mimeTypes = map[string]string{
		"application/octet-stream": ".pth",
		"audio/wav":                ".wav",
		"audio/mpeg":               ".mp3",
		"text/plain":               ".txt",
	}

	if _, ok := mimeTypes[mimetype]; !ok {
		mimetype = "application/octet-stream"
	}

	return &File{
		name:  name,
		types: mimetype,
		body:  body,
	}
}

// Type returns the MIME type of the file
func (f *File) Type() string {
	return f.types
}

// Name returns the file name
func (f *File) Name() string {
	return f.name
}

// Body returns the file body as an io.Reader
func (f *File) Body() io.Reader {
	return f.body
}
