package main

import (
	"log"

	"github.com/guildeyewear/legoserver/models"
	"github.com/stretchr/goweb"
	"github.com/stretchr/goweb/context"
)

type collectionsController struct{}

func (c *collectionsController) ReadMany(ctx context.Context) error {
	templates := []string{"Templates"}
	log.Println("Returning array")
	return goweb.API.RespondWithData(ctx, templates)
}

func (c *collectionsController) Read(collection string, ctx context.Context) error {
	log.Println("Getting designs in collections", collection)
	designs, err := models.GetDesignsWithCollection(collection)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.RespondWithData(ctx, designs)
}
