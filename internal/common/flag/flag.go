package flag

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	flag "bitbucket.org/Amartha/go-feature-flag-sdk"
	"bitbucket.org/Amartha/go-feature-flag-sdk/listener"
)

var ErrVariantNotFound = fmt.Errorf("variant not found")

type Job struct {
	JobName          string
	Version          string
	Date             string
	BucketName       string
	FlagPublishAcuan bool
	FileName         string
}

type Client interface {
	flag.IFlagger
}

type Variant[T any] struct {
	Enabled bool
	Value   T
}

func New(cfg *config.Config) (Client, error) {
	c, err := flag.NewFlagger(&flag.Config{
		AppName:               cfg.App.Name,
		FeatureFlagServiceURL: cfg.FeatureFlagSDKConfig.URL,
		Token:                 cfg.FeatureFlagSDKConfig.Token,
		Env:                   cfg.FeatureFlagSDKConfig.Env,
		RefreshInterval:       cfg.FeatureFlagSDKConfig.RefreshInterval,
		Listener:              listener.DebugListener{},
		HttpClient:            http.DefaultClient,
	})
	if err != nil {
		return nil, err
	}
	c.WaitForReady()

	return c, nil
}

// GetVariant returns the variant for the given key.
// We use this method because golang doesn't support generic type parameters in method interfaces
// [link_issue](https://github.com/golang/go/issues/49085)
func GetVariant[T any](c Client, key string) (*Variant[T], error) {
	variant := c.GetVariant(key)
	if variant == nil {
		return nil, fmt.Errorf("%w: variant for key %s not found", ErrVariantNotFound, key)
	}

	var res T
	if !variant.Enabled {
		return &Variant[T]{
			Enabled: variant.Enabled,
			Value:   res,
		}, nil
	}

	err := json.Unmarshal([]byte(variant.Payload.Value), &res)
	if err != nil {
		return nil, fmt.Errorf("unmarshal variant for key %s failed: %w", key, err)
	}

	return &Variant[T]{
		Enabled: variant.Enabled,
		Value:   res,
	}, nil
}
