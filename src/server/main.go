package main

import (
	"fmt"
	"net/http"
)

func main() {
	port := ":9090"
	fmt.Printf("Running on port %s\n", port)
	err := http.ListenAndServe(port, http.FileServer(http.Dir("../../assets")))
	if err != nil {
		fmt.Println("Failed to start server", err)
		return
	}
}
