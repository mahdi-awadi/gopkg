package pipeline

import (
	"bytes"
	"errors"
	"testing"
)

func TestResolveBridge_SameFormatPassthrough(t *testing.T) {
	f := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	fn, err := resolveBridge(f, f)
	if err != nil {
		t.Fatalf("resolveBridge same→same: %v", err)
	}
	in := Frame{Data: []byte{1, 2, 3, 4}}
	out, err := fn(in)
	if err != nil || !bytes.Equal(out.Data, in.Data) {
		t.Errorf("passthrough mutated: in=%v out=%v err=%v", in.Data, out.Data, err)
	}
}

func TestResolveBridge_Mulaw8kToPCM16LE16k(t *testing.T) {
	src := AudioFormat{Encoding: EncodingMulaw, SampleRate: 8000, Channels: 1}
	dst := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	fn, err := resolveBridge(src, dst)
	if err != nil {
		t.Fatalf("resolveBridge: %v", err)
	}
	in := Frame{Data: make([]byte, 160)} // 20ms @ 8kHz mulaw
	for i := range in.Data {
		in.Data[i] = 0xFF
	}
	out, err := fn(in)
	if err != nil {
		t.Fatalf("bridge fn: %v", err)
	}
	if len(out.Data) != 640 {
		t.Errorf("expected 640 bytes (160 samples * 2 upsample * 2 bytes), got %d", len(out.Data))
	}
}

func TestResolveBridge_PCM16LE24kToMulaw8k(t *testing.T) {
	src := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 24000, Channels: 1}
	dst := AudioFormat{Encoding: EncodingMulaw, SampleRate: 8000, Channels: 1}
	fn, err := resolveBridge(src, dst)
	if err != nil {
		t.Fatalf("resolveBridge: %v", err)
	}
	in := Frame{Data: make([]byte, 960)} // 20ms @ 24kHz pcm16le = 480 samples * 2
	out, err := fn(in)
	if err != nil {
		t.Fatalf("bridge fn: %v", err)
	}
	if len(out.Data) != 160 {
		t.Errorf("expected 160 bytes mulaw (480/3 samples), got %d", len(out.Data))
	}
}

func TestResolveBridge_UnknownPair(t *testing.T) {
	src := AudioFormat{Encoding: EncodingMulaw, SampleRate: 16000, Channels: 1}
	dst := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 48000, Channels: 1}
	_, err := resolveBridge(src, dst)
	if !errors.Is(err, ErrFormatBridge) {
		t.Errorf("expected ErrFormatBridge, got %v", err)
	}
}

func TestResolveBridge_ChannelsMismatchRejected(t *testing.T) {
	src := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	dst := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 16000, Channels: 2}
	_, err := resolveBridge(src, dst)
	if !errors.Is(err, ErrFormatBridge) {
		t.Errorf("stereo→mono must be unsupported, got %v", err)
	}
}
