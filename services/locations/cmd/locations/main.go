package main

import (
	"fmt"
	"github.com/patrickodacre/locations-service/internal/data"
	"net/http"
)

func main() {

	feed := data.NewDataFeed()
	feed.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello from Locations Service")
	})

	http.ListenAndServe(":3000", nil)
}
