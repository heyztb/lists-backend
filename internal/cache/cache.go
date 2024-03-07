package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/rs/zerolog/log"
)

var Cache *bigcache.BigCache
var SRPServerKey = "srp-server-struct:%d"

func init() {
	var err error
	Cache, err = bigcache.New(
		context.Background(),
		bigcache.Config{
			Shards:      1024,
			LifeWindow:  5 * time.Minute,
			CleanWindow: 1 * time.Minute,
		},
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initalize in-memory cache")
	}
}