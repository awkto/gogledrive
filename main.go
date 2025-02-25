package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	server := NewServer()
	if err := server.Initialize(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Set up routes
	http.HandleFunc("/", serveStaticFiles)
	http.HandleFunc("/upload", server.authMiddleware(server.uploadHandler))
	http.HandleFunc("/list", server.authMiddleware(server.listFilesHandler))
	http.HandleFunc("/download", server.authMiddleware(server.downloadHandler))
	http.HandleFunc("/share", server.authMiddleware(server.shareFileHandler))
	http.HandleFunc("/unshare", server.authMiddleware(server.unshareFileHandler))
	http.HandleFunc("/public/", server.publicFileHandler)
	http.HandleFunc("/delete", server.authMiddleware(server.deleteFileHandler))
	http.HandleFunc("/login", server.loginHandler)
	http.HandleFunc("/logout", server.logoutHandler)

	fmt.Printf("Server starting on port %s...\n", Port)
	fmt.Printf("Open http://localhost:%s in your browser\n", Port)

	if err := http.ListenAndServe(":"+Port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
