package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

// Generate a random string for public sharing IDs
func generateRandomID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
