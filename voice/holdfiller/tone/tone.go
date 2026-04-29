// Package tone provides a mu-law hold-tone filler for voice pipelines.
package tone

import (
	"context"
	"math"
	"time"

	"github.com/mahdi-awadi/gopkg/audio/codec"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Options configures the generated tone pattern.
type Options struct {
	SampleRate      int
	FrameDuration   time.Duration
	PatternDuration time.Duration
	ToneDuration    time.Duration
	FrequencyHz     float64
	Amplitude       float64
}

// Filler loops a pre-generated mu-law tone pattern.
type Filler struct {
	chunks [][]byte
}

// New returns a filler with a 2 second pattern: 400ms soft tone followed by
// silence, split into 20ms mu-law frames.
func New() *Filler {
	return NewWithOptions(Options{})
}

// NewWithOptions returns a filler using opts, applying defaults for zero
// values. The output format is always mu-law@8k-compatible byte frames when
// using defaults.
func NewWithOptions(opts Options) *Filler {
	opts = defaults(opts)
	return &Filler{chunks: generate(opts)}
}

func defaults(opts Options) Options {
	if opts.SampleRate == 0 {
		opts.SampleRate = 8000
	}
	if opts.FrameDuration == 0 {
		opts.FrameDuration = 20 * time.Millisecond
	}
	if opts.PatternDuration == 0 {
		opts.PatternDuration = 2 * time.Second
	}
	if opts.ToneDuration == 0 {
		opts.ToneDuration = 400 * time.Millisecond
	}
	if opts.FrequencyHz == 0 {
		opts.FrequencyHz = 600
	}
	if opts.Amplitude == 0 {
		opts.Amplitude = 2000
	}
	if opts.ToneDuration > opts.PatternDuration {
		opts.ToneDuration = opts.PatternDuration
	}
	return opts
}

// Frames loops generated chunks until ctx is cancelled.
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

func generate(opts Options) [][]byte {
	totalSamples := int(float64(opts.SampleRate) * opts.PatternDuration.Seconds())
	toneSamples := int(float64(opts.SampleRate) * opts.ToneDuration.Seconds())
	frameSamples := int(float64(opts.SampleRate) * opts.FrameDuration.Seconds())
	if totalSamples <= 0 {
		totalSamples = opts.SampleRate
	}
	if frameSamples <= 0 {
		frameSamples = opts.SampleRate / 50
	}

	pcm := make([]int16, totalSamples)
	fadeSamples := int(0.05 * float64(opts.SampleRate))
	for i := 0; i < toneSamples && i < totalSamples; i++ {
		t := float64(i) / float64(opts.SampleRate)
		envelope := 1.0
		if fadeSamples > 0 && i < fadeSamples {
			envelope = float64(i) / float64(fadeSamples)
		} else if fadeSamples > 0 && i > toneSamples-fadeSamples {
			envelope = float64(toneSamples-i) / float64(fadeSamples)
			if envelope < 0 {
				envelope = 0
			}
		}
		pcm[i] = int16(opts.Amplitude * envelope * math.Sin(2*math.Pi*opts.FrequencyHz*t))
	}

	mulaw := codec.PCM16ToMulaw(pcm)
	chunks := make([][]byte, 0, (len(mulaw)+frameSamples-1)/frameSamples)
	for i := 0; i < len(mulaw); i += frameSamples {
		end := i + frameSamples
		if end > len(mulaw) {
			end = len(mulaw)
		}
		chunk := make([]byte, end-i)
		copy(chunk, mulaw[i:end])
		chunks = append(chunks, chunk)
	}
	return chunks
}

var _ pipeline.HoldFiller = (*Filler)(nil)
