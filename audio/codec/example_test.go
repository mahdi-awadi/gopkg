package codec_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/audio/codec"
)

func ExampleMulaw8kToPCM16LE16k() {
	// A 20 ms packet of 8 kHz μ-law silence is 160 bytes.
	telephonyFrame := make([]byte, 160)
	for i := range telephonyFrame {
		telephonyFrame[i] = 0xFF // silence in μ-law
	}

	llmFrame := codec.Mulaw8kToPCM16LE16k(telephonyFrame)
	fmt.Println(len(llmFrame), "bytes of 16 kHz PCM16LE")
	// Output: 640 bytes of 16 kHz PCM16LE
}

func ExamplePCM16LE24kToMulaw8k() {
	// A 20 ms packet of 24 kHz PCM16LE is 480 samples * 2 = 960 bytes.
	llmFrame := make([]byte, 960)

	telephonyFrame := codec.PCM16LE24kToMulaw8k(llmFrame)
	fmt.Println(len(telephonyFrame), "bytes of 8 kHz μ-law")
	// Output: 160 bytes of 8 kHz μ-law
}
