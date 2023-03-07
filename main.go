package main

import (
	"fmt"
	"os"

	"github.com/home-sol/multicast-proxy/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error: %s", err)
		if err != nil {
			panic(err)
		}
	}
}
