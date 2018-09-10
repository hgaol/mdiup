# mdiup

mdiup(markdown images uploader)是一个用来将markdown中的图片上传至图床的小工具（目前支持七牛云）。



## Build

```shell
go get -u github.com/hgaol/mdiup
```

## Usage

```shell
# 也可以为了方便，将$GOPATH/bin设置在PATH路径下
Usage: $GOPATH/bin/mdiup [flags] markdown_file_path.
flags:
  -backup
    	set true to backup markdown when upload
  -home string
    	mdiup home directory (default "/Users/youname/.mdiup")
  -log string
    	logging level (default "info")
  -rollback
    	whether rollback markdown files
```

说明：

home - mdiup的家目录地址，`config.json` 文件和备份文件都在这里寻找备份

backup - 是否备份原始文件，默认第一次转换时备份，之后备份（其实是根据home下有无该文件）

log - 设置log等级，Debug, Info, Warn, Error, Panic, Fatal

rollback - 是否回滚，会查找home下的该文件，有则替换目标路径下的markdown文件



## Screenshot

![image-20180910133548355](http://pebbx585u.bkt.clouddn.com/FpVOW6w7wva1Dlmqem-itDaaRpSe)


