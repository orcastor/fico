# f2ico
🔬【文件图标提取】从文件或目录提取图标（支持图片[bmp\/gif\/jpg\/jpeg\/png\/tiff]、ico、icns、pe[exe\/dll\/scr\/icl]、dmg、apk、autorun.inf、desktop.ini、desktop[AppImage\/run]、\*.app）Extract icon from file or directory.

- [x] 获取位置和获取图标方法剥离
- [x] 支持获取png格式的图标
- [x] PE文件无图标的默认图标逻辑
- [ ] PE文件获取图标的index逻辑
- [ ] ICNS逻辑
- [ ] 指定尺寸图标匹配逻辑

#### 安装 go-bindata 工具：
> go install -u github.com/go-bindata/go-bindata/...

#### 使用 go-bindata 将资源文件转换为 Go 代码：
> go-bindata -o assets.go -pkg f2ico assets/...