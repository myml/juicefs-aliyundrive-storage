package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/chyroc/go-aliyundrive"
)

func main() {
}
func New(bucket, accessKey, secretKey string) (interface{}, error) {
	a, err := NewAliyun(bucket, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	return a, nil
}

var (
	fileDefaultBoltBucket = []byte("file")
)

type Aliyun struct {
	sdk      *aliyundrive.AliyunDrive
	driveID  string
	parentID string
	db       *bolt.DB
}

func (aliyun *Aliyun) Get(key string, off, limit int64) (io.ReadCloser, error) {
	log.Println("get", key)

	var fileID string
	aliyun.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(fileDefaultBoltBucket)
		fileID = string(b.Get([]byte(key)))
		return nil
	})
	if len(fileID) == 0 {
		return nil, fmt.Errorf("not found")
	}
	t, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(t)
	dist := filepath.Join(t, "tmp")
	err = aliyun.sdk.File.DownloadFile(context.Background(), &aliyundrive.DownloadFileReq{DriveID: aliyun.driveID, Dist: dist, FileID: fileID})
	if err != nil {
		return nil, err
	}
	f, err := os.Open(dist)
	if err != nil {
		return nil, err
	}
	_, err = f.Seek(off, 0)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (aliyun *Aliyun) Put(key string, r io.Reader) error {
	log.Println("put", key)
	tmpFile, err := os.CreateTemp(os.TempDir(), "")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()
	_, err = io.Copy(tmpFile, r)
	if err != nil {
		return err
	}
	err = tmpFile.Close()
	if err != nil {
		return err
	}
	resp, err := aliyun.sdk.File.UploadFile(context.Background(), &aliyundrive.UploadFileReq{
		DriveID:  aliyun.driveID,
		ParentID: aliyun.parentID,
		FilePath: tmpFile.Name(),
	})
	if err != nil {
		return err
	}
	log.Println(aliyun)
	return aliyun.db.Update(func(t *bolt.Tx) error {
		b := t.Bucket(fileDefaultBoltBucket)
		log.Println("upload file", aliyun.driveID, tmpFile.Name(), key, resp)
		return b.Put([]byte(key), []byte(resp.FileID))
	})
}

func (aliyun *Aliyun) Delete(key string) error {
	log.Println("delete", key)
	return aliyun.db.Update(func(t *bolt.Tx) error {
		b := t.Bucket(fileDefaultBoltBucket)
		fileID := string(b.Get([]byte(key)))
		_, err := aliyun.sdk.File.DeleteFile(context.Background(), &aliyundrive.DeleteFileReq{DriveID: aliyun.driveID, FileID: fileID})
		return err
	})
}

func (aliyun *Aliyun) String() string {
	return "aliyun drive"
}

// NewAliyun 初始化Aliyun网盘的对象存储
func NewAliyun(u, user, passwd string) (*Aliyun, error) {
	cfgURL, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	// 阿里云盘目录
	aliyunDir := cfgURL.Query().Get("dir")
	if len(aliyunDir) == 0 {
		aliyunDir = "juicefs"
	}
	// 阿里云盘SDK的工作目录，用于存储cookie、日志和fileName=>fileID映射
	worker := cfgURL.Query().Get("worker")
	if len(worker) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get user home dir: %w", err)
		}
		worker = filepath.Join(home, ".go-aliyundrive-sdk")
	}
	// 初始化bolt
	log.Println(worker)
	db, err := bolt.Open(filepath.Join(worker, "bolt.db"), 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bolt: %w", err)
	}
	err = db.Update(func(t *bolt.Tx) error {
		_, err := t.CreateBucketIfNotExists(fileDefaultBoltBucket)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create bucket: %w", err)
	}
	store, err := NewConfigStore(db)
	if err != nil {
		return nil, fmt.Errorf("create config store: %w", err)
	}
	opt := []aliyundrive.ClientOptionFunc{aliyundrive.WithStore(store)}
	if len(worker) > 0 {
		opt = append(opt, aliyundrive.WithWorkDir(worker))
	}
	sdk := aliyundrive.New(opt...)

	uinfo, err := sdk.Auth.LoginByQrcode(context.Background())
	if err != nil {
		return nil, fmt.Errorf("login aliyun: %w", err)
	}
	log.Println(uinfo.DefaultDriveID)
	resp, err := sdk.File.CreateFolder(context.Background(), &aliyundrive.CreateFolderReq{
		DriveID:      uinfo.DefaultDriveID,
		ParentFileID: "root",
		Name:         "juicefs",
	})
	if err != nil {
		return nil, fmt.Errorf("create foleder: %w", err)
	}
	log.Println("drive id", uinfo.DefaultDriveID, "parent id", resp.FileID)
	return &Aliyun{db: db, sdk: sdk, driveID: uinfo.DefaultDriveID, parentID: resp.FileID}, nil
}
