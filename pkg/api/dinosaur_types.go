package api

import "github.com/jinzhu/gorm"

type Dinosaur struct {
	Meta
	Species string
}

type DinosaurList []*Dinosaur
type DinosaurIndex map[string]*Dinosaur

func (l DinosaurList) Index() DinosaurIndex {
	index := DinosaurIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (org *Dinosaur) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", NewID())
}

type DinosaurPatchRequest struct {
	Species *string `json:"species,omitempty"`
}
