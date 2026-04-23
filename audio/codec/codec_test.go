package codec

import (
	"math"
	"testing"
)

func TestMulawDecode_Silence(t *testing.T) {
	got := MulawToPCM16([]byte{0xFF})
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	if got[0] > 10 || got[0] < -10 {
		t.Errorf("silence byte 0xFF should decode to near-zero, got %d", got[0])
	}
}

func TestMulawDecode_MaxPositive(t *testing.T) {
	got := MulawToPCM16([]byte{0x80})
	if got[0] < 30000 {
		t.Errorf("mulaw 0x80 should decode to large positive, got %d", got[0])
	}
}

func TestMulawDecode_MaxNegative(t *testing.T) {
	got := MulawToPCM16([]byte{0x00})
	if got[0] > -30000 {
		t.Errorf("mulaw 0x00 should decode to large negative, got %d", got[0])
	}
}

func TestMulawRoundTrip(t *testing.T) {
	original := []int16{0, 1000, -1000, 8000, -8000, 32000, -32000}
	encoded := PCM16ToMulaw(original)
	if len(encoded) != len(original) {
		t.Fatalf("encoded len=%d, want %d", len(encoded), len(original))
	}
	decoded := MulawToPCM16(encoded)
	for i, orig := range original {
		diff := int(math.Abs(float64(orig - decoded[i])))
		// μ-law has logarithmic quantization — error grows with amplitude.
		maxErr := int(math.Abs(float64(orig))*0.05) + 100
		if diff > maxErr {
			t.Errorf("sample %d: original=%d, roundtrip=%d, diff=%d exceeds max=%d",
				i, orig, decoded[i], diff, maxErr)
		}
	}
}

func TestMulawEncode_Clipping(t *testing.T) {
	// Values above mulawClip should saturate, not wrap.
	got := PCM16ToMulaw([]int16{32767, -32768})
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	decoded := MulawToPCM16(got)
	if decoded[0] < 30000 {
		t.Errorf("+32767 should saturate to large positive μ-law, decoded=%d", decoded[0])
	}
	if decoded[1] > -30000 {
		t.Errorf("-32768 should saturate to large negative μ-law, decoded=%d", decoded[1])
	}
}

func TestUpsample8to16(t *testing.T) {
	got := Upsample8to16([]int16{100, 200, 300})
	if len(got) != 6 {
		t.Fatalf("len=%d, want 6", len(got))
	}
	if got[0] != 100 || got[2] != 200 || got[4] != 300 {
		t.Errorf("original samples not preserved: %v", got)
	}
	if got[1] != 150 {
		t.Errorf("got[1]=%d, want 150 (interpolated)", got[1])
	}
	if got[3] != 250 {
		t.Errorf("got[3]=%d, want 250 (interpolated)", got[3])
	}
	// Last sample gets duplicated, not interpolated past the end.
	if got[5] != 300 {
		t.Errorf("got[5]=%d, want 300 (trailing dup)", got[5])
	}
}

func TestUpsample8to16_Empty(t *testing.T) {
	if Upsample8to16(nil) != nil {
		t.Error("nil input should yield nil output")
	}
	if Upsample8to16([]int16{}) != nil {
		t.Error("empty input should yield nil output")
	}
}

func TestDownsample24to8(t *testing.T) {
	input := make([]int16, 24)
	for i := range input {
		input[i] = int16(i * 100)
	}
	got := Downsample24to8(input)
	if len(got) != 8 {
		t.Fatalf("len=%d, want 8", len(got))
	}
	for i, v := range got {
		want := int16(i * 3 * 100)
		if v != want {
			t.Errorf("got[%d]=%d, want %d", i, v, want)
		}
	}
}

func TestDownsample24to8_Partial(t *testing.T) {
	// 7 samples / 3 = 2 output samples; trailing samples dropped.
	got := Downsample24to8([]int16{0, 1, 2, 3, 4, 5, 6})
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
}

func TestMulaw8kToPCM16LE16k(t *testing.T) {
	// 160 bytes @ 8 kHz μ-law = 20 ms of telephony audio.
	// Output: 160 samples * 2 (upsampled to 16 kHz) * 2 bytes/sample = 640 bytes.
	input := make([]byte, 160)
	for i := range input {
		input[i] = 0xFF // silence
	}
	got := Mulaw8kToPCM16LE16k(input)
	if len(got) != 640 {
		t.Fatalf("len=%d, want 640", len(got))
	}
}

func TestPCM16LE24kToMulaw8k(t *testing.T) {
	// 480 samples @ 24 kHz = 20 ms. 480/3 = 160 samples @ 8 kHz.
	pcmBytes := make([]byte, 960) // 480 samples * 2 bytes/sample
	got := PCM16LE24kToMulaw8k(pcmBytes)
	if len(got) != 160 {
		t.Fatalf("len=%d, want 160", len(got))
	}
}

func TestPCM16LE24kToMulaw8k_Endianness(t *testing.T) {
	// Place a known LE-encoded int16 value of 1000 at sample 0 and verify
	// it survives the pipeline: at index 0 of the downsampled 8kHz stream,
	// the μ-law encoding should decode back to approximately +1000.
	pcm := make([]byte, 6) // 3 samples @ 24 kHz
	// sample 0 = 1000 (LE)
	pcm[0] = 1000 & 0xFF
	pcm[1] = (1000 >> 8) & 0xFF
	got := PCM16LE24kToMulaw8k(pcm)
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	decoded := MulawToPCM16(got)[0]
	diff := int(math.Abs(float64(1000 - decoded)))
	if diff > 100 {
		t.Errorf("roundtrip drifted: want ~1000, got %d (diff %d)", decoded, diff)
	}
}
