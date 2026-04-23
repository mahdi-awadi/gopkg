// Package ses provides an Amazon SES implementation of
// communication/provider.EmailProvider.
package ses

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsses "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "ses"

// Logger is the minimum logging contract this provider uses.
type Logger interface {
	Info(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

type noopLogger struct{}

func (noopLogger) Info(string, map[string]any)  {}
func (noopLogger) Warn(string, map[string]any)  {}
func (noopLogger) Error(string, map[string]any) {}

// Config holds AWS SES connection settings and the default From identity.
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	FromEmail       string
	FromName        string
}

// Provider implements provider.EmailProvider via Amazon SES.
type Provider struct {
	cfg    Config
	client *awsses.Client
	logger Logger
}

// New constructs an SES Provider. logger may be nil (becomes noop).
func New(ctx context.Context, cfg Config, logger Logger) (*Provider, error) {
	if logger == nil {
		logger = noopLogger{}
	}

	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	awscfgLoaded, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(cfg.Region),
		awscfg.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("ses: load aws config: %w", err)
	}

	client := awsses.NewFromConfig(awscfgLoaded)
	return &Provider{cfg: cfg, client: client, logger: logger}, nil
}

// Compile-time check.
var _ provider.EmailProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelEmail}
}

// ValidateConfig returns an error if the client is not ready or FromEmail empty.
func (p *Provider) ValidateConfig() error {
	if p.client == nil {
		return fmt.Errorf("ses: client not initialized")
	}
	if p.cfg.FromEmail == "" {
		return fmt.Errorf("ses: FromEmail is required")
	}
	return nil
}

// Enabled returns true when the client is initialized and FromEmail is present.
func (p *Provider) Enabled() bool {
	return p.client != nil && p.cfg.FromEmail != ""
}

// Send sends a single email via SES.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientEmail == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient email is required",
		}, nil
	}

	from := p.cfg.FromEmail
	if p.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.cfg.FromName, p.cfg.FromEmail)
	}

	var body types.Body
	if req.HTMLBody != "" {
		body = types.Body{
			Html: &types.Content{Charset: aws.String("UTF-8"), Data: aws.String(req.HTMLBody)},
			Text: &types.Content{Charset: aws.String("UTF-8"), Data: aws.String(req.Body)},
		}
	} else {
		body = types.Body{
			Text: &types.Content{Charset: aws.String("UTF-8"), Data: aws.String(req.Body)},
		}
	}

	input := &awsses.SendEmailInput{
		Source: aws.String(from),
		Destination: &types.Destination{
			ToAddresses: []string{req.RecipientEmail},
		},
		Message: &types.Message{
			Subject: &types.Content{Charset: aws.String("UTF-8"), Data: aws.String(req.Subject)},
			Body:    &body,
		},
	}

	if req.CorrelationID != "" {
		input.Tags = append(input.Tags, types.MessageTag{Name: aws.String("CorrelationID"), Value: aws.String(req.CorrelationID)})
	}
	if req.TemplateCode != "" {
		input.Tags = append(input.Tags, types.MessageTag{Name: aws.String("TemplateCode"), Value: aws.String(req.TemplateCode)})
	}

	result, err := p.client.SendEmail(ctx, input)
	if err != nil {
		p.logger.Error("ses: send failed", map[string]any{
			"recipient": MaskEmail(req.RecipientEmail),
			"error":     err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}

	raw, _ := json.Marshal(map[string]any{"message_id": aws.ToString(result.MessageId)})
	p.logger.Info("ses: email sent", map[string]any{
		"recipient":  MaskEmail(req.RecipientEmail),
		"subject":    req.Subject,
		"message_id": aws.ToString(result.MessageId),
	})

	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: aws.ToString(result.MessageId),
		RawResponse:       raw,
	}, nil
}

// SendWithAttachments is not yet implemented for SES; falls back to Send.
// Implementing requires SendRawEmail + MIME encoding.
func (p *Provider) SendWithAttachments(ctx context.Context, req *provider.SendRequest, attachments []provider.Attachment) (*provider.SendResponse, error) {
	p.logger.Warn("ses: attachments not yet implemented, sending without them", map[string]any{
		"attachment_count": len(attachments),
	})
	return p.Send(ctx, req)
}

// GetStatus returns StatusSent; real tracking requires SNS or CloudWatch.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

// MaskEmail masks an email for logging.
func MaskEmail(email string) string {
	if len(email) <= 4 {
		return "****"
	}
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "****"
	}
	if at <= 2 {
		return email[:1] + "***" + email[at:]
	}
	return email[:2] + "***" + email[at:]
}
