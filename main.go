package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProgressReader wraps a reader and reports progress
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	read     int64
	Progress func(bytesRead int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)

	if pr.Progress != nil {
		pr.Progress(pr.read)
	}

	return n, err
}

func main() {
	// This should be a simple HTTP server that:
	// Allows file downloads
	// Allows uploads
	// Allows directory listing
	// Allows file deletion
	// Allows file renaming
	var port string = "6040"
	var dir string = "storage/"

	http.HandleFunc("GET /{file...}", func(w http.ResponseWriter, r *http.Request) {
		urlpath := dir + r.URL.Path[1:]
		if stat, err := os.Stat(urlpath); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, urlpath)
			return
		}

		w.WriteHeader(http.StatusOK)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.Contains(path, strings.TrimPrefix(urlpath, dir)) {
				fmt.Fprintf(w, "%s\n", strings.TrimPrefix(path, dir))
			}
			return nil
		})
	})

	http.HandleFunc("POST /{file...}", func(w http.ResponseWriter, r *http.Request) {
		urlpath := dir + r.URL.Path[1:]

		// Parse multipart form FIRST, before any response writing
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error getting file from form: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileSize := header.Size

		if err := os.MkdirAll(filepath.Dir(urlpath), os.ModePerm); err != nil {
			http.Error(w, "Error creating directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		out, err := os.Create(urlpath)
		if err != nil {
			http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer out.Close()

		// Set headers for streaming response ONLY NOW
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)

		// Flush the headers
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Create progress reader that writes updates to response
		progressReader := &ProgressReader{
			Reader: file,
			Total:  fileSize,
			Progress: func(bytesRead int64) {
				if fileSize > 0 {
					percent := float64(bytesRead) / float64(fileSize) * 100
					msg := fmt.Sprintf("Progress: %.1f%% (%d/%d bytes)\r",
						percent, bytesRead, fileSize)
					w.Write([]byte(msg))
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
			},
		}

		// Copy with progress
		_, err = io.Copy(out, progressReader)
		if err != nil {
			fmt.Fprintf(w, "\nError during copy: %v\n", err)
			return
		}

		fmt.Fprintf(w, "Upload completed: %s\n", urlpath)
	})

	http.HandleFunc("DELETE /{file...}", func(w http.ResponseWriter, r *http.Request) {
		// Get path parameter (supports slashes)
		urlpath := dir + r.URL.Path[1:]

		if _, err := os.Stat(urlpath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Delete the file
		if err := os.RemoveAll(urlpath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(urlpath))
	})

	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "-") {
			switch strings.ToLower(arg) {
			case "--port":
			case "-p":
				port = os.Args[i+1]
				// Verify that the port is a number
				if val, err := strconv.Atoi(port); err != nil || val < 1 || val > 65535 {
					fmt.Println("Invalid port number")
					os.Exit(1)
				}
			case "--dir":
			case "-d":
				dir = strings.TrimPrefix(os.Args[i+1], "./")
				if !strings.HasSuffix(dir, "/") {
					dir += "/"
				}
			case "-h":
				fmt.Println("Usage: fileserver [--port <port>] [-h]")
				fmt.Println("Options:")
				fmt.Printf("  --port <port>   Port to listen on (default: %s)\n", port)
				fmt.Printf("  --help       	  Show this help message\n")
				os.Exit(0)
			}
			continue
		}
	}
	fmt.Printf("FileServer listening on http://localhost:%s\n", port)
	fmt.Println("Hosting directory:", dir)
	fmt.Println("Press Ctrl+C to stop")
	http.ListenAndServe(":"+port, nil)
}
