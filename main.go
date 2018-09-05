package main

import (
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"os"
	"context"
	"strings"
	"io/ioutil"
	"qiniupkg.com/x/log.v7"
	"path"
)

type Uploader struct {
	accessKey    string
	secretKey    string
	bucket       string
	filePath     string
	upToken      string
	formUploader *storage.FormUploader
}

func NewUploader(accessKey, secretKey, bucket, filePath string) (Uploader, error) {
	upload := Uploader{
		accessKey: accessKey,
		secretKey: secretKey,
		bucket:    bucket,
		filePath:  filePath,
	}
	err := upload.init()
	return upload, err
}

// Init uploader, generate up token and form uploader
func (u *Uploader) init() (err error) {
	err = nil
	// generate mac and upload token
	putPolicy := storage.PutPolicy{
		Scope: u.bucket,
	}
	mac := qbox.NewMac(u.accessKey, u.secretKey)
	u.upToken = putPolicy.UploadToken(mac)

	cfg := storage.Config{}
	// 空间对应的机房
	cfg.Zone = &storage.ZoneHuadong
	// 是否使用https域名
	cfg.UseHTTPS = false
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false
	// 构建表单上传的对象
	u.formUploader = storage.NewFormUploader(&cfg)

	return
}

func (u *Uploader) checkIsHiddenFile(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	} else {
		return false
	}
}

// Recursive upload local files in dir
func (u *Uploader) uploadLocalDir(dir string) (keys []string, err error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	for _, f := range files {
		// if is hidden file, continue
		if u.checkIsHiddenFile(f.Name()) {
			continue
		}
		log.Debug(path.Join(dir, f.Name()))
		absPath := path.Join(dir, f.Name())
		// if is dir, recursive loop to find files
		if f.IsDir() {
			var childKeys []string
			childKeys, err = u.uploadLocalDir(absPath)
			if err != nil {
				return
			}
			keys = append(keys, childKeys...)
		} else {
			// if is file, append to keys
			keys = append(keys, absPath)
		}
	}
	return
}

// Upload local file
func (u *Uploader) uploadLocalFile(file string) (key string, err error) {
	// check
	isDir, err := u.checkIsDir(file)
	if isDir {
		return
	}
	// extract upload name
	strs := strings.Split(file, "/")
	name := strs[len(strs)-1]
	// upload
	ret := storage.PutRet{}
	err = u.formUploader.PutFile(context.Background(), &ret, u.upToken, name, file, nil)
	if err != nil {
		return
	}
	key = ret.Key

	return
}

func (u *Uploader) checkIsDir(path string) (result bool, err error) {
	f, err := os.Stat(path)
	if err != nil {
		return
	}
	result = f.IsDir()
	return
}

// Upload images in markdown to qiniu and replace
func main() {
	//fromNet()
}

/*
func fromLocal() {
	localFile := "/Users/hgao/Downloads/lk_gouweiba.jpg"
	key := "lk_gouweiba.jpg"

	putPolicy := storage.PutPolicy{
		Scope: bucket,
	}
	mac := qbox.NewMac(accessKey, secretKey)
	upToken := putPolicy.UploadToken(mac)

	cfg := storage.Config{}
	// 空间对应的机房
	cfg.Zone = &storage.ZoneHuadong
	// 是否使用https域名
	cfg.UseHTTPS = false
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false

	// 构建表单上传的对象
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}

	// 可选配置
	//putExtra := storage.PutExtra{
	//	Params: map[string]string{
	//		"x:name": "github logo",
	//	},
	//}
	err := formUploader.PutFile(context.Background(), &ret, upToken, key, localFile, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(ret.Key, ret.Hash)
}

func fromNet() {
	bucket := "image"
	resURL := "http://devtools.qiniu.com/qiniu.png"

	mac := qbox.NewMac(accessKey, secretKey)

	cfg := storage.Config{
		// 是否使用https域名进行资源管理
		UseHTTPS: false,
	}
	// 指定空间所在的区域，如果不指定将自动探测
	// 如果没有特殊需求，默认不需要指定
	cfg.Zone = &storage.ZoneHuadong
	bucketManager := storage.NewBucketManager(mac, &cfg)

	// 指定保存的key
	fetchRet, err := bucketManager.Fetch(resURL, bucket, "qiniu.png")
	if err != nil {
		fmt.Println("fetch error,", err)
	} else {
		fmt.Println(fetchRet.String())
	}

	// 不指定保存的key，默认用文件hash作为文件名
	fetchRet, err = bucketManager.FetchWithoutKey(resURL, bucket)
	if err != nil {
		fmt.Println("fetch error,", err)
	} else {
		fmt.Println(fetchRet.String())
	}
}
*/
