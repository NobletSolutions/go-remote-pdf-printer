package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// timerState holds the timer and a unique ID to prevent race conditions during map cleanup
type timerState struct {
	timer *time.Timer
	id    int64
}

func ClearFileMiddleware(rootDirectory *string) gin.HandlerFunc {
	var mu sync.Mutex
	timers := make(map[string]timerState)
	var idCounter int64

	return func(c *gin.Context) {
		if c.Query("download") == "true" {
			// This tells the browser to download the file
			c.Header("Content-Disposition", "attachment")

			// Optional: if you want to force a specific filename, you can do:
			// filename := path.Base(c.Request.URL.Path)
			// c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		}

		// Allow the static file to be served
		c.Next()

		// --- DO SOMETHING AFTER REQUEST ---
		clearValue := c.Query("clear")
		if clearValue != "" {

			u := c.Request.URL
			u.RawQuery = ""

			if strings.HasPrefix(u.Path, "/preview/") || strings.HasPrefix(u.Path, "/png/") || strings.HasPrefix(u.Path, "/pdfs/") {
				filePath := *rootDirectory + "/files" + u.Path

				// Try to parse the clear query param as an integer
				delaySecs, err := strconv.Atoi(clearValue)

				if err == nil && delaySecs > 0 {
					// Handle delayed deletion with debounce
					mu.Lock()

					// Stop existing timer for this file if one is already running
					if ts, exists := timers[filePath]; exists {
						ts.timer.Stop()
					}

					idCounter++
					currentID := idCounter

					// Create a new timer
					t := time.AfterFunc(time.Duration(delaySecs)*time.Second, func() {
						log.Println("Delayed deletion of:", filePath)
						os.Remove(filePath)

						// Safely clean up the map so it doesn't grow indefinitely
						mu.Lock()
						defer mu.Unlock()

						// Only delete from the map if this is still the most recent timer for this file
						if ts, exists := timers[filePath]; exists && ts.id == currentID {
							delete(timers, filePath)
						}
					})

					// Store the new timer state
					timers[filePath] = timerState{
						timer: t,
						id:    currentID,
					}

					mu.Unlock()

				} else {
					// Fallback to immediate deletion (e.g., if clear is just "true" or "0")
					log.Println("Immediate deletion of:", filePath)
					os.Remove(filePath)
				}
			}
		}
	}
}
