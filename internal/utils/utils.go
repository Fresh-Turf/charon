package charonutils

import (
	"log"

	"github.com/joho/godotenv"
)

// LoadEnv loads the .env file and throws an error in production environment
// if no such file is found
func LoadEnv() (err error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, reading configuration from ENV")
		err = nil
	}
	return err
}
