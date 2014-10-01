package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"time"

	"code.google.com/p/draw2d/draw2d"

	"github.com/stretchr/goweb"
	"github.com/stretchr/goweb/context"
)

type designController struct{}

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
	front.Outercurve = make(BSpline, len(o2))
	for i, pt1 := range o2 {
		pt := pt1.(map[string]interface{})
		front.Outercurve[i] = [2]float64{pt["x"].(float64), pt["y"].(float64)}
	}

	lens := import_design["eyehole"].(map[string]interface{})
	lenspts := lens["points"].([]interface{})
	front.Lens = make(BSpline, len(lenspts))
	for i, ptI := range lenspts {
		pt := ptI.(map[string]interface{})
		front.Lens[i] = [2]float64{pt["x"].(float64), pt["y"].(float64)}
	}
	design.Front = front

	temple := Temple{}
	templec := import_design["templecurve"].(map[string]interface{})
	templepts := templec["points"].([]interface{})
	temple.Contour = make(BSpline, len(templepts))
	for i, ptI := range templepts {
		pt := ptI.(map[string]interface{})
		temple.Contour[i] = [2]float64{pt["x"].(float64), pt["y"].(float64)}
	}

	location := import_design["templelocation"].(map[string]interface{})
	temple.TempleSeparation = int16(location["x"].(float64) * 100)
	temple.TempleHeight = int16(location["y"].(float64) * 100)
	design.Temple = temple

	if err = insertDesign(&design); err != nil {
		return goweb.API.RespondWithError(ctx, 500, err.Error())
	}

	return goweb.API.WriteResponseObject(ctx, 201, design)
}

type RenderResponse struct {
	Url      string  `json:"url"`
	Y_offset float64 `json:"y_offset"`
}

// Design controller
func getDesignRender(ctx context.Context) error {
	designId := ctx.PathParams().Get("id")
	des, err := findDesignById(designId.Str())
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	left := des.Front.Outercurve.scale(10)
	right := des.Front.Outercurve.scale(10)

	_, miny := left.minValues()
	for i, pt := range left {
		left[i] = Point{pt[0] + 1000, pt[1] - miny}     // Center on graphic
		right[i] = Point{pt[0]*-1 + 1000, pt[1] - miny} // Center on graphic
	}
	im := image.NewRGBA(image.Rect(0, 0, 2000, 900))

	origin := left[len(left)-1][1]

	bzs := left.convertToBeziers(false, true)
	bzs_r := right.convertToBeziers(false, true)
	gc := draw2d.NewGraphicContext(im)
	gc.SetFillColor(image.Black)

	gc.MoveTo(bzs[0][0][0], bzs[0][0][1])
	for _, bez := range bzs {
		gc.CubicCurveTo(bez[1][0], bez[1][1], bez[2][0], bez[2][1], bez[3][0], bez[3][1])
	}
	for i := len(bzs_r) - 1; i >= 0; i-- {
		bez := bzs_r[i]
		gc.CubicCurveTo(bez[2][0], bez[2][1], bez[1][0], bez[1][1], bez[0][0], bez[0][1])
	}
	gc.FillStroke()

	lensColor := color.RGBA{255, 255, 255, 255}
	gc.SetFillColor(lensColor)
	lens_l := des.Front.Lens.scale(10)
	lens_r := des.Front.Lens.scale(10)
	for i, pt := range lens_l {
		lens_l[i] = Point{pt[0] + 1000, pt[1] - miny}
		lens_r[i] = Point{-1*pt[0] + 1000, pt[1] - miny}
	}
	lens_bzr := lens_l.convertToBeziers(true, false)
	lens_bzr_r := lens_r.convertToBeziers(true, false)
	gc.MoveTo(lens_bzr[0][0][0], lens_bzr[0][0][1])
	for _, bez := range lens_bzr {
		gc.CubicCurveTo(bez[1][0], bez[1][1], bez[2][0], bez[2][1], bez[3][0], bez[3][1])
	}
	gc.FillStroke()
	gc.MoveTo(lens_bzr_r[0][0][0], lens_bzr_r[0][0][1])
	for _, bez := range lens_bzr_r {
		gc.CubicCurveTo(bez[1][0], bez[1][1], bez[2][0], bez[2][1], bez[3][0], bez[3][1])
	}
	gc.FillStroke()

	// Convert all white to transparent
	bounds := im.Bounds()
	count := 0
	changed := 0
	for i := bounds.Min.X; i <= bounds.Max.X; i++ {
		for j := bounds.Min.Y; j <= bounds.Max.Y; j++ {
			count++
			if im.At(i, j) == lensColor {
				im.Set(i, j, image.Transparent)
				changed++
			}
		}
	}
	log.Printf("Examined %v pixels and changed %v", count, changed)

	filename := fmt.Sprintf("%v.png", designId.Str())
	url := fmt.Sprintf("http://%v/static/%v", ctx.HttpRequest().Host, filename)
	saveToPngFile(filename, im)

	dinfo := RenderResponse{url, 900 - origin}
	return goweb.API.RespondWithData(ctx, dinfo)
}

func saveToPngFile(filePath string, m image.Image) {
	f, err := os.Create("static-files/" + filePath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	b := bufio.NewWriter(f)
	err = png.Encode(b, m)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s OK.\n", filePath)
}
