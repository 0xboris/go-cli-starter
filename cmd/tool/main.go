package main

import (
	"os"

	"example.com/tool/internal/app"
)

func main() {
	os.Exit(int(app.Main()))
}
