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

	w := bufio.NewWriter(f)
	defer w.Flush()

	// path := `C:\Program Files\JetBrains\GoLand 2023.3.2\bin\goland64.exe`
	// path := `E:\prvw测试文档\QQ拼音截图20240208232818.jpg`
	// path := `C:\Users\Administrator\Downloads\imdb-movies-and-tv.apk`
	// path := `C:\Windows\System32\cmd.exe`
	// path := `C:\Windows\System32\alg.exe`
	// path := `C:\Windows\System32\imageres.dll`
	path := `C:\Windows\SystemResources\imageres.dll.mun`
	// path := `D:\Program Files (x86)\Adobe Illustrator CS4\Support Files\Contents\Windows\Illustrator.exe`
	// path := `app.icns`
	// path := `FileZilla.icns`
	// path := `F:\安装包\android-studio-ide-401-201.6858069-mac.dmg`
	err = f2ico.F2ICO(w, path, f2ico.Config{Index: 11})
	// err = f2ico.F2ICO(w, path, f2ico.Config{Format: "png"})
	// err = f2ico.F2ICO(w, path, f2ico.Config{Format: "png", Width: 48, Height: 48})
	// err = f2ico.F2ICO(w, path)
	if err != nil {
		panic(err)
	}
}
