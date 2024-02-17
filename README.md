<p align="center">
  <a href="https://orcastor.github.io/doc/">
    <img src="https://orcastor.github.io/doc/logo.svg">
  </a>
</p>

<h1 align="center"><strong>🔬 FileICOn</strong> <a href="https://github.com/orcastor/addon-previewer">@orcastor-previewer</a></h1>

### 支持文件

- 图片（bmp、gif、jpg、jpeg、jp2、jpeg2000、png、tiff）
- 图标（![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/WIN.png)ico、![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/MAC.png)icns）
- ![](https://raw.githubusercontent.com/drag-and-publish/operating-system-logos/master/src/16x16/WIN.png)Windows可执行文件（exe、dll、mui、mun）
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
- [x] dll加载不到图标问题
  > 答: 在早期的 Windows 版本中，图标资源文件嵌入到目录中的某些 DLL 中C:\Windows\System32。自 Windows 10 版本 1903 起，它们已重新定位到： C:\Windows\SystemResources. 现在这些文件有一个新的扩展名，.mun而不是.mui （仍然存在于system32和syswow64子文件夹中。
- [x] 低于256宽度图标格式转换为PNG的支持（先转换为32位位图）
- [x] 支持获取准确的高度（BITMAPINFOHEADER中大小为48x96，实际数据只有48x48）
- [x] 支持裁剪掉透明边缘（说明是48x48的位图，实际只有32x32是不透明的）

### 如果要更新assets下的默认图标

#### 安装 go-bindata 工具：
> go install -u github.com/go-bindata/go-bindata/...

#### 使用 go-bindata 将资源文件转换为 Go 代码：
> go-bindata -o assets.go -pkg fico assets/...