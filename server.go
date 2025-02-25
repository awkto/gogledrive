package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Configuration
const (
	Port           = "8080" // Changed to uppercase to export
	uploadDir      = "./uploads"
	maxUploadSize  = 50 * 1024 * 1024 // 50MB
	authCookieName = "go_drive_auth"
)

// FileInfo represents metadata for a file
type FileInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Path      string    `json:"-"` // Not exposed in JSON
	PublicID  string    `json:"publicId,omitempty"`
	IsPublic  bool      `json:"isPublic"`
	CreatedAt time.Time `json:"createdAt"`
}

// Server represents our application server
type Server struct {
	Files map[string]FileInfo // filename -> FileInfo
}

// NewServer initializes a new server instance
func NewServer() *Server {
	return &Server{
		Files: make(map[string]FileInfo),
	}
}

// Initialize server and set up the upload directory
func (s *Server) Initialize() error {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Load existing files
	err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(uploadDir, path)
			s.Files[relPath] = FileInfo{
				Name:      info.Name(),
				Size:      info.Size(),
				Path:      path,
				CreatedAt: info.ModTime(),
			}
		}
		return nil
	})

	return err
}
