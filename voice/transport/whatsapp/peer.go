package whatsapptransport

import (
	"context"
	"net"
	"time"

	pionice "github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

const defaultSTUN = "stun:stun.l.google.com:19302"

// Config holds optional configuration for the WebRTC PeerConnection.
type Config struct {
	// ICEServers overrides the default STUN server. When empty AND PublicIP is
	// unset, stun.l.google.com:19302 is used.
	ICEServers []webrtc.ICEServer

	// PublicIP, when set, makes pion advertise a 1:1-NAT host candidate at this
	// IP instead of relying on STUN. Required when the process runs inside a
	// Docker bridge network: the container's own IP (e.g. 172.x/192.168.x) is
	// unreachable by the remote peer, and the STUN-discovered srflx port is not
	// published, so media (DTLS/SRTP) never arrives and the call answers but is
	// silent. Pair with UDPPortMin/Max + a published port range.
	PublicIP string

	// UDPPortMin/UDPPortMax pin ICE/media to a fixed UDP port range so those
	// ports can be published from the container. Zero means "any ephemeral
	// port" (only usable with host networking or PublicIP unset).
	UDPPortMin uint16
	UDPPortMax uint16

	// settingEngine is an optional SettingEngine injected by tests to control
	// ICE timeouts and other low-level parameters. Not part of the public API.
	settingEngine *webrtc.SettingEngine
}

// Peer wraps a negotiated pion PeerConnection ready to exchange Opus audio.
// Callers obtain one via NewPeer, pass it to New to get a pipeline.Transport,
// then hand peer.AnswerSDP back to the WhatsApp Cloud API.
type Peer struct {
	// PC is the underlying WebRTC PeerConnection. Exposed so callers can
	// register additional event handlers or inspect ICE state.
	PC *webrtc.PeerConnection

	// AnswerSDP is the complete SDP answer (with gathered ICE candidates).
	// Return this to the WhatsApp Cloud API to complete the call setup.
	AnswerSDP string

	localTrack  *webrtc.TrackLocalStaticSample
	remoteTrack chan *webrtc.TrackRemote // buffered(1); written by OnTrack
}

// NewPeer builds a PeerConnection from the caller's SDP offer, negotiates
// Opus 48000/2 only, and gathers ICE candidates before returning. The returned
// Peer.AnswerSDP is the complete answer to send back to the WhatsApp caller.
func NewPeer(ctx context.Context, offerSDP string, cfg Config) (*Peer, error) {
	// STUN is only needed to discover a public (srflx) candidate. When PublicIP
	// is configured we advertise a directly-reachable 1:1-NAT host candidate
	// instead, so STUN is skipped.
	iceServers := cfg.ICEServers
	if len(iceServers) == 0 && cfg.PublicIP == "" {
		iceServers = []webrtc.ICEServer{{URLs: []string{defaultSTUN}}}
	}

	// Register Opus only — matches the real WhatsApp SDP offer (a=rtpmap:111 opus/48000/2).
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
		return nil, err
	}

	// SettingEngine: tests inject their own; in production a configured PublicIP
	// makes pion emit a 1:1-NAT host candidate at that IP and (when a port range
	// is given) bind media to a fixed, publishable UDP port range. Without this,
	// inside a Docker bridge network the only candidates are the unreachable
	// container IP and an unpublished STUN srflx port, so the call answers but
	// carries no audio.
	var se webrtc.SettingEngine
	haveSE := false
	switch {
	case cfg.settingEngine != nil:
		se = *cfg.settingEngine
		haveSE = true
	case cfg.PublicIP != "":
		se.SetNAT1To1IPs([]string{cfg.PublicIP}, webrtc.ICECandidateTypeHost)
		if cfg.UDPPortMin > 0 && cfg.UDPPortMax >= cfg.UDPPortMin {
			if err := se.SetEphemeralUDPPortRange(cfg.UDPPortMin, cfg.UDPPortMax); err != nil {
				return nil, err
			}
		}
		haveSE = true
	}

	apiOpts := []func(*webrtc.API){webrtc.WithMediaEngine(me)}
	if haveSE {
		apiOpts = append(apiOpts, webrtc.WithSettingEngine(se))
	}

	api := webrtc.NewAPI(apiOpts...)
	pc, err := api.NewPeerConnection(webrtc.Configuration{ICEServers: iceServers})
	if err != nil {
		return nil, err
	}

	// Local sendonly track for outbound AI audio (Opus 48k stereo header, pion
	// timestamps on the 48k clock via WriteSample Duration).
	localTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "adminli-voice",
	)
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if _, err := pc.AddTrack(localTrack); err != nil {
		_ = pc.Close()
		return nil, err
	}

	// remoteTrack receives the caller's inbound track from OnTrack.
	remoteTrackCh := make(chan *webrtc.TrackRemote, 1)
	pc.OnTrack(func(tr *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		select {
		case remoteTrackCh <- tr:
		default: // already delivered; ignore additional tracks
		}
	})

	// Apply offer → create answer → gather ICE candidates.
	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	}); err != nil {
		_ = pc.Close()
		return nil, err
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, err
	}

	// Register gathering-complete promise before SetLocalDescription so we
	// don't miss the event.
	gatherDone := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return nil, err
	}

	// Wait for ICE gathering or context cancellation.
	select {
	case <-gatherDone:
	case <-ctx.Done():
		_ = pc.Close()
		return nil, ctx.Err()
	}

	return &Peer{
		PC:          pc,
		AnswerSDP:   pc.LocalDescription().SDP,
		localTrack:  localTrack,
		remoteTrack: remoteTrackCh,
	}, nil
}

// testConfig returns a Config suitable for unit tests: loopback candidates
// only (no external IPs, no STUN, no mDNS) so ICE gathering completes in
// milliseconds and tests do not require network access.
func testConfig() Config {
	se := webrtc.SettingEngine{}
	// Only gather host candidates on loopback to avoid binding to every
	// network interface on the host (which can take seconds).
	se.SetIPFilter(func(ip net.IP) bool { return ip.IsLoopback() })
	se.SetSTUNGatherTimeout(time.Millisecond)
	se.SetICEMulticastDNSMode(pionice.MulticastDNSModeDisabled)
	return Config{
		ICEServers:    []webrtc.ICEServer{},
		settingEngine: &se,
	}
}
