// Package twilio ships a pipeline.HoldFiller that loops the
// pre-generated hold tone (soft 600Hz beep + silence, mulaw@8k, 20ms
// per frame) until the caller's context is cancelled.
//
// The tone pattern matches the one previously inlined in
// services/communication-service/internal/voice/holdtone.go. Frames
// are raw mulaw bytes (160 bytes each); the transport layer handles
// base64 encoding when writing to Twilio.
package twilio

import (
	"context"
	"math"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Filler is a pipeline.HoldFiller that loops the hold tone.
type Filler struct {
	chunks [][]byte
}

// New returns a Filler pre-loaded with the 2-second hold pattern
// (400ms soft beep + 1600ms silence) split into 20ms mulaw chunks.
func New() *Filler {
	return &Filler{chunks: generateHoldToneChunks()}
}

// Frames loops the hold-tone chunks, emitting one Frame per iteration
// and exiting cleanly when ctx is cancelled.
func (f *Filler) Frames(ctx context.Context) <-chan pipeline.Frame {
	ch := make(chan pipeline.Frame)
	go func() {
		defer close(ch)
		for {
			for _, chunk := range f.chunks {
				select {
				case ch <- pipeline.Frame{Data: chunk, Timestamp: time.Now()}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}

// generateHoldToneChunks builds the 2-second hold pattern as 20ms
// mulaw@8k chunks. Called once in New; the result is cached per
// Filler instance.
func generateHoldToneChunks() [][]byte {
	const sampleRate = 8000
	const totalSamples = sampleRate * 2 // 2 seconds
	const toneFreq = 600.0
	const amplitude = 2000.0

	pcm := make([]int16, totalSamples)
	toneDuration := int(0.4 * float64(sampleRate)) // 400ms
	fadeLen := 0.05 * float64(sampleRate)          // 50ms fade

	for i := 0; i < toneDuration && i < totalSamples; i++ {
		t := float64(i) / float64(sampleRate)
		envelope := 1.0
		if i < int(fadeLen) {
			envelope = float64(i) / fadeLen
		} else if i > toneDuration-int(fadeLen) {
			envelope = float64(toneDuration-i) / fadeLen
		}
		pcm[i] = int16(amplitude * envelope * math.Sin(2*math.Pi*toneFreq*t))
	}

	mulaw := pcm16ToMulaw(pcm)
	const chunkSize = 160
	var chunks [][]byte
	for i := 0; i < len(mulaw); i += chunkSize {
		end := i + chunkSize
		if end > len(mulaw) {
			end = len(mulaw)
		}
		chunk := make([]byte, end-i)
		copy(chunk, mulaw[i:end])
		chunks = append(chunks, chunk)
	}
	return chunks
}

// pcm16ToMulaw converts PCM16 LE samples to mulaw bytes. Matches the
// pre-migration implementation in services/communication-service/
// internal/voice/codec.go (PCM16ToMulaw) — kept inline here so this
// package has no cross-package codec coupling.
func pcm16ToMulaw(pcm []int16) []byte {
	out := make([]byte, len(pcm))
	for i, s := range pcm {
		out[i] = linearToMulaw(int32(s))
	}
	return out
}

// linearToMulaw is the standard mulaw encoding (ITU-T G.711). Byte-for-
// byte port of the helper in the pre-migration voice/codec.go.
func linearToMulaw(sample int32) byte {
	const (
		muLawMax  = 0x1FFF
		muLawBias = 0x84
	)
	sign := byte(0)
	if sample < 0 {
		sample = -sample
		sign = 0x80
	}
	if sample > muLawMax {
		sample = muLawMax
	}
	sample += muLawBias

	exponent := byte(7)
	for expMask := int32(0x4000); (sample & expMask) == 0 && exponent > 0; expMask >>= 1 {
		exponent--
	}
	mantissa := byte((sample >> (exponent + 3)) & 0x0F)
	return ^(sign | (exponent << 4) | mantissa)
}
