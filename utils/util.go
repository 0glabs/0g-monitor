package utils

import (
	"fmt"
	"time"
)

func PrettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}
