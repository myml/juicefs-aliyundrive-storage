package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/chyroc/go-aliyundrive"
)

var (
	configDefaultBucket = []byte("aliyundrive_store")
)

type ConfigStore struct {
	db *bolt.DB
}

func (store *ConfigStore) Get(ctx context.Context, key string) (*aliyundrive.Token, error) {
	var token aliyundrive.Token
	return &token, store.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(configDefaultBucket)
		data := b.Get([]byte("token"))
		if len(data) == 0 {
			return errors.New("not found")
		}
		return json.Unmarshal(data, &token)
	})
}

func (store *ConfigStore) Set(ctx context.Context, token *aliyundrive.Token) error {
	return store.db.Update(func(t *bolt.Tx) error {
		b := t.Bucket(configDefaultBucket)
		data, err := json.Marshal(token)
		if err != nil {
			return err
		}
		return b.Put([]byte("token"), data)
	})
}

func NewConfigStore(db *bolt.DB) (*ConfigStore, error) {
	err := db.Update(func(t *bolt.Tx) error {
		_, err := t.CreateBucketIfNotExists(configDefaultBucket)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &ConfigStore{db: db}, nil
}
