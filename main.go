package main

import (
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"os"
	"context"
	"strings"
	"io/ioutil"
	"github.com/qiniu/x/log.v7"
	"flag"
	"path"
	"fmt"
	"encoding/json"
)

var (
	mdSuffix   = []string{".md", ".markdown"}
	imgSuffix  = []string{".jpg", ".png", ".ico"}
	configFile = "config.json"
	config     = Config{}
	// command line flags
	// log等级，mdiup家目录地址，默认$HOME/.mdiup
	logLevel, home string
	// 是否备份，是否回滚
	backup, rollback bool
	// file or dir path
	filePath string
)

func init() {
	parseArgs()
	makeHomeDir()
	parseConfigFile()
	setLogLevel()
}

func setLogLevel() {
	//level: 0(Debug), 1(Info), 2(Warn), 3(Error), 4(Panic), 5(Fatal)
	switch logLevel {
	case "debug", "DEBUG":
		log.Std.Level = 0
	case "info", "INFO":
		log.Std.Level = 1
	case "warn", "WARN":
		log.Std.Level = 2
	case "error", "ERROR":
		log.Std.Level = 3
	case "panic", "PANIC":
		log.Std.Level = 4
	case "fatal", "FATAL":
		log.Std.Level = 5
	default:
		return
	}
}

func parseConfigFile() {
	conf, err := ioutil.ReadFile(path.Join(home, configFile))
	if err != nil {
		log.Fatal(err)
	}
	if json.Unmarshal(conf, &config); err != nil {
		log.Fatal(err)
	}
}

func parseArgs() {
	flag.StringVar(&logLevel, "log", "info", "logging level")
	flag.StringVar(&home, "home", path.Join(os.Getenv("HOME"), ".mdiup"), "mdiup home directory")
	flag.BoolVar(&backup, "backup", false, "set true to backup markdown when upload")
	flag.BoolVar(&rollback, "rollback", false, "whether rollback markdown files")
	setupFlags(flag.CommandLine)
	flag.Parse()
	filePath = flag.Arg(0)
}

func makeHomeDir() {
	if _, err := os.Stat(home); os.IsNotExist(err) {
		// make dir
		os.Mkdir(home, 0755)
	}
}

func setupFlags(f *flag.FlagSet) {
	f.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] markdown_file_path.\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "flags: \n")
		flag.PrintDefaults()
	}
}

// Check if is hidden file
func checkIsHiddenFile(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	} else {
		return false
	}
}

// Recursive loop dir to list files with definite suffix
func listDirFiles(dir string, suffix []string) (fileAbsPaths []string, err error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	for _, f := range files {
		// if is hidden file, continue
		if checkIsHiddenFile(f.Name()) {
			continue
		}
		log.Debug(path.Join(dir, f.Name()))
		absPath := path.Join(dir, f.Name())
		// if is dir, recursive loop to find files
		if f.IsDir() {
			var childKeys []string
			childKeys, err = listDirFiles(absPath, suffix)
			if err != nil {
				return
			}
			fileAbsPaths = append(fileAbsPaths, childKeys...)
		} else {
			// if is file, append to keys
			fileAbsPaths = append(fileAbsPaths, absPath)
		}
	}

	return
}

type Config struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Bucket    string `json:"bucket"`
	Domain    string `json:"domain"`
}

type Uploader struct {
	accessKey      string
	secretKey      string
	bucket         string
	domain         string
	upToken        string
	formUploader   *storage.FormUploader
	bucketManager  *storage.BucketManager
	allowImgSuffix []string
	markdownSuffix []string
}

func NewUploader(accessKey, secretKey, bucket string, allowImgSuffix []string) *Uploader {
	if allowImgSuffix == nil {
		allowImgSuffix = imgSuffix
	}
	upload := &Uploader{
		accessKey:      accessKey,
		secretKey:      secretKey,
		bucket:         bucket,
		allowImgSuffix: allowImgSuffix,
		domain:         config.Domain,
	}
	upload.init()
	return upload
}

// Init uploader, generate up token and form uploader
func (u *Uploader) init() {
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
	u.bucketManager = storage.NewBucketManager(mac, &cfg)

	return
}

// Upload local file, return file name
func (u *Uploader) uploadLocalFileWithKey(file string) (key string, err error) {
	// check
	isDir, err := u.checkIsDir(file)
	if isDir {
		return
	}
	if !u.validateSuffix(file) {
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

// Upload local file, return hash
func (u *Uploader) uploadLocalFileWithoutKey(file string) (key string, err error) {
	// check
	isDir, err := u.checkIsDir(file)
	if isDir {
		return
	}
	if !u.validateSuffix(file) {
		return
	}
	// upload
	ret := storage.PutRet{}
	err = u.formUploader.PutFileWithoutKey(context.Background(), &ret, u.upToken, file, nil)
	if err != nil {
		return
	}
	key = ret.Key

	return
}

// Upload net file, return hash
func (u *Uploader) uploadNetWithoutKey(url string) (key string, err error) {
	ret, err := u.bucketManager.FetchWithoutKey(url, u.bucket)
	if err != nil {
		return
	}
	key = ret.Key

	return
}

// Validate file suffix
func (u *Uploader) validateSuffix(name string) bool {
	for _, suffix := range u.allowImgSuffix {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
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
	if filePath == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}
	markdownUp := NewMarkdownUp(filePath, config.AccessKey, config.SecretKey, config.Bucket, nil)
	markdownUp.upload()
}