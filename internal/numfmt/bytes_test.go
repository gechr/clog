package numfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	kByte = 1000
	mByte = kByte * 1000
	gByte = mByte * 1000
	tByte = gByte * 1000
	pByte = tByte * 1000
	eByte = pByte * 1000

	kiByte = 1024
	miByte = kiByte * 1024
	giByte = miByte * 1024
	tiByte = giByte * 1024
	piByte = tiByte * 1024
	eiByte = piByte * 1024
)

func TestBytes(t *testing.T) {
	tests := []struct {
		in     uint64
		digits int
		out    string
	}{
		// Below threshold: raw bytes.
		{0, 2, "0 B"},
		{1, 2, "1 B"},
		{9, 2, "9 B"},
		// Below MB: decimals always suppressed.
		{803, 2, "803 B"},
		{999, 2, "999 B"},
		{1024, 2, "1 kB"},
		{9999, 2, "10 kB"},
		{mByte - 1, 2, "1000 kB"},
		// MB+: significant digits, trailing zeros stripped.
		{mByte, 2, "1 MB"},
		{gByte, 2, "1 GB"},
		{tByte, 2, "1 TB"},
		{pByte, 2, "1 PB"},
		{eByte, 2, "1 EB"},
		{uint64(5.5 * float64(gByte)), 2, "5.5 GB"},
		{82854982, 2, "83 MB"},
		// digits=3: more precision.
		{82854982, 3, "82.9 MB"},
		{gByte, 3, "1 GB"},
		{uint64(1.5 * float64(gByte)), 3, "1.5 GB"},
		{uint64(100 * float64(mByte)), 3, "100 MB"},
		// digits=1: minimal.
		{gByte, 1, "1 GB"},
		{82854982, 1, "83 MB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.out, Bytes(tt.in, tt.digits),
			"Bytes(%d, %d)", tt.in, tt.digits)
	}
}

func TestIBytes(t *testing.T) {
	tests := []struct {
		in     uint64
		digits int
		out    string
	}{
		{0, 2, "0 B"},
		{1, 2, "1 B"},
		{803, 2, "803 B"},
		{1023, 2, "1023 B"},
		// Below MiB: decimals always suppressed.
		{1024, 2, "1 KiB"},
		{miByte - 1, 2, "1024 KiB"},
		// MiB+: significant digits, trailing zeros stripped.
		{miByte, 2, "1 MiB"},
		{giByte, 2, "1 GiB"},
		{tiByte, 2, "1 TiB"},
		{piByte, 2, "1 PiB"},
		{eiByte, 2, "1 EiB"},
		{uint64(5.5 * float64(giByte)), 2, "5.5 GiB"},
		{82854982, 2, "79 MiB"},
		// digits=3: more precision.
		{82854982, 3, "79 MiB"},
		{uint64(5.5 * float64(giByte)), 3, "5.5 GiB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.out, IBytes(tt.in, tt.digits),
			"IBytes(%d, %d)", tt.in, tt.digits)
	}
}

func TestBytesWidth(t *testing.T) {
	// Width must be >= any stripped value in the same unit range.
	w := BytesWidth(uint64(100*mByte), 3) // total = 100 MB
	assert.GreaterOrEqual(t, w, len(Bytes(82854982, 3)))
	assert.GreaterOrEqual(t, w, len(Bytes(uint64(100*mByte), 3)))
	assert.GreaterOrEqual(t, w, len(Bytes(uint64(50*mByte), 3)))
}
