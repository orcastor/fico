package f2ico

import (
	"archive/zip"
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"gopkg.in/ini.v1"

	_ "image/gif"
	_ "image/jpeg"

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
	case "exe", "dll", "scr", "icl":
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

	case "dmg", "apk":
		r, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer r.Close()

		switch ext {
		case "dmg":
			/*
				在 macOS 的 DMG（Disk Image）文件中，图标文件通常存放在.VolumeIcon.icns 文件中。

				.VolumeIcon.icns 文件：.VolumeIcon.icns 文件是存储在 DMG 文件中的磁盘图标文件。您可以通过创建一个包含所需图标的 .icns 文件，并将其命名为 .VolumeIcon.icns，然后将其添加到 DMG 文件中。这样，当用户挂载 DMG 文件时，磁盘会显示指定的图标。
			*/
			for _, f := range r.File {
				// 检查文件名是否是.VolumeIcon.icns
				if strings.HasSuffix(f.Name, ".VolumeIcon.icns") {
					// 打开文件
					rc, err := f.Open()
					if err != nil {
						return err
					}
					defer rc.Close()

					return ICNS2ICO(w, rc, cfg...)
				}
			}
		case "apk":
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
	case "exe", "dll", "scr", "icl", "ico", "bmp", "gif", "jpg", "jpeg", "png", "tiff", "icns", "dmg", "apk":
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

func ICNS2ICO(w io.Writer, r io.Reader, cfg ...Config) error {
	// TODO
	return nil
}

const (
	RT_ICON       = 3
	RT_GROUP_ICON = 14
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
			dirString := name & (0x80000000 - 1)
			length := int(binary.LittleEndian.Uint16(b[dirString : dirString+2]))
			b = b[dirString+2 : dirString+2+length*2]
			var r []uint16
			for {
				if len(b) < 2 {
					break
				}
				v := binary.LittleEndian.Uint16(b[0:2])
				r = append(r, v)
				b = b[2:]
			}
			path += string(utf16.Decode(r))
		} else { // ID entry
			path += strconv.Itoa(name)
		}

		if offsetToData&0x80000000 > 0 { // Ptr to other directory if high bit is set
			subdir := offsetToData & (0x80000000 - 1)

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

	rsrc := peFile.Section(".rsrc")
	if rsrc == nil {
		return errors.New("not Windows GUI executable, there's no icon resource")
	}

	// 解析资源表
	resTable, err := rsrc.Data()
	if err != nil {
		return err
	}

	resources := parseDir(resTable, 0, "", rsrc.SectionHeader.VirtualAddress)
	idmap := make(map[uint16]*Resource)
	gid := GRPICONDIR{}
	for _, r := range resources {
		if strings.HasPrefix(r.Name, "14/") {
			rd := bytes.NewReader(r.Data)
			binary.Read(rd, binary.LittleEndian, &gid.ICONDIR)
			gid.Entries = make([]RESDIR, gid.Count)
			for i := uint16(0); i < gid.Count; i++ {
				binary.Read(rd, binary.LittleEndian, &gid.Entries[i])
			}
		}
		// FIXME：如果只需要其中的一种尺寸
		if strings.HasPrefix(r.Name, "3/") {
			n := strings.Split(r.Name, "/")
			id, _ := strconv.ParseUint(n[1], 10, 64)
			idmap[uint16(id)] = r
		}
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
