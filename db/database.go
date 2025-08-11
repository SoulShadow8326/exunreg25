package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Database struct{
	data map[string]interface{}
	file string
	mu sync.RWMutex
}

func NewConnection(dbPath string)(*Database, error){
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil{
		return nil, fmt.Errorf("error creating database directory: %v", err)
	}
	db := &Database{
		data: make(map[string]interface{}),
		file: dbPath,
	}
	if err := db.load(); err!=nil{
		log.Printf("Warning: Could not load existing database: %v", err)
	}
	log.Printf("db connection success: %s", dbPath)
	return db, nil
}

func (db *Database) Close() error{
	return db.save()
}

func (db *Database) Create(key string, value interface{}) error{
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.data[key]; exists{
		return fmt.Errorf("key already exists: %s", key)
	}
	db.data[key] = value
	return db.save()
}

func (db *Database) Get(key string)(interface{}, error){
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[key]
	if !exists{
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

func (db *Database) Update(key string, value interface{}) error{
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.data[key]; !exists{
		return fmt.Errorf("key not found: %s", key)
	}
	db.data[key] = value
	return db.save()
}

func (db *Database) Delete(key string) error{
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.data[key]; !exists{
		return fmt.Errorf("key not found: %s", key)
	}
	delete(db.data, key)
	return db.save()
}

func (db *Database) load() error{
	data, err := os.ReadFile(db.file)
	if err!=nil{
		if os.IsNotExist(err){
			return nil
		}
		return fmt.Errorf("error reading database file: %v", err)
	}
	if len(data) == 0{
		return nil
	}
	if err := json.Unmarshal(data, &db.data); err != nil{
		return fmt.Errorf("error unmarshaling database data: %v", err)
	}
	return nil
}

func (db *Database) save()error{
	data, err := json.MarshalIndent(db.data, "", "  ")
	if err != nil{
		return fmt.Errorf("error marshaling database data: %v", err)
	}
	if err := os.WriteFile(db.file, data, 0644); err != nil{
		return fmt.Errorf("error writing database file: %v", err)
	}
	return nil
}