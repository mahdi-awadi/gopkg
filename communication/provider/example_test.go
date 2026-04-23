package provider_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// fakeProvider is a hand-rolled fake Provider for demonstration.
type fakeProvider struct{}

func (fakeProvider) Code() string                               { return "demo" }
func (fakeProvider) SupportedChannels() []provider.Channel      { return []provider.Channel{provider.ChannelSMS} }
func (fakeProvider) ValidateConfig() error                      { return nil }
func (fakeProvider) Enabled() bool                              { return true }
func (fakeProvider) Send(_ context.Context, _ *provider.SendRequest) (*provider.SendResponse, error) {
	return &provider.SendResponse{Success: true, ProviderCode: "demo"}, nil
}
func (fakeProvider) GetStatus(_ context.Context, id string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: id, Status: provider.StatusDelivered}, nil
}

func ExampleRegistry() {
	r := provider.NewRegistry()
	_ = r.Register(fakeProvider{})

	fmt.Println("count:", r.Len())
	fmt.Println("sms providers:", len(r.ByChannel(provider.ChannelSMS)))
	fmt.Println("email providers:", len(r.ByChannel(provider.ChannelEmail)))
	// Output:
	// count: 1
	// sms providers: 1
	// email providers: 0
}

func ExampleProviderError() {
	err := provider.NewProviderError("twilio", "20429", "too many requests", true, nil)
	fmt.Println(err.Error())
	// Output: twilio: too many requests
}
