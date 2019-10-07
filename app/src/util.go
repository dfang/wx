package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

// just print request headers
func printRequestHeaders(r *http.Request) {
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			log.Printf("\n\n%s: %s\n\n", name, h)
		}
	}
}

// DumpHTTPRequest for debugging
func DumpHTTPRequest(r *http.Request) {
	output, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println("Error dumping request:", err)
		return
	}
	log.Println(string(output))
}

// DumpHTTPResponse for debugging
func DumpHTTPResponse(resp *http.Response) {
	output, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Println("Error dumping response:", err)
		return
	}
	log.Println(string(output))
}
