package main

import (
	"bufio"
	"os"

	"github.com/orcastor/f2ico"
)

func main() {
	f, err := os.OpenFile("f2ico_demo.ico", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	// path := `C:\Program Files\JetBrains\GoLand 2023.3.2\bin\goland64.exe`
	path := `C:\Users\Administrator\Downloads\imdb-movies-and-tv.apk`
	err = f2ico.F2ICO(bufio.NewWriter(f), path)
	if err != nil {
		panic(err)
	}
}
