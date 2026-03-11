package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	ol "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin"
	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)

// Client wraps the OneLogin SDK client with provider configuration.
type Client struct {
	SDK    *ol.OneloginSDK
	APIURL string

	// appNameCache caches app ID → display name lookups for the lifetime of
	// a Terraform operation, shared across all role resource reads.
	appNameCache     sync.Map // map[int32]string
	appCacheLoadOnce sync.Once
}

// PreloadAppCache fetches all apps from OneLogin once (paginated) and populates
// the app name cache. Subsequent calls are no-ops (sync.Once).
func (c *Client) PreloadAppCache(ctx context.Context) error {
	var loadErr error
	c.appCacheLoadOnce.Do(func() {
		total := 0
		page := 1
		tflog.Debug(ctx, "onelogin: PreloadAppCache starting")
		for {
			result, err := c.SDK.GetApps(&models.AppQuery{
				Limit: "1000",
				Page:  fmt.Sprintf("%d", page),
			})
			if err != nil {
				tflog.Debug(ctx, "onelogin: PreloadAppCache error", map[string]any{"page": page, "error": err.Error()})
				loadErr = err
				return
			}
			apps, err := UnmarshalApps(result)
			if err != nil {
				loadErr = err
				return
			}
			tflog.Debug(ctx, "onelogin: PreloadAppCache page done", map[string]any{"page": page, "count": len(apps)})
			for i := range apps {
				if apps[i].ID != nil && apps[i].Name != nil {
					c.appNameCache.Store(*apps[i].ID, *apps[i].Name)
					total++
				}
			}
			if len(apps) < 1000 {
				break
			}
			page++
		}
		tflog.Debug(ctx, "onelogin: PreloadAppCache complete", map[string]any{"total": total})
	})
	return loadErr
}

// CachedAppName returns the cached name for appID if present, else ("", false).
func (c *Client) CachedAppName(appID int32) (string, bool) {
	v, ok := c.appNameCache.Load(appID)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// SetCachedAppName stores name for appID in the cache.
func (c *Client) SetCachedAppName(appID int32, name string) {
	c.appNameCache.Store(appID, name)
}
