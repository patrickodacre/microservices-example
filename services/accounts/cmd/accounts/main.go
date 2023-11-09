package main

import (
	"fmt"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello, Accounts")
	})

	http.ListenAndServe(":3000", nil)
}
