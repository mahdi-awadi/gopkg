// Package id provides short helpers for generating identifiers.
//
// Currently exposes UUIDv7 (time-ordered UUID, RFC 9562) built entirely
// from stdlib primitives. UUIDv7 is preferred over UUIDv4 for database
// primary keys because sequential generation keeps B-tree indexes dense.
//
// Zero third-party deps.
package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// UUIDv7 returns a new time-ordered UUID as a 36-character hex string
// with standard hyphenation: 8-4-4-4-12.
//
// Layout per RFC 9562:
//
//	48 bits: unix_ts_ms (big-endian)
//	 4 bits: version = 0b0111
//	12 bits: rand_a
//	 2 bits: variant = 0b10
//	62 bits: rand_b
//
// Safe for concurrent use.
func UUIDv7() string {
	return uuidToString(uuidv7Bytes())
}

// UUIDv7Raw returns the 16 raw bytes (no hyphens, no hex encoding).
// Useful when writing directly to a binary UUID column.
func UUIDv7Raw() [16]byte {
	return uuidv7Bytes()
}

var uuidv7Mu sync.Mutex

func uuidv7Bytes() [16]byte {
	var b [16]byte
	now := uint64(time.Now().UnixMilli())

	// Fill random bytes first.
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should never fail on supported platforms; panic is
		// fine for a library that's already inside the critical section.
		panic(fmt.Sprintf("id: crypto/rand: %v", err))
	}

	uuidv7Mu.Lock()
	defer uuidv7Mu.Unlock()

	// Insert timestamp into first 6 bytes (48 bits big-endian).
	b[0] = byte(now >> 40)
	b[1] = byte(now >> 32)
	b[2] = byte(now >> 24)
	b[3] = byte(now >> 16)
	b[4] = byte(now >> 8)
	b[5] = byte(now)

	// Version = 7 in top 4 bits of byte 6.
	b[6] = (b[6] & 0x0F) | 0x70
	// Variant = 10xx in top 2 bits of byte 8.
	b[8] = (b[8] & 0x3F) | 0x80
	return b
}

func uuidToString(b [16]byte) string {
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst)
}
