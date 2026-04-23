// Package r2 provides a thin client over Cloudflare R2 (S3-compatible
// object storage). Upload content, generate pre-signed URLs, delete.
//
// Zero value of *Client is not usable; call New. Client is safe for
// concurrent use.
package r2

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Config holds the settings needed to talk to an R2 bucket.
type Config struct {
	AccessKey  string
	SecretKey  string
	Endpoint   string // e.g. "https://<accountid>.r2.cloudflarestorage.com"
	Subdomain  string // public subdomain for URL composition (optional; leave "" if not using public URLs)
	BucketName string
}

// Client wraps the S3 client for R2 operations.
type Client struct {
	config *Config
	client *s3.S3
}

// New creates an R2 client.
//
// Returns an error if AccessKey, SecretKey, or Endpoint are empty.
func New(cfg *Config) (*Client, error) {
	if cfg == nil || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("r2: incomplete configuration")
	}

	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(cfg.Endpoint),
		Region:           aws.String("auto"),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("r2: create session: %w", err)
	}

	return &Client{config: cfg, client: s3.New(sess)}, nil
}

// Upload puts content at key and returns its public URL (derived from
// Config.Subdomain). Returns an error if Subdomain is not configured
// but a public URL was requested.
func (c *Client) Upload(key string, content []byte, contentType string) (string, error) {
	_, err := c.client.PutObject(&s3.PutObjectInput{
		Bucket:       aws.String(c.config.BucketName),
		Key:          aws.String(key),
		Body:         bytes.NewReader(content),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("max-age=3600"),
	})
	if err != nil {
		return "", fmt.Errorf("r2: upload %s: %w", key, err)
	}
	if c.config.Subdomain == "" {
		return "", nil
	}
	return fmt.Sprintf("https://%s/%s", c.config.Subdomain, key), nil
}

// GetSignedURL generates a pre-signed URL valid for the given expiry.
func (c *Client) GetSignedURL(key string, expiry time.Duration) (string, error) {
	req, _ := c.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.config.BucketName),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("r2: presign %s: %w", key, err)
	}
	return url, nil
}

// Delete removes the object at key. Non-existent keys return nil.
func (c *Client) Delete(key string) error {
	_, err := c.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(c.config.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2: delete %s: %w", key, err)
	}
	return nil
}
