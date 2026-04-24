package pipeline

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/audio/codec"
)

// bridgeFn converts one Frame from the source format to the
// destination format declared at resolveBridge time.
type bridgeFn func(in Frame) (Frame, error)

// resolveBridge returns a converter function that transforms a Frame
// from src format to dst format, or ErrFormatBridge if the pair is
// not supported in this version.
//
// Supported pairs (v0.1.0):
//   - any → same (passthrough)
//   - mulaw@8kHz → pcm16le@16kHz (Twilio voice → Gemini-style STT)
//   - pcm16le@24kHz → mulaw@8kHz (Gemini-style TTS → Twilio voice)
//
// Add more pairs in future minor versions.
func resolveBridge(src, dst AudioFormat) (bridgeFn, error) {
	if src.Channels != dst.Channels || src.Channels != 1 {
		return nil, fmt.Errorf("%w: channels %d→%d (only mono=1 supported)",
			ErrFormatBridge, src.Channels, dst.Channels)
	}
	if src == dst {
		return bridgePassthrough, nil
	}
	switch {
	case src.Encoding == EncodingMulaw && src.SampleRate == 8000 &&
		dst.Encoding == EncodingPCM16LE && dst.SampleRate == 16000:
		return bridgeMulaw8kToPCM16LE16k, nil
	case src.Encoding == EncodingPCM16LE && src.SampleRate == 24000 &&
		dst.Encoding == EncodingMulaw && dst.SampleRate == 8000:
		return bridgePCM16LE24kToMulaw8k, nil
	}
	return nil, fmt.Errorf("%w: %s@%d → %s@%d",
		ErrFormatBridge,
		src.Encoding, src.SampleRate,
		dst.Encoding, dst.SampleRate)
}

func bridgePassthrough(in Frame) (Frame, error) {
	return in, nil
}

func bridgeMulaw8kToPCM16LE16k(in Frame) (Frame, error) {
	return Frame{
		Data:      codec.Mulaw8kToPCM16LE16k(in.Data),
		Timestamp: in.Timestamp,
	}, nil
}

func bridgePCM16LE24kToMulaw8k(in Frame) (Frame, error) {
	return Frame{
		Data:      codec.PCM16LE24kToMulaw8k(in.Data),
		Timestamp: in.Timestamp,
	}, nil
}
