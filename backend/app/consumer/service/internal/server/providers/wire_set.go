package providers

import (
	"github.com/google/wire"

	"go-wind-admin/app/consumer/service/internal/server"
)

// ProviderSet is the Wire provider set for server layer.
var ProviderSet = wire.NewSet(
	server.NewRestMiddleware,
	server.NewRestServer,
	server.NewKafkaServer,
)
