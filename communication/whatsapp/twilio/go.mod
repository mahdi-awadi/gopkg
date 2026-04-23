module github.com/mahdi-awadi/gopkg/communication/whatsapp/twilio

go 1.23

require (
	github.com/mahdi-awadi/gopkg/communication/provider v0.1.0
	github.com/twilio/twilio-go v1.20.0
)

require (
	github.com/golang/mock v1.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider
