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
		// Get path parameter (supports slashes)
		urlpath := dir + r.URL.Path[1:]

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
