package helpers

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// The background() helper accepts an arbitrary function as a parameter.
func Background(logger *logrus.Logger, wg sync.WaitGroup, fn func()) {
	wg.Add(1)

	// Launch a background goroutine.
	go func() {

		// decrement the WaitGroup counter before the goroutine returns.
		defer wg.Done()

		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				logger.Error(fmt.Errorf("%s", err))
			}
		}()

		// Execute the arbitrary function that we passed as the parameter.
		fn()
	}()
}
