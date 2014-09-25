package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/stretchr/goweb"
	"github.com/stretchr/goweb/context"
)

type designController struct{}

// Import old designs
func importDesign(ctx context.Context) error {
	log.Println("Importing design")
	userdata := ctx.Data()["user"]
	if userdata == nil {
		return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
	}
	user := userdata.(User)

	// Read the data
	data, err := ctx.RequestBody()
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	var old_design interface{}
	err = json.Unmarshal(data, &old_design)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	import_design := old_design.(map[string]interface{})

	design := Design{}
	design.Name = import_design["name"].(string)
	design.Collections = []string{"Templates"}
	design.Updated = time.Now()
	design.Designer = user.Id

	front := Front{}
	o1 := import_design["outercurve"].(map[string]interface{})
	o2 := o1["points"].([]interface{})
	front.Outercurve = make([][2]int16, len(o2))
	for i, pt1 := range o2 {
		pt := pt1.(map[string]interface{})
		front.Outercurve[i] = [2]int16{int16(pt["x"].(float64) * 100), int16(pt["y"].(float64) * 100)}
	}

	lens := import_design["eyehole"].(map[string]interface{})
	lenspts := lens["points"].([]interface{})
	front.Lens = make([][2]int16, len(lenspts))
	for i, ptI := range lenspts {
		pt := ptI.(map[string]interface{})
		front.Lens[i] = [2]int16{int16(pt["x"].(float64) * 100), int16(pt["y"].(float64) * 100)}
	}
	design.Front = front

	temple := Temple{}
	templec := import_design["templecurve"].(map[string]interface{})
	templepts := templec["points"].([]interface{})
	temple.Contour = make([][2]int16, len(templepts))
	for i, ptI := range templepts {
		pt := ptI.(map[string]interface{})
		temple.Contour[i] = [2]int16{int16(pt["x"].(float64) * 100), int16(pt["y"].(float64) * 100)}
	}

	location := import_design["templelocation"].(map[string]interface{})
	temple.TempleSeparation = int16(location["x"].(float64) * 100)
	temple.TempleHeight = int16(location["y"].(float64) * 100)
	design.Temple = temple

	log.Println("Constructed design, saving")

	if err = insertDesign(&design); err != nil {
		return goweb.API.RespondWithError(ctx, 500, err.Error())
	}

	return goweb.API.WriteResponseObject(ctx, 201, design)
}

// Design controller
