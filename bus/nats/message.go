package nats

import "github.com/nats-io/nats.go/jetstream"

// natsMessage wraps jetstream.Msg to implement bus.Message.
type natsMessage struct {
	msg jetstream.Msg
}

func (m *natsMessage) Subject() string {
	return m.msg.Subject()
}

func (m *natsMessage) Data() []byte {
	return m.msg.Data()
}
