package router

import (
	"context"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

type fakeRegistry struct {
	providers map[string]provider.Provider
}

func (f fakeRegistry) Get(code string) provider.Provider { return f.providers[code] }

type fakeSMS struct {
	code    string
	enabled bool
}

func (f fakeSMS) Code() string { return f.code }
func (f fakeSMS) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelSMS}
}
func (f fakeSMS) Send(context.Context, *provider.SendRequest) (*provider.SendResponse, error) {
	return &provider.SendResponse{Success: true, ProviderCode: f.code}, nil
}
func (f fakeSMS) GetStatus(context.Context, string) (*provider.DeliveryStatus, error) {
	return nil, nil
}
func (f fakeSMS) ValidateConfig() error                                         { return nil }
func (f fakeSMS) Enabled() bool                                                 { return f.enabled }
func (f fakeSMS) SupportedCountries() []string                                  { return nil }
func (f fakeSMS) CostEstimate(context.Context, string) (float64, string, error) { return 0, "", nil }

func TestExtractCountryCode(t *testing.T) {
	for number, want := range map[string]string{
		"+965 1234 5678": "+965",
		"+1 (555) 0000":  "+1",
		"07700000000":    "+964",
	} {
		if got := ExtractCountryCode(number); got != want {
			t.Fatalf("ExtractCountryCode(%q) = %q, want %q", number, got, want)
		}
	}
}

func TestSMSRouterChoosesLowestPriorityActiveRule(t *testing.T) {
	reg := fakeRegistry{providers: map[string]provider.Provider{
		"a": fakeSMS{code: "a", enabled: true},
		"b": fakeSMS{code: "b", enabled: true},
	}}
	r := NewSMSRouter(reg)
	r.LoadRoutes([]SMSRoutingRule{
		{CountryCode: "+965", ProviderCode: "b", ProviderPriority: 20, IsActive: true},
		{CountryCode: "+965", ProviderCode: "a", ProviderPriority: 10, IsActive: true},
	})
	got, country, err := r.GetProviderForNumber("+96512345678")
	if err != nil {
		t.Fatalf("GetProviderForNumber() err = %v", err)
	}
	if country != "+965" || got.Code() != "a" {
		t.Fatalf("provider=%s country=%s", got.Code(), country)
	}
}

func TestSMSRouterFallsBackToDefault(t *testing.T) {
	reg := fakeRegistry{providers: map[string]provider.Provider{
		"twilio_sms": fakeSMS{code: "twilio_sms", enabled: true},
	}}
	r := NewSMSRouter(reg)
	resp, err := r.SendSMS(context.Background(), &provider.SendRequest{RecipientPhone: "+447000000000"})
	if err != nil {
		t.Fatalf("SendSMS() err = %v", err)
	}
	if !resp.Success || resp.ProviderCode != "twilio_sms" {
		t.Fatalf("resp = %+v", resp)
	}
}
