package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/gofiber/fiber/v2"
)

func (m *AppMiddleware) CheckIdempotentRequest() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// only transaction method POST
		if c.Method() != fiber.MethodPost {
			return c.Next()
		}

		idempotencyKey := c.Get("X-Idempotency-Key")
		if idempotencyKey == "" {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, common.ErrMissingIdempotencyKey)
		}

		// generate new context, prevent context timeout error if request already finished from handler
		ctx := c.UserContext()

		idm, err := m.getOrCreateIdempotency(ctx, idempotencyKey, c.Body())
		if err != nil {
			if errors.Is(err, common.ErrInvalidFingerprint) {
				return http.RestErrorResponse(c, fiber.StatusUnprocessableEntity, err)
			} else if errors.Is(err, common.ErrRequestBeingProcessed) {
				return http.RestErrorResponse(c, fiber.StatusConflict, err)
			} else {
				return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
			}
		}

		if idm.StatusProcess == models.IdempotencyStatusProcessFinished {
			c.Response().SetStatusCode(idm.HTTPStatusCode)
			c.Response().SetBodyString(idm.ResponseBody)
			for k, v := range idm.ResponseHeaders {
				c.Response().Header.Set(k, v)
			}
			return nil
		}

		err = c.Next()
		if err != nil {
			return err
		}

		statusCode := c.Response().StatusCode()

		if statusCode < fiber.StatusOK || statusCode >= fiber.StatusMultipleChoices {
			// release lock if request failed, so if the same request is made, it will be processed again
			// this is useful for retry mechanism (ex: 5xx error / timeout / insufficient balance / etc.)
			return m.releaseLock(ctx, idm)
		}

		// set response to idempotency data and set status idempotency to finished
		headers := make(map[string]string)
		for k, v := range c.GetRespHeaders() {
			if len(v) > 0 {
				headers[k] = v[len(v)-1] // use last value
			}
		}

		idm.SetResponse(statusCode, headers, string(c.Response().Body()))

		// save idempotency data to cache if the request is successful
		// so the next request with same idempotency key & fingerprint will get the same response
		err = m.saveResponseToCache(ctx, idm)
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		return nil
	}
}

// getOrCreateIdempotency will get idempotency data from cache, if not found, it will create new one.
// created idempotency will be using status pending since the request is still being processed
func (m *AppMiddleware) getOrCreateIdempotency(ctx context.Context, key string, requestBody []byte) (*models.Idempotency, error) {
	idm := models.NewIdempotency(key, models.IdempotencyStatusProcessPending, requestBody)

	strIdm, err := m.cacheRepo.Get(ctx, idm.CacheKey)
	if errors.Is(err, common.ErrDataNotFound) {
		err = m.createLock(ctx, idm)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get idempotency data: %w", err)
	}

	if strIdm == "" {
		// no previous idempotency data found
		return idm, nil
	}

	var cachedIdm models.Idempotency
	err = json.Unmarshal([]byte(strIdm), &cachedIdm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal idempotency data: %w", err)
	}

	if cachedIdm.Fingerprint != idm.Fingerprint {
		return nil, common.ErrInvalidFingerprint
	}

	if cachedIdm.StatusProcess == models.IdempotencyStatusProcessPending {
		return nil, common.ErrRequestBeingProcessed
	}

	return &cachedIdm, nil
}

func (m *AppMiddleware) saveResponseToCache(ctx context.Context, idm *models.Idempotency) error {
	var bytIdm []byte
	bytIdm, err := json.Marshal(idm)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency data: %w", err)
	}

	err = m.cacheRepo.Set(ctx, idm.CacheKey, string(bytIdm), models.TTLIdempotency)
	if err != nil {
		return fmt.Errorf("failed to save idempotency data: %w", err)
	}

	return nil
}

func (m *AppMiddleware) createLock(ctx context.Context, idm *models.Idempotency) error {
	var bytIdm []byte
	bytIdm, err := json.Marshal(idm)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency data: %w", err)
	}

	set, err := m.cacheRepo.SetIfNotExists(ctx, idm.CacheKey, string(bytIdm), models.TTLIdempotency)
	if err != nil {
		return fmt.Errorf("failed to save idempotency data: %w", err)
	}

	// there is possibility same request is being processed by another process simultaneously
	if !set {
		return common.ErrRequestBeingProcessed
	}

	return nil
}

func (m *AppMiddleware) releaseLock(ctx context.Context, idm *models.Idempotency) error {
	err := m.cacheRepo.Del(ctx, idm.CacheKey)
	if err != nil {
		return fmt.Errorf("failed to release idempotency data: %w", err)
	}

	return nil
}
