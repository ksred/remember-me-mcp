// +build ignore

package main

import (
	"fmt"
	"github.com/ksred/remember-me-mcp/internal/utils"
)

func main() {
	fmt.Println("=== Development Logger (Pretty Output) ===")
	devLogger := utils.NewLogger(utils.DevelopmentConfig())
	devLogger.Info().Str("component", "server").Msg("Server started")
	devLogger.Debug().Int("port", 8080).Msg("Listening on port")
	devLogger.Warn().Msg("Using default configuration")
	devLogger.Error().Err(fmt.Errorf("connection failed")).Msg("Database connection error")
	
	fmt.Println("\n=== Production Logger (JSON Output) ===")
	prodLogger := utils.NewLogger(utils.ProductionConfig())
	prodLogger.Info().Str("component", "server").Msg("Server started")
	prodLogger.Debug().Int("port", 8080).Msg("This won't show - debug level is below info")
	prodLogger.Warn().Msg("Using default configuration")
	prodLogger.Error().Err(fmt.Errorf("connection failed")).Msg("Database connection error")
}