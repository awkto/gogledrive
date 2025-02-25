package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Middleware to check authentication
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(authCookieName)
		if err != nil || cookie.Value != "authenticated" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// loginHandler handles user login
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Simple authentication check (replace with real authentication logic)
	if username == "admin" && password == "password" {
		http.SetCookie(w, &http.Cookie{
			Name:  authCookieName,
			Value: "authenticated",
			Path:  "/",
		})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Login successful")
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

// logoutHandler handles user logout
func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   authCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Logout successful")
}

// uploadHandler handles file uploads
func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form with a max memory of 32MB
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Printf("Error retrieving file: %v", err)
		http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size
	if handler.Size > maxUploadSize {
		log.Printf("File size exceeds maximum allowed: %d > %d", handler.Size, maxUploadSize)
		http.Error(w, "File size exceeds maximum allowed", http.StatusBadRequest)
		return
	}

	// Create a file path
	filename := handler.Filename
	path := filepath.Join(uploadDir, filename)

	// Check if file already exists and generate a unique name if needed
	if _, exists := s.Files[filename]; exists {
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		ext := filepath.Ext(filename)
		timestamp := time.Now().Format("20060102150405")
		filename = fmt.Sprintf("%s_%s%s", base, timestamp, ext)
		path = filepath.Join(uploadDir, filename)
	}

	// Create the file
	dst, err := os.Create(path)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the created file
	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Error saving file: %v", err)
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create file info
	fileInfo := FileInfo{
		Name:      filename,
		Size:      handler.Size,
		Path:      path,
		CreatedAt: time.Now(),
	}

	// Store file info
	s.Files[filename] = fileInfo

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"filename": filename,
	})
}

// listFilesHandler handles listing all files
func (s *Server) listFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files := make([]FileInfo, 0, len(s.Files))
	for _, file := range s.Files {
		files = append(files, file)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// downloadHandler handles file downloads
func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "File parameter is required", http.StatusBadRequest)
		return
	}

	fileInfo, exists := s.Files[filename]
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Open the file
	file, err := os.Open(fileInfo.Path)
	if err != nil {
		http.Error(w, "Error opening file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers for file download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileInfo.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size))

	// Copy the file to the response
	io.Copy(w, file)
}

// shareFileHandler creates a public share for a file
func (s *Server) shareFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	filename := r.FormValue("file")
	if filename == "" {
		http.Error(w, "File parameter is required", http.StatusBadRequest)
		return
	}

	fileInfo, exists := s.Files[filename]
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Generate a public ID if not already shared
	if fileInfo.PublicID == "" {
		fileInfo.PublicID = generateRandomID()
		fileInfo.IsPublic = true
		s.Files[filename] = fileInfo
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"filename": filename,
		"publicId": fileInfo.PublicID,
		"url":      fmt.Sprintf("/public/%s", fileInfo.PublicID),
	})
}

// unshareFileHandler removes public sharing for a file
func (s *Server) unshareFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	filename := r.FormValue("file")
	if filename == "" {
		http.Error(w, "File parameter is required", http.StatusBadRequest)
		return
	}

	fileInfo, exists := s.Files[filename]
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	fileInfo.PublicID = ""
	fileInfo.IsPublic = false
	s.Files[filename] = fileInfo

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"filename": filename,
	})
}

// publicFileHandler serves publicly shared files
func (s *Server) publicFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the public ID from the URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	publicID := parts[2]

	// Find the file with this public ID
	var fileInfo FileInfo
	var found bool
	for _, info := range s.Files {
		if info.PublicID == publicID && info.IsPublic {
			fileInfo = info
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "File not found or not public", http.StatusNotFound)
		return
	}

	// Open the file
	file, err := os.Open(fileInfo.Path)
	if err != nil {
		http.Error(w, "Error opening file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers for file download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileInfo.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size))

	// Copy the file to the response
	io.Copy(w, file)
}

// deleteFileHandler handles file deletion
func (s *Server) deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	filename := r.FormValue("file")
	if filename == "" {
		http.Error(w, "File parameter is required", http.StatusBadRequest)
		return
	}

	fileInfo, exists := s.Files[filename]
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Delete the file
	if err := os.Remove(fileInfo.Path); err != nil {
		http.Error(w, "Error deleting file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from files map
	delete(s.Files, filename)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"filename": filename,
	})
}

// serveStaticFiles serves the frontend static files
func serveStaticFiles(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "./static/index.html")
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
