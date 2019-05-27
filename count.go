package routine

import (
	"io"
)

//CountReader count io reader
type CountReader struct {
	cnt int
	rd  io.Reader
}

//CountWriter count io writer
type CountWriter struct {
	cnt int
	wr  io.Writer
}

//NewCountReader  new
func NewCountReader(rd io.Reader) *CountReader {
	return &CountReader{rd: rd}
}

//Read impl
func (rc *CountReader) Read(p []byte) (int, error) {
	cnt, err := rc.rd.Read(p)
	rc.cnt += cnt
	return cnt, err
}

//Count get count
func (rc *CountReader) Count() int {
	return rc.cnt
}

//NewCountWriter new
func NewCountWriter(wr io.Writer) *CountWriter {
	return &CountWriter{wr: wr}
}

//Write impl
func (wc *CountWriter) Write(p []byte) (int, error) {
	cnt, err := wc.wr.Write(p)
	wc.cnt += cnt
	return cnt, err
}

//Count get count
func (wc *CountWriter) Count() int {
	return wc.cnt
}
