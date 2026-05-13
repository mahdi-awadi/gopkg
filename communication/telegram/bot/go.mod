module github.com/mahdi-awadi/gopkg/communication/telegram/bot

go 1.26

replace github.com/mahdi-awadi/eticket-v3/services/common => /home/eticket-v3/services/common

replace github.com/mahdi-awadi/gopkg/ai/conversation => /home/gopkg/ai/conversation

require (
	github.com/google/uuid v1.6.0
	github.com/mahdi-awadi/eticket-v3/services/common v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.18.0
	go.uber.org/zap v1.28.0
	gopkg.in/telebot.v4 v4.0.0-beta.8
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/mahdi-awadi/gopkg/ai/conversation v0.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
