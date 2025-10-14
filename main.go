package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// This should be a simple HTTP server that:
	// Allows file downloads
	// Allows uploads
	// Allows directory listing
	// Allows file deletion
	// Allows file renaming

	http.HandleFunc("GET /{file...}", func(w http.ResponseWriter, r *http.Request) {
		urlpath := r.URL.Path[1:]
		if strings.HasPrefix(urlpath, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		if stat, err := os.Stat(urlpath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, r.URL.Path[1:])
			return
		}

		w.WriteHeader(http.StatusOK)
		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && !strings.HasPrefix(path, ".") && strings.HasPrefix(path, urlpath) {
				fmt.Fprintf(w, "%s\n", path)
			}
			return nil
		})
	})

	http.HandleFunc("POST /{file...}", func(w http.ResponseWriter, r *http.Request) {
		// Get path parameter (supports slashes)
		urlpath := r.URL.Path[1:]
		if strings.HasPrefix(urlpath, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(urlpath), os.ModePerm); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get the file from the request
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Write the uploaded file
		out, err := os.Create(urlpath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(urlpath))
	})

	http.HandleFunc("DELETE /{file...}", func(w http.ResponseWriter, r *http.Request) {
		// Get path parameter (supports slashes)
		urlpath := r.URL.Path[1:]
		if strings.HasPrefix(urlpath, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		// Delete the file
		if err := os.Remove(urlpath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(urlpath))
	})

	fmt.Println("Listening on http://localhost:6040")
	http.ListenAndServe(":6040", nil)
}
