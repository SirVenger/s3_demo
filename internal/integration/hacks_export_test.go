package integration

import (
	"time"

	_ "unsafe"
)

//go:linkname callSweepOnce github.com/sir_venger/s3_lite/internal/app/storagehttp.sweepOnce
func callSweepOnce(root string, ttl time.Duration) error
