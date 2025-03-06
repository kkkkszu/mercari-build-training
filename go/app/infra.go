package app

import (
	"context"
        "encoding/json"
	"errors"
        "os"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID   int    `db:"id" json:"-"`
	Name string `db:"name" json:"name"`
        Category string `db:"Category" json:"Category"`
        Image string `db:"image" json:"image"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	// STEP 4-2: add an implementation to store an item
        data, err := os.ReadFile(i.fileName)
    if err != nil && !os.IsNotExist(err) {
        return err 
    }

    var items struct {
        Items []Item `json:"items"` 
    }

    // 既存のデータがある場合
    if len(data) > 0 {
        err = json.Unmarshal(data, &items)
        if err != nil {
            return err 
        }
    }

    items.Items = append(items.Items, *item)


    updatedData, err := json.MarshalIndent(items, "", "  ")
    if err != nil {
        return err
    }


    err = os.WriteFile(i.fileName, updatedData, 0644)
    if err != nil {
        return err
    }

	return nil
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image

	return nil
}
