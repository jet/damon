package plugin

type devNull int

// DevNull is a no-op io device
// It implements:
// - io.Reader
// - io.Writer
// - io.Closer (implicitly: io.ReadCloser & io.WriteCloser)
// - io.Seeker (implicitly: io.ReadSeeker & io.WriteSeeker)
// - io.ReaderAt
// - io.WriterAt
// - io.StringWriter
const DevNull = devNull(0)

// Read implements a no-op io.Reader
func (n devNull) Read(p []byte) (int, error) {
	return 0, nil
}

// ReadAt implements a no-op io.ReaderAt
func (n devNull) ReadAt(p []byte, off int64) (int, error) {
	return 0, nil
}

//Write implements a discarding io.Writer
func (n devNull) Write(p []byte) (int, error) {
	return len(p), nil
}

//WriteAt implements a discarding io.WriterAt
func (n devNull) WriteAt(p []byte, off int64) (int, error) {
	return len(p), nil
}

// WriteString implements a discarding io.StringWriter
func (n devNull) WriteString(s string) (int, error) {
	return len(s), nil
}

// Close implements a no-op io.Closer
func (n devNull) Close() error {
	return nil
}

// Seek implements a no-op io.Seeker
func (n devNull) Seek(offset, whence int) (int64, error) {
	return 0, nil
}
