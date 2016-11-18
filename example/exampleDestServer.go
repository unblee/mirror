package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, req.Host+req.URL.Path+req.URL.RawPath)
	})
	http.HandleFunc("/dest2", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, req.Host+req.URL.Path+req.URL.RawPath)
	})
	http.ListenAndServe(":"+port, nil)
}
