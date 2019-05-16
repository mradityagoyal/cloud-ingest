package copy

import (
	"hash/crc32"
	"io"
)

var (
	CRC32CTable = crc32.MakeTable(crc32.Castagnoli)
)

// CRC32UpdatingReader is an io.Reader that wraps another io.Reader and a
// starting crc32c. This reader updates the crc32c as bytes are read.
type CRC32UpdatingReader struct {
	reader io.Reader
	curCRC *uint32
}

// NewCRC32UpdatingReader returns a CRC32UpdatingReader. 'currentCRC' is the
// starting crc32 value for the reader, and will be updated as bytes are read.
func NewCRC32UpdatingReader(r io.Reader, currentCRC *uint32) io.Reader {
	return &CRC32UpdatingReader{reader: r, curCRC: currentCRC}
}

// Read implements the io.Reader interface.
func (cr *CRC32UpdatingReader) Read(buf []byte) (n int, err error) {
	if n, err = cr.reader.Read(buf); err != nil {
		return 0, err
	}
	*cr.curCRC = crc32.Update(*cr.curCRC, CRC32CTable, buf[:n])
	return n, nil
}
