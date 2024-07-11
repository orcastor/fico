package fico

import (
	"archive/zip"
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"

	"gopkg.in/ini.v1"

	_ "image/gif"
	_ "image/jpeg"

	"github.com/andrianbdn/iospng"
	"github.com/appflight/apkparser"
	_ "github.com/cbeer/jpeg2000"
	"github.com/tmc/icns"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
)

type Config struct {
	Format string // png or ico(default)
	Width  int    // 0 for all
	Height int    // 0 for all
	Index  *int   // 0 default, nil for all，enabled for PE only
}

func F2ICO(w io.Writer, path string, cfg ...Config) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	// https://superuser.com/questions/1480268/icons-no-longer-in-imageres-dll-in-windows-10-1903-4kb-file
	case ".exe", ".dll", ".mui", ".mun":
		return PE2ICO(w, path, cfg...)
	}

	switch ext {
	case ".ico", ".icns", ".bmp", ".gif", ".jpg", ".jpeg", ".png", ".tiff":
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		switch ext {
		case ".ico": // FIXME：如果只需要其中的一种尺寸
			_, err = io.Copy(w, f)
			return err
		case ".icns":
			return ICNS2ICO(w, f, cfg...)
		case ".bmp", ".gif", ".jpg", ".jpeg", ".png", ".tiff":
			return IMG2ICO(w, f, cfg...)
		}

	case ".apk":
		appInfo, err := apkparser.ParseApk(path)
		if err != nil {
			return err
		}

		return img2ICO(w, appInfo.Icon, cfg...)

	case ".ipa":
		r, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer r.Close()

		var iosIconFile *zip.File
		for _, f := range r.File {
			switch {
			case strings.Contains(f.Name, "AppIcon"):
				iosIconFile = f
			}
		}

		rc, err := iosIconFile.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		var buf bytes.Buffer
		iospng.PngRevertOptimization(rc, &buf)

		return IMG2ICO(w, bytes.NewReader(buf.Bytes()), cfg...)
	}

	return errors.New("conversion failed")
}

type Info struct {
	IconFile  string
	IconIndex *int
}

func GetInfo(path string) (info Info, err error) {
	ext := strings.ToLower(filepath.Ext(path))

	var f *ini.File
	switch ext {
	case ".inf", ".ini", ".desktop":
		f, err = ini.Load(path)
		if err != nil {
			return info, err
		}

	// *.app目录
	case ".app":
		/*
		*.app/Contents/Resources/AppIcon.icns
		 */
		info.IconFile = filepath.Join(path, "Contents/Resources/AppIcon.icns")
		return
	case ".exe", ".dll", ".mui", ".mun", ".ico", ".bmp", ".gif", ".jpg", ".jpeg", ".png", ".tiff", ".icns", ".dmg", ".ipa", ".apk":
		// 尝试把iconfile设置为自己
		info.IconFile = path
		return
	default:
		// 不支持的格式，返回空
		return
	}

	switch ext {
	// 配置文件
	// autorun.inf、desktop.ini、*.desktop(*.AppImage/*.run)
	case ".inf":
		/*
			在 Windows 系统中，autorun.inf 文件用于自定义 CD、DVD 或 USB 驱动器上的自动运行功能。您可以在 autorun.inf 文件中定义要显示的图标。以下是如何定义图标的方法：

			使用 Icon 指令：
			在 autorun.inf 文件中添加 Icon 指令，并指定要显示的图标文件的路径。图标文件可以是 .ico 格式的图标文件。

			示例：

			[AutoRun]
			Icon=path\to\icon.ico

			在这个示例中，Icon 指令指定了要显示的图标文件的路径。

			使用 DefaultIcon 指令：
			另一种定义图标的方法是使用 DefaultIcon 指令。与 Icon 指令类似，DefaultIcon 指令也用于指定要显示的图标文件的路径。

			示例：

			[AutoRun]
			DefaultIcon=path\to\icon.ico

			与 Icon 指令不同的是，DefaultIcon 指令可以同时用于指定文件和文件夹的图标。

			在这两种方法中，path\to\icon.ico 是要显示的图标文件的路径。

			完成后，将 autorun.inf 文件与您的可移动媒体（如 CD、DVD 或 USB 驱动器）一起放置，并在 Windows 系统中插入该媒体，系统会根据 autorun.inf 文件中的设置自动运行，并显示所指定的图标。
		*/
		section, err := f.GetSection("AutoRun")
		if err != nil {
			return info, err
		}

		info.IconFile = section.Key("IconFile").MustString(section.Key("DefaultIcon").String())
	case ".ini":
		/*
			在 Windows 操作系统中，desktop.ini 文件用于自定义文件夹的外观和行为。您可以在文件夹中创建 desktop.ini 文件，并在其中指定如何显示该文件夹的图标。

			要在 desktop.ini 文件中定义图标，可以使用 IconFile 和 IconIndex 字段。下面是一个示例 desktop.ini 文件的基本结构：

			[.ShellClassInfo]
			IconFile=path\to\icon.ico
			IconIndex=0
			[.ShellClassInfo]
			IconResource=%SystemRoot%\system32\imageres.dll,-184

			IconFile 字段指定要用作文件夹图标的图标文件的路径。这可以是包含图标的 .ico 文件，也可以是 .exe 或 .dll 文件，其中包含一个或多个图标资源。
			IconIndex 字段指定要在 IconFile 中使用的图标的索引。如果 IconFile 是 .ico 文件，则索引从0开始，表示图标在文件中的位置。如果 IconFile 是 .exe 或 .dll 文件，则索引表示图标资源的标识符。
			完成后，您可以将 desktop.ini 文件放置在所需文件夹中，并在 Windows 资源管理器中刷新文件夹，以查看所指定的图标。
		*/
		section, err := f.GetSection(".ShellClassInfo")
		if err != nil {
			return info, err
		}

		info.IconFile = section.Key("IconFile").String()
		if info.IconFile != "" {
			if idx, err := section.Key("IconIndex").Int(); err == nil {
				info.IconIndex = &idx
			}
		} else {
			iconResource := section.Key("IconResource").String()
			s := strings.Split(iconResource, ",")
			if len(s) >= 1 {
				info.IconFile = s[0]
				if len(s) >= 2 {
					if idx, err := strconv.Atoi(s[1]); err == nil {
						info.IconIndex = &idx
					}
				}
			}
		}
	case ".desktop":
		/*
			创建包含图标和其他资源的 .desktop 文件来为 .AppImage/.run 文件指定图标。然后，您可以将 .AppImage/.run 文件与 .desktop 文件一起分发，并通过 .desktop 文件来启动 .AppImage/.run 文件，并在系统中显示指定的图标。

			以下是一个示例 .desktop 文件的基本结构：

			[Desktop Entry]
			Version=1.0
			Type=Application
			Name=YourApp
			Icon=/path/to/your/icon.png
			Exec=/path/to/your/run/file.run
			Terminal=false

			您需要将 Icon 字段设置为指向您要在系统中显示的图标文件的路径，并将 Exec 字段设置为指向您的 .AppImage/.run 文件的路径。然后，您可以将 .desktop 文件放置在系统的应用程序启动器中，用户可以通过单击该图标来运行 .run 文件，并显示指定的图标。
		*/
		section, err := f.GetSection("Desktop Entry")
		if err != nil {
			return info, err
		}

		info.IconFile = section.Key("Icon").String()
		if info.IconFile == "" {
			info.IconFile = section.Key("Exec").String()
		}
	}
	return
}

func IMG2ICO(w io.Writer, r io.Reader, cfg ...Config) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}

	return img2ICO(w, zoomImg(img, cfg...), cfg...)
}

func img2ICO(w io.Writer, img image.Image, cfg ...Config) (err error) {
	var buf bytes.Buffer
	png.Encode(&buf, img)

	if len(cfg) <= 0 || cfg[0].Format != "png" {
		err = binary.Write(w, binary.LittleEndian, &ICONDIR{Type: 1, Count: 1})
		if err != nil {
			return err
		}

		err = binary.Write(w, binary.LittleEndian, &ICONDIRENTRY{
			IconCommon: IconCommon{
				Width:      uint8(img.Bounds().Dx()),
				Height:     uint8(img.Bounds().Dy()),
				Planes:     1,
				BitCount:   32,
				BytesInRes: uint32(buf.Len()),
			},
			Offset: 0x16,
		})
		if err != nil {
			return err
		}
	}

	_, err = w.Write(buf.Bytes())
	return err
}

// https://github.com/nyteshade/ByteRunLengthCoder/blob/main/ByteRunLengthCoder.swift
func icnsBRLDecode(d []byte) (ret []byte) {
	for i := 0; i < len(d); {
		b := d[i]
		if b < 0x80 {
			cnt := int(b) + 1
			if i+cnt >= len(d) {
				break
			}
			ret = append(ret, d[i+1:i+1+cnt]...)
			i += cnt + 1
		} else {
			cnt := int(b) - 0x80 + 3
			if i+1 >= len(d) {
				break
			}
			tb := d[i+1]
			s := make([]byte, cnt)
			for i := range s {
				s[i] = tb
			}
			ret = append(ret, s...)
			i += 2
		}
	}
	return
}

func isPNG(d []byte) bool {
	return len(d) > 8 && string(d[:8]) == "\211PNG\r\n\032\n"
}

func isARGB(d []byte) bool {
	return len(d) > 4 && string(d[:4]) == "ARGB"
}

// https://en.wikipedia.org/wiki/Apple_Icon_Image_format
func ICNS2ICO(w io.Writer, r io.Reader, cfg ...Config) error {
	iconSet, err := icns.Parse(r)
	if err != nil {
		return err
	}

	// 掩码映射
	maskMap := make(map[int]*icns.Icon)
	var newSet icns.IconSet
	// 过滤掉无用的OSType
	for _, icon := range iconSet {
		switch string(icon.Type[:]) {
		case "TOC ", "icnV", "name", "info", "sbtp", "slct", "\xFD\xD9\x2F\xA8":
			continue
		case "s8mk", "l8mk", "h8mk", "t8mk":
			maskMap[len(newSet)-1] = icon
		default:
			newSet = append(newSet, icon)
		}
	}

	var d [][]byte
	var entries []ICONDIRENTRY
	offset := 6 + len(newSet)*16
	for i, icon := range newSet {
		// it32 data always starts with a header of four zero-bytes
		// (tested all icns files in macOS 10.15.7 and macOS 11).
		// Usage unknown, the four zero-bytes can be any value and are quietly ignored.
		if string(icon.Type[:]) == "it32" && len(icon.Data) >= 4 {
			icon.Data = icon.Data[4:]
		}

		var w, h, s int

		if isPNG(icon.Data) {
			d = append(d, icon.Data)
			img, err := png.DecodeConfig(bytes.NewReader(icon.Data))
			if err != nil {
				return err
			}
			w, h, s = img.Width, img.Height, len(icon.Data)
		} else {
			decoded, hasA := false, 1
			var rgba *image.RGBA
			switch string(icon.Type[:]) {
			// 24-bit RGB
			case "is32", "il32", "ih32", "it32", "icp4", "icp5":
				if maskData, ok := maskMap[i]; ok {
					// 构造成ARGB格式
					newData := append([]byte("ARGB"), maskData.Data...)
					icon.Data = append(newData, icnsBRLDecode(icon.Data)...)
				} else {
					icon.Data = append([]byte("ARGB"), icnsBRLDecode(icon.Data)...)
					// 说明有没有透明度数据
					hasA = 0
				}
				decoded = true
			default:
			}

			if isARGB(icon.Data) {
				if decoded {
					icon.Data = icon.Data[4:]
				} else {
					icon.Data = icnsBRLDecode(icon.Data[4:])
				}
				pixles := len(icon.Data) / 4
				w := int(math.Sqrt(float64(pixles)))
				h = w

				rgba = image.NewRGBA(image.Rect(0, 0, w, h))
				for y := 0; y < h; y++ {
					for x := 0; x < w; x++ {
						no := (y*w + x)

						var alpha uint8
						if hasA > 0 {
							// 最前面是透明度数据
							alpha = icon.Data[no]
						} else {
							alpha = 0xFF
						}
						rgba.Set(x, y, color.RGBA{icon.Data[no+hasA*pixles], icon.Data[no+(1+hasA)*pixles], icon.Data[no+(2+hasA)*pixles], alpha})
					}
				}
			} else {
				img, _, err := image.Decode(bytes.NewReader(icon.Data))
				if err != nil {
					return err
				}

				rgba = image.NewRGBA(img.Bounds())
				draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
			}

			var buf bytes.Buffer
			png.Encode(&buf, rgba)
			d = append(d, buf.Bytes())

			w, h, s = rgba.Bounds().Dx(), rgba.Bounds().Dy(), buf.Len()
		}

		entries = append(entries, ICONDIRENTRY{
			IconCommon: IconCommon{
				Width:      uint8(w),
				Height:     uint8(h),
				Planes:     1,
				BitCount:   32,
				BytesInRes: uint32(s),
			},
			Offset: uint32(offset),
		})

		offset += s
	}

	return writeICO(w, ICONDIR{Type: 1, Count: uint16(len(iconSet))}, entries, d, cfg...)
}

const (
	SECTION_RESOURCES = ".rsrc"
	RT_ICON           = "3/"
	RT_GROUP_ICON     = "14/"
)

// resource holds the full name and data of a data entry in a resource directory structure.
// The name represents all 3 parts of the tree, separated by /, <type>/<name>/<language> with
// For example: "3/1/1033" for a resources with ID names, or "10/SOMERES/1033" for a named
// resource in language 1033.
type resource struct {
	Name string
	Data []byte
}

// Recursively parses a IMAGE_RESOURCE_DIRECTORY in slice b starting at position p
// building on path prefix. virtual is needed to calculate the position of the data
// in the resource
func parseDir(b []byte, p int, prefix string, addr uint32) []*resource {
	if prefix != "" && !strings.HasPrefix(prefix, RT_ICON) && !strings.HasPrefix(prefix, RT_GROUP_ICON) {
		return nil
	}

	le := binary.LittleEndian

	var res []*resource
	// Skip Characteristics, Timestamp, Major, Minor in the directory
	n := int(le.Uint16(b[p+12:p+14])) + int(le.Uint16(b[p+14:p+16]))

	// Iterate over all entries in the current directory record
	for i := 0; i < n; i++ {
		o := 8*i + p + 16
		name := int(le.Uint32(b[o : o+4]))
		offsetToData := int(le.Uint32(b[o+4 : o+8]))
		path := prefix
		if name&0x80000000 > 0 { // Named entry if the high bit is set in the name
			dirStr := name & 0x7FFFFFFF
			length := int(le.Uint16(b[dirStr : dirStr+2]))
			resID := make([]uint16, length)
			binary.Read(bytes.NewReader(b[dirStr+2:dirStr+2+length<<1]), le, resID)
			path += string(utf16.Decode(resID))
		} else { // ID entry
			path += strconv.Itoa(name)
		}

		if offsetToData&0x80000000 > 0 { // Ptr to other directory if high bit is set
			subdir := offsetToData & 0x7FFFFFFF

			// Recursively get the res from the sub dirs
			l := parseDir(b, subdir, path+"/", addr)
			res = append(res, l...)
			continue
		}

		// Leaf, ptr to the data entry. Read IMAGE_RESOURCE_DATA_ENTRY
		offset := int(le.Uint32(b[offsetToData : offsetToData+4]))
		length := int(le.Uint32(b[offsetToData+4 : offsetToData+8]))

		// The offset in IMAGE_RESOURCE_DATA_ENTRY is relative to the virual address.
		// Calculate the address in the file
		offset -= int(addr)

		// Add boundary checks to prevent panic
		if offset < 0 || offset+length > len(b) {
			continue
		}

		// Add resource to the list
		res = append(res, &resource{Name: path, Data: b[offset : offset+length]})
	}
	return res
}

// https://www.cnblogs.com/cswuyg/p/3603707.html
// https://www.cnblogs.com/cswuyg/p/3619687.html
// https://en.wikipedia.org/wiki/ICO_(file_format)#Header
type ICONDIR struct {
	Reserved uint16 // 保留字段，必须为0
	Type     uint16 // 图标类型，必须为1
	Count    uint16 // 图标数量
}

type IconCommon struct {
	Width      uint8  // 图标的宽度，以像素为单位
	Height     uint8  // 图标的高度，以像素为单位
	Color      uint8  // 色深，例如 16、256(0如果是256色)
	Reserved   uint8  // 保留字段
	Planes     uint16 // 颜色平面数
	BitCount   uint16 // 每个像素的位数
	BytesInRes uint32 // 图像数据的大小
}

type RESDIR struct {
	IconCommon
	ID uint16 // 图像数据的ID
}

type GRPICONDIR struct {
	ICONDIR
	Entries []RESDIR
}

type ICONDIRENTRY struct {
	IconCommon
	Offset uint32 // 图像数据的偏移量
}

func defaultICO(w io.Writer, peFile *pe.File, cfg ...Config) error {
	n := ""
	if peFile.FileHeader.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		n = "assets/DLL.ico"
	} else {
		// 如果没有资源段
		var subsystem uint16
		switch peFile.OptionalHeader.(type) {
		case *pe.OptionalHeader32:
			subsystem = peFile.OptionalHeader.(*pe.OptionalHeader32).Subsystem
		case *pe.OptionalHeader64:
			subsystem = peFile.OptionalHeader.(*pe.OptionalHeader64).Subsystem
		}

		switch subsystem {
		case pe.IMAGE_SUBSYSTEM_WINDOWS_CUI, pe.IMAGE_SUBSYSTEM_OS2_CUI, pe.IMAGE_SUBSYSTEM_POSIX_CUI:
			n = "assets/CUI.ico"
		default: // pe.IMAGE_SUBSYSTEM_WINDOWS_GUI, pe.IMAGE_SUBSYSTEM_WINDOWS_CE_GUI
			n = "assets/GUI.ico"
		}
	}

	iconData, _ := Asset(n)

	gid := GRPICONDIR{}
	rd := bytes.NewReader(iconData)
	binary.Read(rd, binary.LittleEndian, &gid.ICONDIR)
	entries := make([]ICONDIRENTRY, gid.Count)
	for i := uint16(0); i < gid.Count; i++ {
		binary.Read(rd, binary.LittleEndian, &entries[i])
	}

	var d [][]byte
	for i := uint16(0); i < gid.Count; i++ {
		d = append(d, iconData[entries[i].Offset:])
	}

	return writeICO(w, gid.ICONDIR, entries, d, cfg...)
}

/*
在 Windows 中，当匹配一个 EXE 文件的图标时，通常会选择其中的一个资源，
这个资源通常是包含在 PE 文件中的一组图标资源中的一个。
选择的资源不一定是具有最小 ID 的资源，而是根据一些规则进行选择。
Choosing an Icon: https://learn.microsoft.com/en-us/previous-versions/ms997538(v=msdn.10)?redirectedfrom=MSDN#choosing-an-icon
*/
func PE2ICO(w io.Writer, path string, cfg ...Config) error {
	// 解析PE文件
	peFile, err := pe.Open(path)
	if err != nil {
		return err
	}

	rsrc := peFile.Section(SECTION_RESOURCES)
	if rsrc == nil {
		return defaultICO(w, peFile, cfg...)
	}

	// 解析资源表
	resTable, err := rsrc.Data()
	if err != nil {
		return err
	}

	resources := parseDir(resTable, 0, "", rsrc.SectionHeader.VirtualAddress)
	idmap := make(map[uint16]*resource)
	gid := GRPICONDIR{}
	var grpIcons []*resource
	for _, r := range resources {
		if strings.HasPrefix(r.Name, RT_GROUP_ICON) {
			grpIcons = append(grpIcons, r)
		} else if strings.HasPrefix(r.Name, RT_ICON) {
			n := strings.Split(r.Name, "/")
			id, _ := strconv.ParseUint(n[1], 10, 64)
			idmap[uint16(id)] = r
		}
	}

	// 如果没有图标
	if len(grpIcons) <= 0 {
		return defaultICO(w, peFile, cfg...)
	}

	// 获取指定的图标
	var grpData []byte
	if len(cfg) > 0 {
		if cfg[0].Index != nil && *cfg[0].Index < 0 {
			// 如果是负数，那么尝试id
			if r, ok := idmap[uint16(-*cfg[0].Index)]; ok {
				return res2ICO(w, r.Data, cfg...)
			}
			return defaultICO(w, peFile, cfg...)
		}
		if cfg[0].Index == nil || int(*cfg[0].Index) >= len(grpIcons) {
			grpData = grpIcons[0].Data
		} else {
			grpData = grpIcons[*cfg[0].Index].Data
		}
	} else {
		grpData = grpIcons[0].Data
	}

	rd := bytes.NewReader(grpData)
	binary.Read(rd, binary.LittleEndian, &gid.ICONDIR)
	gid.Entries = make([]RESDIR, gid.Count)
	for i := uint16(0); i < gid.Count; i++ {
		binary.Read(rd, binary.LittleEndian, &gid.Entries[i])
	}

	// 如果没有图标
	if gid.Count <= 0 {
		return defaultICO(w, peFile, cfg...)
	}

	entries := make([]ICONDIRENTRY, gid.Count)
	var d [][]byte
	offset := binary.Size(gid.ICONDIR) + len(entries)*binary.Size(entries[0])
	for i := uint16(0); i < gid.Count; i++ {
		if r, ok := idmap[gid.Entries[i].ID]; ok {
			entries[i].IconCommon = gid.Entries[i].IconCommon
			entries[i].Offset = uint32(offset)

			offset += len(r.Data)
			d = append(d, r.Data)
		}
	}

	return writeICO(w, gid.ICONDIR, entries, d, cfg...)
}

// check 1bit FLAG of x,y coordinator
func f(d []byte, x, y, w, h int) byte {
	return d[(w>>3*((h-1)-y))+(x>>3)] >> uint(0x07-(x&0x07)) & 1
}

func convert16BitToARGB(value uint16, mask uint32) color.RGBA {
	return color.RGBA{
		uint8((uint32(value>>8&0xF8) * (mask >> 16)) >> 8),
		uint8((uint32(value>>3&0xFC) * (mask >> 8)) >> 8),
		uint8((uint32(value<<3&0xF8) * mask) >> 8),
		uint8(mask >> 24),
	}
}

func getMaskBit(d []byte, x, y, w, h int) uint32 {
	if len(d) > 0 && f(d, x, y, w, h) != 0 {
		return 0
	}
	return 0xFFFFFFFF
}

// https://stackoverflow.com/questions/16330403/get-hbitmaps-for-all-sizes-and-depths-of-a-file-type-icon-c
func res2BMP32(d []byte) *image.RGBA {
	var bmpHdr struct {
		Size            uint32 // The size of the header (in bytes)
		Width           int32  // The bitmap's width (in pixels)
		Height          int32  // The bitmap's height (in pixels)
		Planes          uint16 // The number of color planes (must be 1)
		BitCount        uint16 // The number of bits per pixel
		Compression     uint32 // The compression method being used
		SizeImage       uint32 // The image size (in bytes)
		XPelsPerMeter   int32  // The horizontal resolution (pixels per meter)
		YPelsPerMeter   int32  // The vertical resolution (pixels per meter)
		ColorsUsed      uint32 // The number of colors in the color palette
		ColorsImportant uint32 // The number of important colors used
	}
	binary.Read(bytes.NewReader(d), binary.LittleEndian, &bmpHdr)
	w, h, colors := int(bmpHdr.Width), int(bmpHdr.Height), int(bmpHdr.ColorsUsed)
	var bmp *image.RGBA
	if h >= w<<1 {
		bmp = image.NewRGBA(image.Rect(0, 0, w, h>>1))
	} else {
		bmp = image.NewRGBA(image.Rect(0, 0, w, h))
	}

	d = d[40:]

	var bitmask []byte
	switch bmpHdr.BitCount {
	case 32: // BGRA
		if h >= w<<1 {
			bitmask = d[w*w<<2:]
			h >>= 1
		}
		pixel := 0
		for yy := h - 1; yy > 0; yy-- {
			for xx := 0; xx < w; xx++ {
				mask := getMaskBit(bitmask, xx, yy, w, h)
				bmp.Set(xx, yy, color.RGBA{
					d[pixel<<2+2] & uint8(mask>>16),
					d[pixel<<2+1] & uint8(mask>>8),
					d[pixel<<2] & uint8(mask),
					d[pixel<<2+3] & uint8(mask>>24),
				})
				pixel++
			}
		}
	case 24: // BGR
		if h == w<<1 {
			bitmask = d[w*w*3:]
			h >>= 1
		}
		pixel := 0
		for yy := h - 1; yy > 0; yy-- {
			for xx := 0; xx < w; xx++ {
				mask := getMaskBit(bitmask, xx, yy, w, h)
				bmp.Set(xx, yy, color.RGBA{
					d[pixel*3+2] & uint8(mask>>16),
					d[pixel*3+1] & uint8(mask>>8),
					d[pixel*3] & uint8(mask),
					uint8(mask >> 24),
				})
				pixel++
			}
		}
	case 16:
		if h == w<<1 {
			bitmask = d[w*w<<1:]
			h >>= 1
		}
		pixel := 0
		for yy := h - 1; yy > 0; yy-- {
			for xx := 0; xx < w; xx++ {
				bmp.Set(xx, yy, convert16BitToARGB(
					binary.LittleEndian.Uint16(d[pixel<<1:]),
					getMaskBit(bitmask, xx, yy, w, h)))
				pixel++
			}
		}
	case 8:
		if colors > 256 || colors <= 0 {
			colors = 256
		}
		if h == w<<1 {
			bitmask = d[(colors<<2)+(w*w):]
			h >>= 1
		}
		pal := make([]color.RGBA, colors)
		for i := 0; i < colors; i++ {
			pal[i] = color.RGBA{d[i<<2+2], d[i<<2+1], d[i<<2], 0xFF} // RGBQUAD BGR
		}
		pixel := 0
		for yy := h - 1; yy > 0; yy-- {
			for xx := 0; xx < w; xx++ {
				if getMaskBit(bitmask, xx, yy, w, h) != 0 {
					bmp.Set(xx, yy, pal[d[(colors<<2)+pixel]])
				}
				pixel++
			}
		}
	case 4:
		if colors > 16 || colors <= 0 {
			colors = 16
		}
		if h == w<<1 {
			bitmask = d[(colors<<2)+(w*w>>1):]
			h >>= 1
		}
		pal := make([]color.RGBA, colors)
		for i := 0; i < colors; i++ {
			pal[i] = color.RGBA{d[i<<2+2], d[i<<2+1], d[i<<2], 0xFF} // RGBQUAD BGR
		}
		pixel := 0
		for yy := h - 1; yy > 0; yy-- {
			for xx := 0; xx < w; xx++ {
				if getMaskBit(bitmask, xx, yy, w, h) != 0 {
					if pixel&1 > 0 {
						bmp.Set(xx, yy, pal[d[(colors<<2)+(pixel>>1)]>>4])
					} else {
						bmp.Set(xx, yy, pal[d[(colors<<2)+(pixel>>1)]&0x0F])
					}
				}
				pixel++
			}
		}
	case 1:
		if colors > 2 {
			colors = 2
		}
		if colors <= 0 {
			colors = 2
		}
		pal := make([]color.RGBA, colors)
		for i := 0; i < colors; i++ {
			pal[i] = color.RGBA{d[i<<2+2], d[i<<2+1], d[i<<2], 0xFF} // RGBQUAD BGR
		}
		retColors := []color.RGBA{pal[0], {0x00, 0xFF, 0x00, 0xFF}, pal[1], {0x00, 0x00, 0xFF, 0xFF}}
		xorBits, andBits := d[(colors<<2):], d[(colors<<2)+(w*w>>3):]
		for yy := h - 1; yy > 0; yy-- {
			for xx := 0; xx < w; xx++ {
				bmp.Set(xx, yy, retColors[f(xorBits, xx, yy, w, h)<<1|f(andBits, xx, yy, w, h)])
			}
		}
	}

	return bmp
}

func res2ICO(w io.Writer, d []byte, cfg ...Config) error {
	if isPNG(d) {
		return IMG2ICO(w, bytes.NewReader(d), cfg...)
	}

	return img2ICO(w, zoomImg(res2BMP32(d), cfg...), cfg...)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func writeICO(w io.Writer, id ICONDIR, entries []ICONDIRENTRY, d [][]byte, cfg ...Config) error {
	// 如果wh设置了，选择合适的单张图标
	if len(cfg) > 0 && cfg[0].Width > 0 && cfg[0].Height > 0 {
		var m, wdiff, hdiff, bm int
		wdiff, hdiff = 0xFFFFF, 0xFFFFF
		for i, e := range entries {
			if e.BitCount >= uint16(bm) {
				bm = int(e.BitCount)
				var ws, hs int
				if e.Width <= 0 || e.Height <= 0 { // 超过大小的一定是PNG的
					img, _, _ := image.DecodeConfig(bytes.NewReader(d[i]))
					ws, hs = img.Width, img.Height
				} else {
					ws, hs = int(e.Width), int(e.Height)
				}
				if abs(ws-cfg[0].Width) <= wdiff && abs(hs-cfg[0].Height) <= hdiff {
					wdiff, hdiff = abs(ws-cfg[0].Width), abs(hs-cfg[0].Height)
					m = i
				}
			}
		}

		return res2ICO(w, d[m], cfg...)
	}

	// 没有设置，或者不是png格式
	if len(cfg) <= 0 || cfg[0].Format != "png" {
		err := binary.Write(w, binary.LittleEndian, id)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			err = binary.Write(w, binary.LittleEndian, entry)
			if err != nil {
				return err
			}
		}

		for _, d := range d {
			_, err = w.Write(d)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 如果是png格式，且wh未设置那么选择色值最多里面像素最大的
	var m, wm, hm, bm int
	for i, e := range entries {
		if e.BitCount >= uint16(bm) {
			bm = int(e.BitCount)
			var ws, hs int
			if e.Width <= 0 || e.Height <= 0 { // 超过大小的一定是PNG的
				img, _, _ := image.DecodeConfig(bytes.NewReader(d[i]))
				ws, hs = img.Width, img.Height
			} else {
				ws, hs = int(e.Width), int(e.Height)
			}
			if ws > wm && hs > hm {
				wm, hm = ws, hs
				m = i
			}
		}
	}

	_, err := w.Write(d[m])
	return err
}

func zoomImg(srcImg image.Image, cfg ...Config) *image.RGBA {
	if len(cfg) > 0 && (cfg[0].Width == srcImg.Bounds().Dx() || cfg[0].Height == srcImg.Bounds().Dy()) {
		switch srcImg := srcImg.(type) {
		case (*image.RGBA):
			return srcImg
		default:
			rgba := image.NewRGBA(srcImg.Bounds())
			draw.Draw(rgba, rgba.Bounds(), srcImg, image.Point{0, 0}, draw.Src)
			return rgba
		}
	}

	// 计算目标图片的纵横比
	srcRatio := float64(srcImg.Bounds().Dx()) / float64(srcImg.Bounds().Dy())

	// 计算缩放后的宽度和高度
	var width, height int
	if srcRatio > float64(cfg[0].Width)/float64(cfg[0].Height) {
		width = cfg[0].Width
		height = int(float64(width) / srcRatio)
	} else {
		height = cfg[0].Height
		width = int(float64(height) * srcRatio)
	}

	// 计算目标图片的起始位置
	x := (cfg[0].Width - width) >> 1
	y := (cfg[0].Height - height) >> 1

	// 使用nearest-neighbor算法缩放图像
	resizedImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resizedImg, resizedImg.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)

	// 将缩放后的图像绘制到目标图片上
	img := image.NewRGBA(image.Rect(0, 0, cfg[0].Width, cfg[0].Height))
	draw.Draw(img, image.Rect(x, y, x+width, y+height), resizedImg, image.Point{0, 0}, draw.Src)
	return img
}
