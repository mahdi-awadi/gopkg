// Package codec provides zero-dependency audio-codec primitives for
// realtime voice pipelines: ITU-T G.711 μ-law encode/decode, linear-PCM
// resampling, and endian-aware conversions between telephony-format
// byte streams and LLM-format PCM byte streams.
//
// Everything in this package is stateless, allocation-predictable, and
// safe for concurrent use. The zero value of every helper is fine.
//
// Typical use case: bridging an 8 kHz G.711 μ-law telephony stream
// (WhatsApp Calling, Twilio voice, SIP/RTP) with a 16 kHz or 24 kHz
// linear-PCM stream expected by modern speech-to-text / text-to-speech
// services (Google Gemini Live, OpenAI Realtime, Deepgram, etc.).
package codec

import "encoding/binary"

// G.711 μ-law decode table (ITU-T Recommendation G.711, August 1972).
// Maps each of the 256 μ-law code points to its signed 16-bit PCM value.
var mulawDecodeTable = [256]int16{
	-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
	-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
	-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
	-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
	-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
	-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
	-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
	-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
	-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
	-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
	-876, -844, -812, -780, -748, -716, -684, -652,
	-620, -588, -556, -524, -492, -460, -428, -396,
	-372, -356, -340, -324, -308, -292, -276, -260,
	-244, -228, -212, -196, -180, -164, -148, -132,
	-120, -112, -104, -96, -88, -80, -72, -64,
	-56, -48, -40, -32, -24, -16, -8, 0,
	32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
	23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
	15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
	11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
	7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
	5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
	3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
	2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
	1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
	1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
	876, 844, 812, 780, 748, 716, 684, 652,
	620, 588, 556, 524, 492, 460, 428, 396,
	372, 356, 340, 324, 308, 292, 276, 260,
	244, 228, 212, 196, 180, 164, 148, 132,
	120, 112, 104, 96, 88, 80, 72, 64,
	56, 48, 40, 32, 24, 16, 8, 0,
}

const (
	mulawBias = 0x84
	mulawClip = 32635
)

// MulawToPCM16 decodes a μ-law byte stream to signed 16-bit PCM samples.
// One μ-law byte produces exactly one PCM sample.
func MulawToPCM16(mulaw []byte) []int16 {
	pcm := make([]int16, len(mulaw))
	for i, b := range mulaw {
		pcm[i] = mulawDecodeTable[b]
	}
	return pcm
}

// PCM16ToMulaw encodes signed 16-bit PCM samples to a μ-law byte stream.
// One PCM sample produces exactly one μ-law byte.
func PCM16ToMulaw(pcm []int16) []byte {
	mulaw := make([]byte, len(pcm))
	for i, sample := range pcm {
		mulaw[i] = encodeMulaw(sample)
	}
	return mulaw
}

func encodeMulaw(sample int16) byte {
	// Compute |sample| in int (not int16) so -32768 doesn't overflow on
	// negation. Without this guard, -32768 silently wrapped to itself
	// and bypassed the saturation check below.
	sign := 0
	mag := int(sample)
	if mag < 0 {
		sign = 0x80
		mag = -mag
	}
	if mag > mulawClip {
		mag = mulawClip
	}
	mag += mulawBias
	exponent := 7
	expMask := 0x4000
	for i := 0; i < 8; i++ {
		if mag&expMask != 0 {
			break
		}
		exponent--
		expMask >>= 1
	}
	mantissa := (mag >> (uint(exponent) + 3)) & 0x0F
	return byte(^(sign | (exponent << 4) | mantissa))
}

// Upsample8to16 doubles the sample rate by linear interpolation.
// The output length is len(samples)*2. Intended for 8 kHz → 16 kHz PCM.
// The last sample is duplicated (no lookahead past the end of input).
func Upsample8to16(samples []int16) []int16 {
	if len(samples) == 0 {
		return nil
	}
	out := make([]int16, len(samples)*2)
	for i := 0; i < len(samples); i++ {
		out[i*2] = samples[i]
		if i < len(samples)-1 {
			out[i*2+1] = int16((int32(samples[i]) + int32(samples[i+1])) / 2)
		} else {
			out[i*2+1] = samples[i]
		}
	}
	return out
}

// Downsample24to8 reduces the sample rate by a factor of 3 via
// straight decimation (take every third sample). No anti-alias filter
// is applied — callers that care should low-pass-filter first.
// Intended for 24 kHz → 8 kHz PCM (common for LLM → telephony bridge).
func Downsample24to8(samples []int16) []int16 {
	outLen := len(samples) / 3
	out := make([]int16, outLen)
	for i := 0; i < outLen; i++ {
		out[i] = samples[i*3]
	}
	return out
}

// Mulaw8kToPCM16LE16k converts an 8 kHz μ-law byte stream (the
// telephony format used by Twilio, WhatsApp Calling, SIP) to a 16 kHz
// little-endian signed-16-bit PCM byte stream (the format many
// real-time STT/TTS services expect as input).
//
// Each input byte produces four output bytes.
func Mulaw8kToPCM16LE16k(mulaw []byte) []byte {
	pcm8k := MulawToPCM16(mulaw)
	pcm16k := Upsample8to16(pcm8k)
	out := make([]byte, len(pcm16k)*2)
	for i, s := range pcm16k {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

// PCM16LE24kToMulaw8k converts a 24 kHz little-endian signed-16-bit PCM
// byte stream (the common output of realtime LLM TTS services) to an
// 8 kHz μ-law byte stream (for delivery to telephony platforms).
//
// Each 6 input bytes (3 PCM samples at 24 kHz) produce 1 output byte
// (1 μ-law sample at 8 kHz). Odd trailing bytes are ignored.
func PCM16LE24kToMulaw8k(pcmData []byte) []byte {
	numSamples := len(pcmData) / 2
	pcm24k := make([]int16, numSamples)
	for i := 0; i < numSamples; i++ {
		pcm24k[i] = int16(binary.LittleEndian.Uint16(pcmData[i*2:]))
	}
	pcm8k := Downsample24to8(pcm24k)
	return PCM16ToMulaw(pcm8k)
}
