package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	mux := http.NewServeMux()

	dir := "./out"
	fs := http.FileServer(http.Dir(dir))
	mux.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Add("Cache-Control", "no-cache")
		fs.ServeHTTP(resp, req)
	})

	mux.HandleFunc("/api/build", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Add("Cache-Control", "no-cache")

		sum := md5SumOfFiles(dir)
		encoder := json.NewEncoder(resp)
		encoder.Encode(map[string]interface{}{
			"BuildHash": sum,
		})
	})

	log.Print("Serving " + dir + " on http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}

func md5SumOfFiles(path string) string {
	h := md5.New()
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		modifiedTime := info.ModTime().Format(time.RFC3339Nano)
		_, err = h.Write([]byte(modifiedTime))
		return err
	})
	if err != nil {
		panic(err)
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}
