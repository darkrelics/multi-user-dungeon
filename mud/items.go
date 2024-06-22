package main

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type Item struct {
	Index       uint64
	Name        string
	Description string
	Mass        float64
	Wearable    bool
	Verbs       map[string]string
	Overrides   map[string]string
	Container   bool
	Contents    []uint64
	IsPrototype bool
}

type ItemData struct {
	Index       uint64            `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mass        float64           `json:"mass"`
	Wearable    bool              `json:"wearable"`
	Verbs       map[string]string `json:"verbs"`
	Overrides   map[string]string `json:"overrides"`
	Container   bool              `json:"container"`
	Contents    []uint64          `json:"contents"`
	IsPrototype bool              `json:"is_prototype"`
}

func (k *KeyPair) LoadItem(indexKey uint64, isPrototype bool) (*Item, error) {
	var objectData []byte
	bucketName := "Items"
	if isPrototype {
		bucketName = "ItemPrototypes"
	}

	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("%s bucket not found", bucketName)
		}
		indexKey := fmt.Sprintf("%d", indexKey)
		objectData = bucket.Get([]byte(indexKey))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if objectData == nil {
		return nil, fmt.Errorf("object not found")
	}

	var od ItemData
	if err := json.Unmarshal(objectData, &od); err != nil {
		return nil, fmt.Errorf("error unmarshalling object data: %v", err)
	}

	object := &Item{
		Index:       od.Index,
		Name:        od.Name,
		Description: od.Description,
		Mass:        od.Mass,
		Verbs:       od.Verbs,
		Overrides:   od.Overrides,
		Container:   od.Container,
		Contents:    od.Contents,
		IsPrototype: od.IsPrototype,
	}

	return object, nil
}

func (k *KeyPair) WriteItem(obj *Item) error {
	// First, serialize the Items to JSON
	objData := ItemData{
		Index:       obj.Index,
		Name:        obj.Name,
		Description: obj.Description,
		Mass:        obj.Mass,
		Verbs:       obj.Verbs,
		Overrides:   obj.Overrides,
		Container:   obj.Container,
		Contents:    obj.Contents,
		IsPrototype: obj.IsPrototype,
	}
	serializedData, err := json.Marshal(objData)
	if err != nil {
		return fmt.Errorf("error marshalling object data: %v", err)
	}

	bucketName := "Items"
	if obj.IsPrototype {
		bucketName = "ItemPrototypes"
	}

	// Write serialized data to the database
	err = k.db.Update(func(tx *bolt.Tx) error {
		// Ensure the appropriate bucket exists
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		// Use the object's Index as the key for the database entry
		indexKey := fmt.Sprintf("%d", obj.Index)

		// Store the serialized object data in the bucket
		err = bucket.Put([]byte(indexKey), serializedData)
		if err != nil {
			return fmt.Errorf("failed to write object data: %v", err)
		}

		return nil
	})

	return err
}
