package main

import (
	"fmt"
	"net/http"

	"github.com/patrickodacre/accounts-service/internal/token"
)

func main() {

	maker, err := token.NewJWTMaker("testing")

	if err != nil {
		fmt.Println(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello, Accounts")
	})

	http.ListenAndServe(":3000", nil)
}
