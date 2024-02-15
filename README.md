<p align="center">
  <a href="https://orcastor.github.io/doc/">
    <img src="https://orcastor.github.io/doc/logo.svg">
  </a>
</p>

<h1 align="center"><strong>🔬 f2ico</strong> <a href="https://github.com/orcastor/addon-previewer">@orcastor-previewer</a></h1>

### 支持文件

- 图片（bmp、gif、jpg、jpeg、jp2、jpeg2000、png、tiff）
- 图标（![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/WIN.png)ico、![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/MAC.png)icns）
- ![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/WIN.png)Windows可执行文件（exe、dll）
- ![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/LIN.png)Linux可执行文件（\*.desktop【\*.AppImage、\*.run】）
- ![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/AND.png)apk包
- ![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/WIN.png)文件夹图标（autorun.inf、desktop.ini）
- ![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/MAC.png)MacOSX程序（\*.app）

### 开发进度

- [x] 获取位置和获取图标方法剥离
- [x] 支持获取png格式的图标
- [x] PE文件无图标的默认图标逻辑
- [x] PE文件获取图标的index逻辑
- [x] 支持icns转换ico逻辑
- [x] 指定尺寸缩放逻辑
- [x] 指定尺寸图标匹配逻辑
- [ ] dll加载不到图标问题

### 如果要更新assets下的默认图标

#### 安装 go-bindata 工具：
> go install -u github.com/go-bindata/go-bindata/...

#### 使用 go-bindata 将资源文件转换为 Go 代码：
> go-bindata -o assets.go -pkg f2ico assets/...