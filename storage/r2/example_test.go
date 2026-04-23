package r2_test

import (
	"time"

	"github.com/mahdi-awadi/gopkg/storage/r2"
)

func Example() {
	c, err := r2.New(&r2.Config{
		AccessKey:  "ak",
		SecretKey:  "sk",
		Endpoint:   "https://<account-id>.r2.cloudflarestorage.com",
		Subdomain:  "files.example.com",
		BucketName: "my-bucket",
	})
	if err != nil {
		return
	}
	_, _ = c.Upload("avatars/123.png", []byte{0x89, 0x50}, "image/png")
	_, _ = c.GetSignedURL("avatars/123.png", 5*time.Minute)
	_ = c.Delete("avatars/123.png")
}
