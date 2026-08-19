package whatsapptransport

import (
	"context"
	"strings"
	"testing"
	"time"

	pionice "github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

// newTestOffererWithTrack creates a pion PeerConnection in the role of the
// remote caller (offerer), configured for fast ICE gathering (host candidates
// only, no STUN, no mDNS) so tests do not depend on external network access.
func newTestOffererWithTrack(t *testing.T) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticSample, string) {
	t.Helper()

	se := webrtc.SettingEngine{}
	se.SetSTUNGatherTimeout(time.Millisecond)
	se.SetICEMulticastDNSMode(pionice.MulticastDNSModeDisabled)

	me := &webrtc.MediaEngine{}
	if err := me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMU,
			ClockRate: 8000,
			Channels:  1,
		},
		PayloadType: 0,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		t.Fatalf("offerer RegisterCodec: %v", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(se))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("offerer NewPeerConnection: %v", err)
	}

	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000, Channels: 1},
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

	tp := New(peer)
	defer tp.Close()

	in := tp.InboundFormat()
	out := tp.OutboundFormat()

	if in.Encoding != "mulaw" || in.SampleRate != 8000 || in.Channels != 1 {
		t.Errorf("InboundFormat = %+v, want mulaw@8k mono", in)
	}
	if out.Encoding != "mulaw" || out.SampleRate != 8000 || out.Channels != 1 {
		t.Errorf("OutboundFormat = %+v, want mulaw@8k mono", out)
	}
}
