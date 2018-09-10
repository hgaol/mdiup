package main

import (
	"io/ioutil"
	"regexp"
	"os"
	"strings"
	"net/url"
	"path"
	"qiniupkg.com/x/log.v7"
)

type ImageType int

const (
	FROM_NET   ImageType = 0
	FROM_LOCAL ImageType = 1
	FROM_QINIU ImageType = 2
)

var (
	mdImageReg = regexp.MustCompile("!\\[.*\\]\\(.*\\)")
)

type MarkdownUp struct {
	// uploader is a tool to upload images in markdown to qiniu
	up *Uploader
	// markdown file or dir path
	path string
}

func contains(list []string, str string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

func NewMarkdownUp(path, accessKey, secretKey, bucket string, allowImgSuffix []string) MarkdownUp {
	uploader := NewUploader(accessKey, secretKey, bucket, allowImgSuffix)
	uploader.init()
	return MarkdownUp{
		up:   uploader,
		path: path,
	}
}

func (mu *MarkdownUp) upload() {
	mdFiles := mu.getMdFiles()
	for _, mdFile := range mdFiles {
		// 回滚，不进行后续操作
		if rollback {
			mu.rollback(mdFile)
			continue
		}
		log.Info("file: " + mdFile)
		dat, err := ioutil.ReadFile(mdFile)
		if err != nil {
			panic(err)
		}
		fileStr := string(dat)
		// 1. 备份
		mu.backup(mdFile)
		// 2. 找出其所有图片
		images := mu.findAllImages(fileStr)
		imgMap := mu.uploadImages(images)
		// replace
		target := mu.replace(fileStr, imgMap)
		// write file
		ioutil.WriteFile(mdFile, []byte(target), 664)
	}
	//fmt.Println(mdFiles)
}

func (mu *MarkdownUp) backup(filePath string) {
	fileName := filePath[(strings.LastIndex(filePath, "/"))+1:]
	backupFile := path.Join(home, fileName)
	_, err := os.Stat(backupFile)
	if backup || os.IsNotExist(err) {
		// copy file
		input, err := ioutil.ReadFile(filePath)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(backupFile, input, 0644)
		if err != nil {
			panic(err)
		}
	}
}

// Find all images in a file
func (mu *MarkdownUp) findAllImages(str string) (ret []string) {
	imgs := mdImageReg.FindAllString(str, -1)
	for _, img := range imgs {
		imgUrl := img[strings.LastIndex(img, "(")+1 : strings.LastIndex(img, ")")]
		if contains(ret, imgUrl) {
			continue
		}
		ret = append(ret, imgUrl)
	}
	return
}

func (mu *MarkdownUp) uploadImages(images []string) map[string]string {
	ret := make(map[string]string)
	// todo
	for _, image := range images {
		imgType := mu.imageType(image)
		var err error
		var hash string
		switch imgType {
		case FROM_LOCAL:
			hash, err = mu.up.uploadLocalFileWithoutKey(image)
		case FROM_NET:
			hash, err = mu.up.uploadNetWithoutKey(image)
		case FROM_QINIU:
			continue
		}
		if err != nil {
			panic(err)
		}
		ret[image] = hash
		log.Infof("[uploaded] from: %v to: %v", image, hash)
	}
	return ret
}

func (mu *MarkdownUp) imageType(imageUrl string) ImageType {
	if strings.HasPrefix(imageUrl, "/") {
		return FROM_LOCAL
	} else {
		u, err := url.Parse(imageUrl)
		if err != nil {
			panic(err)
		}
		if mu.isQiniuDomain(u) {
			return FROM_QINIU
		} else {
			return FROM_NET
		}
	}
}

func (mu *MarkdownUp) isQiniuDomain(url *url.URL) bool {
	isHttpOrHttps := url.Scheme == "http" || url.Scheme == "https"
	domainUrl, err := url.Parse(mu.up.domain)
	if err != nil {
		panic(err)
	}
	isQiniuHost := domainUrl.Host == url.Host
	return isHttpOrHttps && isQiniuHost
}

// Get markdown files
func (mu *MarkdownUp) getMdFiles() (mdFiles []string) {
	f, err := os.Stat(mu.path)
	if err != nil {
		panic(err)
	}
	if f.IsDir() {
		mdFiles, err = listDirFiles(mu.path, mdSuffix)
		if err != nil {
			panic(err)
		}
	} else {
		mdFiles = []string{mu.path}
	}
	return
}

func (mu *MarkdownUp) replace(data string, imgMap map[string]string) string {
	ret := data
	for key, val := range imgMap {
		reg, err := regexp.Compile(key)
		if err != nil {
			panic(err)
		}
		ret = reg.ReplaceAllString(ret, mu.decorateWithMarkImage(val))
	}
	return ret
}

func (mu *MarkdownUp) decorateWithMarkImage(src string) string {
	u, err := url.Parse(mu.up.domain)
	if err != nil {
		panic(err)
	}
	u.Path = path.Join(u.Path, src)
	return u.String()
}

func (mu *MarkdownUp) rollback(filePath string) {
	fileName := filePath[(strings.LastIndex(filePath, "/"))+1:]
	backupFile := path.Join(home, fileName)
	log.Info("roll back file: " + filePath)
	// copy file, backup to file path
	input, err := ioutil.ReadFile(backupFile)
	if err != nil {
		log.Error(err)
	}
	err = ioutil.WriteFile(filePath, input, 0644)
	if err != nil {
		log.Error(err)
	}
}