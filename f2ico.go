package f2ico

import (
	"archive/zip"
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"gopkg.in/ini.v1"

	_ "image/gif"
	_ "image/jpeg"

	_ "github.com/cbeer/jpeg2000"
	"github.com/tmc/icns"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

type Config struct {
	Format string // png or ico(default)
	Width  uint16 // 0 for all
	Height uint16 // 0 for all
	Index  uint16 // 0 default
}

var apkRegex = regexp.MustCompile(`^res/mipmap-((:?x{0,3}h)|[ml])dpi[^\/]*/.*\.png$`)

var apkDensityWeight = map[string]int8{
	"xxxh": 6,
	"xxh":  5,
	"xh":   4,
	"h":    3,
	"m":    2,
	"l":    1,
}

func F2ICO(w io.Writer, path string, cfg ...Config) error {
	ext := strings.ToLower(filepath.Ext(path))[1:]
	switch ext {
	case "exe", "dll":
		return PE2ICO(w, path, cfg...)
	}

	switch ext {
	case "ico", "icns", "bmp", "gif", "jpg", "jpeg", "png", "tiff":
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		switch ext {
		case "ico": // FIXME：如果只需要其中的一种尺寸
			_, err = io.Copy(w, f)
			return err
		case "icns":
			return ICNS2ICO(w, f, cfg...)
		case "bmp", "gif", "jpg", "jpeg", "png", "tiff":
			return IMG2ICO(w, f, cfg...)
		}

	case "apk":
		r, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer r.Close()

		/*
			APK 文件实际上是一个 ZIP 压缩文件，其中包含了应用程序的各种资源和文件。应用程序的图标通常存放在以下路径：

			res/mipmap-<density>(-...)/ic_launcher.png
			在这个路径中，<density> 是密度相关的标识符，代表了不同分辨率的图标。常见的标识符包括 hdpi、xhdpi、xxhdpi 等。不同密度的图标可以提供给不同密度的屏幕使用，以保证图标在不同设备上显示时具有良好的清晰度和质量。

			注意：实际的路径可能会因应用程序的结构而有所不同，上述路径仅为一般情况。
		*/
		var maxWeight int8
		var maxF *zip.File
		for _, f := range r.File {
			// 检查文件名
			if match := apkRegex.FindStringSubmatch(f.Name); match != nil {
				// 提取density信息
				if apkDensityWeight[match[1]] > maxWeight {
					maxF = f
					maxWeight = apkDensityWeight[match[1]]
				}
			}
		}
		if maxF != nil {
			// 打开文件
			rc, err := maxF.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			return IMG2ICO(w, rc, cfg...)
		}
	}

	return errors.New("conversion failed")
}

type Info struct {
	IconFile  string
	IconIndex uint16
	FilePath  string
}

func GetInfo(path string) (info Info, err error) {
	ext := strings.ToLower(filepath.Ext(path))[1:]

	var f *ini.File
	switch ext {
	case "inf", "ini", "desktop":
		f, err = ini.Load(path)
		if err != nil {
			return info, err
		}

	// *.app目录
	case "app":
		/*
		*.app/Contents/Resources/AppIcon.icns
		 */
		info.IconFile = filepath.Join(path, "Contents/Resources/AppIcon.icns")
		return
	case "exe", "dll", "ico", "bmp", "gif", "jpg", "jpeg", "png", "tiff", "icns", "dmg", "apk":
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
	case "inf":
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
	case "ini":
		/*
			在 Windows 操作系统中，desktop.ini 文件用于自定义文件夹的外观和行为。您可以在文件夹中创建 desktop.ini 文件，并在其中指定如何显示该文件夹的图标。

			要在 desktop.ini 文件中定义图标，可以使用 IconFile 和 IconIndex 字段。下面是一个示例 desktop.ini 文件的基本结构：

			[.ShellClassInfo]
			IconFile=path\to\icon.ico
			IconIndex=0

			IconFile 字段指定要用作文件夹图标的图标文件的路径。这可以是包含图标的 .ico 文件，也可以是 .exe 或 .dll 文件，其中包含一个或多个图标资源。
			IconIndex 字段指定要在 IconFile 中使用的图标的索引。如果 IconFile 是 .ico 文件，则索引从0开始，表示图标在文件中的位置。如果 IconFile 是 .exe 或 .dll 文件，则索引表示图标资源的标识符。
			完成后，您可以将 desktop.ini 文件放置在所需文件夹中，并在 Windows 资源管理器中刷新文件夹，以查看所指定的图标。
		*/
		section, err := f.GetSection(".ShellClassInfo")
		if err != nil {
			return info, err
		}

		info.IconFile = section.Key("IconFile").String()
		info.IconIndex = uint16(section.Key("IconFile").MustUint(0))
	case "desktop":
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
		info.FilePath = section.Key("Exec").String()
	}
	return
}

func IMG2ICO(w io.Writer, r io.Reader, cfg ...Config) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var buf bytes.Buffer
	png.Encode(&buf, rgba)

	if len(cfg) <= 0 || cfg[0].Format != "png" {
		err = binary.Write(w, binary.LittleEndian, &ICONDIR{Type: 1, Count: 1})
		if err != nil {
			return err
		}

		err = binary.Write(w, binary.LittleEndian, &ICONDIRENTRY{
			IconCommon: IconCommon{
				Width:      uint8(rgba.Bounds().Dx()),
				Height:     uint8(rgba.Bounds().Dy()),
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

func ICNSBRLDecode(data []byte) (ret []byte) {
	for i := 0; i < len(data); {
		b := data[i]
		if b < 0x80 {
			cnt := int(b) + 1
			if i+cnt >= len(data) {
				break
			}
			ret = append(ret, data[i+1:i+1+cnt]...)
			i += cnt + 1
		} else {
			cnt := int(b) - 0x80 + 3
			if i+1 >= len(data) {
				break
			}
			tb := data[i+1]
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

func isPNG(data []byte) bool {
	return len(data) > 8 && bytes.Compare(data, []byte{'\x89', 'P', 'N', 'G', '\r', '\n', '\x1a', '\n'}) == 0
}

func isARGB(data []byte) bool {
	return len(data) > 4 && string(data[:4]) == "ARGB"
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
			break
		default:
			newSet = append(newSet, icon)
		}
	}

	var data [][]byte
	var entries []*ICONDIRENTRY
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
			data = append(data, icon.Data)
			img, err := png.DecodeConfig(bytes.NewReader(icon.Data))
			if err != nil {
				return err
			}
			w, h, s = img.Width, img.Height, len(icon.Data)
		} else {
			needDecode := true
			hasAlphaChan := 1
			var rgba *image.RGBA
			switch string(icon.Type[:]) {
			// 24-bit RGB
			case "is32", "il32", "ih32", "it32", "icp4", "icp5":
				if maskData, ok := maskMap[i]; ok {
					// 构造成ARGB格式
					newData := append([]byte("ARGB"), maskData.Data...)
					icon.Data = append(newData, ICNSBRLDecode(icon.Data)...)
				} else {
					icon.Data = append([]byte("ARGB"), ICNSBRLDecode(icon.Data)...)
					// 说明有没有透明度数据
					hasAlphaChan = 0
				}
				needDecode = false
			default:
			}

			if isARGB(icon.Data) {
				if needDecode {
					icon.Data = ICNSBRLDecode(icon.Data[4:])
				} else {
					icon.Data = icon.Data[4:]
				}
				pixles := len(icon.Data) / 4
				w := int(math.Sqrt(float64(pixles)))
				h = w

				rgba = image.NewRGBA(image.Rect(0, 0, w, h))
				for y := 0; y < h; y++ {
					for x := 0; x < w; x++ {
						no := (y*w + x)

						var alpha uint8
						if hasAlphaChan > 0 {
							// 最前面是透明度数据
							alpha = icon.Data[no]
						} else {
							alpha = 0xFF
						}
						rgba.Set(x, y, color.RGBA{
							R: icon.Data[no+hasAlphaChan*pixles],
							G: icon.Data[no+(1+hasAlphaChan)*pixles],
							B: icon.Data[no+(2+hasAlphaChan)*pixles],
							A: alpha,
						})
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
			data = append(data, buf.Bytes())

			w, h, s = rgba.Bounds().Dx(), rgba.Bounds().Dy(), buf.Len()
		}

		entries = append(entries, &ICONDIRENTRY{
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

	err = binary.Write(w, binary.LittleEndian, &ICONDIR{Type: 1, Count: uint16(len(iconSet))})
	if err != nil {
		return err
	}

	for i := 0; i < len(entries); i++ {
		err = binary.Write(w, binary.LittleEndian, entries[i])
		if err != nil {
			return err
		}
	}

	for _, d := range data {
		_, err = w.Write(d)
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	SECTION_RESOURCES = ".rsrc"
	RT_ICON           = "3/"
	RT_GROUP_ICON     = "14/"
)

// Resource holds the full name and data of a data entry in a resource directory structure.
// The name represents all 3 parts of the tree, separated by /, <type>/<name>/<language> with
// For example: "3/1/1033" for a resources with ID names, or "10/SOMERES/1033" for a named
// resource in language 1033.
type Resource struct {
	Name string
	Data []byte
}

// Recursively parses a IMAGE_RESOURCE_DIRECTORY in slice b starting at position p
// building on path prefix. virtual is needed to calculate the position of the data
// in the resource
func parseDir(b []byte, p int, prefix string, virtual uint32) []*Resource {
	if prefix != "" && !strings.HasPrefix(prefix, RT_GROUP_ICON) && !strings.HasPrefix(prefix, RT_ICON) {
		return nil
	}

	var resources []*Resource

	// Skip Characteristics, Timestamp, Major, Minor in the directory

	numberOfNamedEntries := int(binary.LittleEndian.Uint16(b[p+12 : p+14]))
	numberOfIdEntries := int(binary.LittleEndian.Uint16(b[p+14 : p+16]))
	n := numberOfNamedEntries + numberOfIdEntries

	// Iterate over all entries in the current directory record
	for i := 0; i < n; i++ {
		o := 8*i + p + 16
		name := int(binary.LittleEndian.Uint32(b[o : o+4]))
		offsetToData := int(binary.LittleEndian.Uint32(b[o+4 : o+8]))
		path := prefix
		if name&0x80000000 > 0 { // Named entry if the high bit is set in the name
			dirString := name & 0x7FFFFFFF
			length := int(binary.LittleEndian.Uint16(b[dirString : dirString+2]))
			c := b[dirString+2 : dirString+2+length*2]
			var r []uint16
			for {
				if len(c) < 2 {
					break
				}
				v := binary.LittleEndian.Uint16(c[0:2])
				r = append(r, v)
				c = c[2:]
			}
			path += string(utf16.Decode(r))
		} else { // ID entry
			path += strconv.Itoa(name)
		}

		if offsetToData&0x80000000 > 0 { // Ptr to other directory if high bit is set
			subdir := offsetToData & 0x7FFFFFFF

			// Recursively get the resources from the sub dirs
			l := parseDir(b, subdir, path+"/", virtual)
			resources = append(resources, l...)
			continue
		}

		// Leaf, ptr to the data entry. Read IMAGE_RESOURCE_DATA_ENTRY
		offset := int(binary.LittleEndian.Uint32(b[offsetToData : offsetToData+4]))
		length := int(binary.LittleEndian.Uint32(b[offsetToData+4 : offsetToData+8]))

		// The offset in IMAGE_RESOURCE_DATA_ENTRY is relative to the virual address.
		// Calculate the address in the file
		offset -= int(virtual)
		data := b[offset : offset+length]

		// Add Resource to the list
		resources = append(resources, &Resource{Name: path, Data: data})
	}
	return resources
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

type ICOFILEHEADER struct {
	ICONDIR
	Entries []ICONDIRENTRY
}

func DefaultICO(w io.Writer, peFile *pe.File) error {
	// 如果没有资源段
	var subsystem uint16
	switch peFile.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		subsystem = peFile.OptionalHeader.(*pe.OptionalHeader32).Subsystem
	case *pe.OptionalHeader64:
		subsystem = peFile.OptionalHeader.(*pe.OptionalHeader64).Subsystem
	}

	n := ""
	switch subsystem {
	case pe.IMAGE_SUBSYSTEM_WINDOWS_CUI, pe.IMAGE_SUBSYSTEM_OS2_CUI, pe.IMAGE_SUBSYSTEM_POSIX_CUI:
		n = "assets/CUI.ico"
	default: // pe.IMAGE_SUBSYSTEM_WINDOWS_GUI, pe.IMAGE_SUBSYSTEM_WINDOWS_CE_GUI
		n = "assets/GUI.ico"
	}

	d, _ := Asset(n)
	_, err := w.Write(d)
	return err
}

/*
在 Windows 中，当匹配一个 EXE 文件的图标时，通常会选择其中的一个资源，这个资源通常是包含在 PE 文件中的一组图标资源中的一个。选择的资源不一定是具有最小 ID 的资源，而是根据一些规则进行选择。
Choosing an Icon: https://learn.microsoft.com/en-us/previous-versions/ms997538(v=msdn.10)?redirectedfrom=MSDN#choosing-an-icon
*/
// 支持格式exe dll scr icl
func PE2ICO(w io.Writer, path string, cfg ...Config) error {
	// 解析PE文件
	peFile, err := pe.Open(path)
	if err != nil {
		return err
	}

	rsrc := peFile.Section(SECTION_RESOURCES)
	if rsrc == nil {
		return DefaultICO(w, peFile)
	}

	// 解析资源表
	resTable, err := rsrc.Data()
	if err != nil {
		return err
	}

	resources := parseDir(resTable, 0, "", rsrc.SectionHeader.VirtualAddress)
	idmap := make(map[uint16]*Resource)
	gid := GRPICONDIR{}
	var grpIcons []*Resource
	for _, r := range resources {
		if strings.HasPrefix(r.Name, RT_GROUP_ICON) {
			grpIcons = append(grpIcons, r)
		} else if strings.HasPrefix(r.Name, RT_ICON) {
			// FIXME：如果只需要其中的一种尺寸
			n := strings.Split(r.Name, "/")
			id, _ := strconv.ParseUint(n[1], 10, 64)
			idmap[uint16(id)] = r
		}
	}

	// 如果没有图标，有资源段说明是GUI
	if len(grpIcons) <= 0 {
		return DefaultICO(w, peFile)
	}

	// 获取指定的图标
	if len(cfg) > 0 && cfg[0].Index > 0 {
		if int(cfg[0].Index) >= len(grpIcons) {
			cfg[0].Index = 0
		}
		r := grpIcons[cfg[0].Index]
		rd := bytes.NewReader(r.Data)
		binary.Read(rd, binary.LittleEndian, &gid.ICONDIR)
		gid.Entries = make([]RESDIR, gid.Count)
		for i := uint16(0); i < gid.Count; i++ {
			binary.Read(rd, binary.LittleEndian, &gid.Entries[i])
		}
	}

	// 如果没有图标，有资源段说明是GUI
	if gid.Count <= 0 {
		return DefaultICO(w, peFile)
	}

	// FIXME：如果输出PNG
	err = binary.Write(w, binary.LittleEndian, gid.ICONDIR)
	if err != nil {
		return err
	}

	entries := make([]ICONDIRENTRY, gid.Count)
	offset := binary.Size(gid.ICONDIR) + len(entries)*binary.Size(entries[0])
	for i := uint16(0); i < gid.Count; i++ {
		if r, ok := idmap[gid.Entries[i].ID]; ok {
			entries[i].IconCommon = gid.Entries[i].IconCommon
			entries[i].Offset = uint32(offset)

			offset += len(r.Data)

			err = binary.Write(w, binary.LittleEndian, entries[i])
			if err != nil {
				return err
			}
		}
	}

	for i := uint16(0); i < gid.Count; i++ {
		if r, ok := idmap[gid.Entries[i].ID]; ok {
			_, err = w.Write(r.Data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
