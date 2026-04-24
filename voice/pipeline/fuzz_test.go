package pipeline

import "testing"

func FuzzCodecBridge_Mulaw8kToPCM16LE16k(f *testing.F) {
	src := AudioFormat{Encoding: EncodingMulaw, SampleRate: 8000, Channels: 1}
	dst := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	bridge, err := resolveBridge(src, dst)
	if err != nil {
		f.Fatalf("resolveBridge: %v", err)
	}
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF, 0x00, 0x80})
	f.Fuzz(func(t *testing.T, in []byte) {
		out, err := bridge(Frame{Data: in})
		if err != nil {
			t.Fatal(err)
		}
		if len(in) == 0 && len(out.Data) != 0 {
			t.Errorf("empty input should produce empty output")
		}
		if len(in) > 0 && len(out.Data) != len(in)*4 {
			t.Errorf("output size=%d, want %d (2× upsample, 2 bytes/sample)",
				len(out.Data), len(in)*4)
		}
	})
}
