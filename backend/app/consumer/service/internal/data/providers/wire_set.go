//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

// This file defines the dependency injection ProviderSet for the data layer and contains no business logic.
// The build tag `wireinject` excludes this source from normal `go build` and final binaries.
// Run `go generate ./...` or `go run github.com/google/wire/cmd/wire` to regenerate the Wire output (e.g. `wire_gen.go`), which will be included in final builds.
// Keep provider constructors here only; avoid init-time side effects or runtime logic in this file.

package providers

import (
	"github.com/google/wire"

	"go-wind-admin/app/consumer/service/internal/data"
)

// ProviderSet is the Wire provider set for data layer.
var ProviderSet = wire.NewSet(
	data.NewData,
	data.NewEntClient,
	data.NewPasswordCrypto,
	data.NewJWTManager,
	data.NewSMSManager,
	data.NewWechatClient,
	data.NewOSSClient,
	data.NewEventBus,

	// Repository providers
	data.NewConsumerRepo,
	data.NewLoginLogRepo,
	data.NewSMSLogRepo,
	data.NewPaymentOrderRepo,
	data.NewFinanceAccountRepo,
	data.NewFinanceTransactionRepo,
	data.NewMediaFileRepo,
	data.NewLogisticsTrackingRepo,
	// data.NewFreightTemplateRepo,
)
