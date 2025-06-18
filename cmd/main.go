package main

import (
	"log"

	_ "gitlab.com/prolegis/form-ai/docs"
	"gitlab.com/prolegis/form-ai/internal/app"
	"gitlab.com/prolegis/form-ai/internal/config"
)

func main() {
	appConfig, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.RunApi(appConfig)
}
