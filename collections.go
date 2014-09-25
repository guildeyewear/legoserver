package main

import (
	"log"

	"github.com/stretchr/goweb"
	"github.com/stretchr/goweb/context"
)

type collectionsController struct {
	collections []string
}

func newCollectionsController() *collectionsController {
	c := collectionsController{}
	c.collections = []string{"Temples"}
	return &c
}

func (c *collectionsController) Read(collection string, ctx context.Context) error {
	log.Println("Getting designs in collections", collection)
	designs, err := getDesignsWithCollection(collection)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.RespondWithData(ctx, designs)
}
