package copy

import (
	"io"
	"strings"
	"testing"
)

func TestCRC32UpdatingReader(t *testing.T) {
	tests := []struct {
		desc     string
		input    string
		startCRC int
		want     int
	}{
		{"Empty", "", 1234, 1234},
		{"Basic", "this is some data", 0, 1363046907},
		{"Basic, non-zero start", "this is some data", 1234, 59782035},
	}
	for _, tc := range tests {
		var r io.Reader = strings.NewReader(tc.input)
		crc := uint32(tc.startCRC)
		r = NewCRC32UpdatingReader(r, &crc)

		buf := make([]byte, 256)
		var err error
		for err == nil {
			_, err = r.Read(buf)
		}

		if crc != uint32(tc.want) {
			t.Errorf("%v: got crc %v, want %v", tc.desc, crc, tc.want)
		}
	}
}
