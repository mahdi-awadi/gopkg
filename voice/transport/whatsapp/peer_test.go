package whatsapptransport

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
	pionice "github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

// newTestOffererWithTrack creates a pion PeerConnection in the role of the
// remote caller (offerer), configured for fast ICE gathering (host candidates
// only, no STUN, no mDNS) so tests do not depend on external network access.
// The offerer negotiates Opus 48000/2, matching the real WhatsApp Cloud API.
func newTestOffererWithTrack(t *testing.T) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticSample, string) {
	t.Helper()

	se := webrtc.SettingEngine{}
	se.SetSTUNGatherTimeout(time.Millisecond)
	se.SetICEMulticastDNSMode(pionice.MulticastDNSModeDisabled)

	me := &webrtc.MediaEngine{}
	if err := me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		t.Fatalf("offerer RegisterCodec: %v", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(se))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("offerer NewPeerConnection: %v", err)
	}

	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "offerer-voice",
	)
	if err != nil {
		t.Fatalf("offerer NewTrackLocalStaticSample: %v", err)
	}
	if _, err := pc.AddTrack(track); err != nil {
		t.Fatalf("offerer AddTrack: %v", err)
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatalf("offerer CreateOffer: %v", err)
	}

	gather := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		t.Fatalf("offerer SetLocalDescription: %v", err)
	}
	<-gather

	return pc, track, pc.LocalDescription().SDP
}

// TestNewPeer_BuildsAnswerSDP verifies that NewPeer accepts a synthetic Opus
// offer and returns a non-empty answer containing m=audio.
func TestNewPeer_BuildsAnswerSDP(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer error: %v", err)
	}
	defer peer.PC.Close()

	if peer.AnswerSDP == "" {
		t.Fatal("AnswerSDP is empty")
	}
	if !strings.Contains(peer.AnswerSDP, "m=audio") {
		t.Errorf("AnswerSDP does not contain 'm=audio': %q", peer.AnswerSDP)
	}
}

// TestNewPeer_RealWhatsAppOffer feeds the real WhatsApp Cloud API SDP offer
// (testdata/offer-whatsapp-opus.sdp) to NewPeer and asserts that the answer
// negotiates Opus and not PCMU.
func TestNewPeer_RealWhatsAppOffer(t *testing.T) {
	sdpBytes, err := os.ReadFile("testdata/offer-whatsapp-opus.sdp")
	if err != nil {
		t.Fatalf("reading testdata/offer-whatsapp-opus.sdp: %v", err)
	}
	offerSDP := string(sdpBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer with real WhatsApp offer: %v", err)
	}
	defer peer.PC.Close()

	if !strings.Contains(peer.AnswerSDP, "m=audio") {
		t.Errorf("AnswerSDP missing m=audio: %q", peer.AnswerSDP)
	}
	if !strings.Contains(strings.ToLower(peer.AnswerSDP), "opus/48000") {
		t.Errorf("AnswerSDP does not contain opus/48000: %q", peer.AnswerSDP)
	}
	if strings.Contains(strings.ToUpper(peer.AnswerSDP), "PCMU") {
		t.Errorf("AnswerSDP contains PCMU — codec mismatch: %q", peer.AnswerSDP)
	}
}

// TestNewPeer_Formats verifies InboundFormat and OutboundFormat after a
// successful negotiation using the synthetic Opus offerer.
func TestNewPeer_Formats(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer error: %v", err)
	}
	defer peer.PC.Close()

	tp, err := New(peer)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer tp.Close()

	in := tp.InboundFormat()
	out := tp.OutboundFormat()

	if in.Encoding != pipeline.EncodingPCM16LE || in.SampleRate != 16000 || in.Channels != 1 {
		t.Errorf("InboundFormat = %+v, want pcm16le@16k mono", in)
	}
	if out.Encoding != pipeline.EncodingPCM16LE || out.SampleRate != 24000 || out.Channels != 1 {
		t.Errorf("OutboundFormat = %+v, want pcm16le@24k mono", out)
	}
}
