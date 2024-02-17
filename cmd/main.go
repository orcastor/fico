package main

import (
	"bufio"
	"os"

	"github.com/orcastor/fico"
)

func main() {
	f, err := os.OpenFile("fico_demo.png", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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
	// path := `C:\Windows\SystemResources\imageres.dll.mun`
	// path := `D:\Program Files (x86)\Adobe Illustrator CS4\Support Files\Contents\Windows\Illustrator.exe`
	// path := `app.icns`
	// path := `FileZilla.icns`
	// path := `F:\安装包\android-studio-ide-401-201.6858069-mac.dmg`
	path := `E:\Download\ETax.exe`
	// path := `E:\Download\weixin6.2.5.apk`
	// err = fico.F2ICO(w, path, fico.Config{Width: 48, Height: 48})
	// err = fico.F2ICO(w, path, fico.Config{Format: "png"})
	// err = fico.F2ICO(w, path, fico.Config{Format: "png", Width: 48, Height: 48})
	err = fico.F2ICO(w, path, fico.Config{Format: "png", Width: 32, Height: 32})
	// idx := -184
	// err = fico.F2ICO(w, path, fico.Config{Format: "png", Index: &idx, WiWidth: 32, Height: 32})
	// err = fico.F2ICO(w, path)
	if err != nil {
		panic(err)
	}
}
