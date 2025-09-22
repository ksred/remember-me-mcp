package main

import (
	"fmt"
	"log"

	"github.com/ksred/remember-me-mcp/internal/utils"
)

func main() {
	fmt.Println("Generating encryption master key...")
	
	key, err := utils.GenerateMasterKey()
	if err != nil {
		log.Fatalf("Failed to generate master key: %v", err)
	}
	
	fmt.Println("\nGenerated master key (base64 encoded):")
	fmt.Println(key)
	fmt.Println("\nAdd this to your .env file or environment variables as:")
	fmt.Printf("ENCRYPTION_MASTER_KEY=%s\n", key)
	fmt.Println("\nIMPORTANT: Keep this key secure. If you lose it, you won't be able to decrypt your data!")
}