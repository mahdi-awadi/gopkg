// Package router contains reusable communication routing helpers.
package router

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// SMSRoutingRule defines routing for a specific country.
type SMSRoutingRule struct {
	CountryCode      string
	ProviderCode     string
	ProviderPriority int
	IsActive         bool
	IsFallback       bool
	RateLimitPerSec  int
	RateLimitPerDay  int
}

// ProviderLookup is the minimal registry method SMSRouter needs.
type ProviderLookup interface {
	Get(code string) provider.Provider
}

// SMSRouter handles country-based SMS provider routing.
type SMSRouter struct {
	registry     ProviderLookup
	routes       map[string][]SMSRoutingRule
	defaultRoute SMSRoutingRule
	mu           sync.RWMutex
}

// NewSMSRouter creates a new SMS router.
func NewSMSRouter(registry ProviderLookup) *SMSRouter {
	return &SMSRouter{
		registry: registry,
		routes:   make(map[string][]SMSRoutingRule),
		defaultRoute: SMSRoutingRule{
			CountryCode:  "*",
			ProviderCode: "twilio_sms",
			IsActive:     true,
			IsFallback:   true,
		},
	}
}

// LoadRoutes loads active routing rules.
func (r *SMSRouter) LoadRoutes(routes []SMSRoutingRule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = make(map[string][]SMSRoutingRule)
	for _, route := range routes {
		if !route.IsActive {
			continue
		}
		r.routes[route.CountryCode] = append(r.routes[route.CountryCode], route)
	}
	for country := range r.routes {
		sortRules(r.routes[country])
	}
}

// AddRoute adds one routing rule.
func (r *SMSRouter) AddRoute(rule SMSRoutingRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[rule.CountryCode] = append(r.routes[rule.CountryCode], rule)
	sortRules(r.routes[rule.CountryCode])
}

// SetDefaultRoute sets the default/fallback route.
func (r *SMSRouter) SetDefaultRoute(rule SMSRoutingRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultRoute = rule
}

// GetProviderForNumber returns the selected SMS provider and country code.
func (r *SMSRouter) GetProviderForNumber(phoneNumber string) (provider.SMSProvider, string, error) {
	countryCode := ExtractCountryCode(phoneNumber)

	r.mu.RLock()
	rules := append([]SMSRoutingRule(nil), r.routes[countryCode]...)
	defaultRule := r.defaultRoute
	r.mu.RUnlock()

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		if sms := r.lookupSMS(rule.ProviderCode); sms != nil && sms.Enabled() {
			return sms, countryCode, nil
		}
	}

	if sms := r.lookupSMS(defaultRule.ProviderCode); sms != nil && sms.Enabled() {
		return sms, countryCode, nil
	}
	return nil, countryCode, fmt.Errorf("no SMS provider available for country %s", countryCode)
}

// SendSMS sends an SMS using the country router.
func (r *SMSRouter) SendSMS(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	sms, _, err := r.GetProviderForNumber(req.RecipientPhone)
	if err != nil {
		return nil, err
	}
	return sms.Send(ctx, req)
}

// GetRoutesForCountry returns a copy of all routes for a country.
func (r *SMSRouter) GetRoutesForCountry(countryCode string) []SMSRoutingRule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]SMSRoutingRule(nil), r.routes[countryCode]...)
	return out
}

// ListAllRoutes returns a copy of all routes.
func (r *SMSRouter) ListAllRoutes() map[string][]SMSRoutingRule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]SMSRoutingRule, len(r.routes))
	for country, routes := range r.routes {
		out[country] = append([]SMSRoutingRule(nil), routes...)
	}
	return out
}

func (r *SMSRouter) lookupSMS(code string) provider.SMSProvider {
	if r.registry == nil {
		return nil
	}
	p := r.registry.Get(code)
	sms, ok := p.(provider.SMSProvider)
	if !ok {
		return nil
	}
	return sms
}

func sortRules(rules []SMSRoutingRule) {
	for i := 0; i < len(rules)-1; i++ {
		for j := 0; j < len(rules)-i-1; j++ {
			if rules[j].ProviderPriority > rules[j+1].ProviderPriority {
				rules[j], rules[j+1] = rules[j+1], rules[j]
			}
		}
	}
}

// ExtractCountryCode extracts the country code from a phone number.
func ExtractCountryCode(phoneNumber string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, phoneNumber)

	if strings.HasPrefix(cleaned, "+") {
		cleaned = cleaned[1:]
		countryCodeMap := map[string]bool{
			"964": true, "971": true, "966": true, "962": true, "963": true,
			"965": true, "968": true, "973": true, "974": true, "961": true,
			"967": true, "970": true, "218": true, "249": true, "213": true,
			"212": true, "216": true, "20": true, "44": true, "49": true,
			"33": true, "39": true, "34": true, "31": true, "32": true,
			"90": true, "86": true, "91": true, "81": true, "82": true,
			"61": true, "64": true, "1": true, "7": true,
		}
		for _, n := range []int{3, 2, 1} {
			if len(cleaned) >= n && countryCodeMap[cleaned[:n]] {
				return "+" + cleaned[:n]
			}
		}
	}
	return "+964"
}
