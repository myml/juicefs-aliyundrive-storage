# 阿里云盘 + juicefs

juicefs 是一个高性能网络文件系统，支持 完备的 POSIX 兼容性，支持提供 S3 网关并且可以进行透明加密。

阿里云盘是阿里巴巴推出的个人网络网盘，免费的大容量空间（我有内侧送永久 1T 存储），并且声称未来永不限速度。

这个 go 插件可以让 juicefs 存储数据到阿里云盘中，配合 juicefs 的透明加密，也无需担心云存储厂商读取你的文件。

目前已经有一些开源的阿里云盘 webdav 服务，相对于 webdav，juicefs 提供更完整的 POSIX 兼容，更容易在 linux 挂载使用。

## 局限性

由于 juicefs 是分块存储数据的，在 juicefs 存储的文件会被分割成 4M 的数据库，使用官方的客户端无法直接使用 juicefs 存储的文件。默认只是进行分块，如果不想让阿里偷看你的文件，需要使用 juicefs 的 AES 加密功能。
