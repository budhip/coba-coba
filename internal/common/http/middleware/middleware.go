package middleware

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
)

type AppMiddleware struct {
	conf         config.Config
	cacheRepo    repositories.CacheRepository
	dlqProcessor services.DLQProcessorService
}

func NewMiddleware(
	conf config.Config,
	cacheRepo repositories.CacheRepository,
	dlqProcessor services.DLQProcessorService) AppMiddleware {
	return AppMiddleware{
		conf:         conf,
		cacheRepo:    cacheRepo,
		dlqProcessor: dlqProcessor,
	}
}
