package datagenerator

import (
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"math/rand"
)

// BytesGenerator generates bytes based on a Distribution.
type BytesGenerator struct {
	d     helpers.Distribution
	bytes []byte
}

// NewBytesGenerator creates a BytesGenerator based on a distribution.
func NewBytesGenerator(d helpers.Distribution) *BytesGenerator {
	bytes := make([]byte, d.Max())
	rand.Read(bytes)
	return &BytesGenerator{d, bytes}
}

// GetBytes returns random bytes. The size of the returned bytes is based on
// generator distribution.
func (g BytesGenerator) GetBytes() []byte {
	size := g.d.GetNext()
	return g.bytes[:size]
}
