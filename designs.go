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

// Design controller
func getDesignRender(ctx context.Context) error {
	log.Println("Getting design render")
	// Load the design
	designId := ctx.PathParams().Get("id")
	des, err := findDesignById(designId.Str())
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	// Load the frame material. Default to black.
	materialId := "542c5f3bc296ec236005bffa" // black
	//materialId = "542d7ad1119e3247afd88f82"  // havana
	if matId := ctx.FormValue("materialid"); len(matId) > 0 {
		materialId = matId
	}

	left := des.Front.Outercurve.scale(10)
	right := des.Front.Outercurve.scale(10)
	// The Y coordinate of the bottom of the bridge of the glasses, relative to the HRL
	origin := left[len(left)-1][1]

	filename := fmt.Sprintf("%v-%v.png", designId.Str(), materialId)
	url := fmt.Sprintf("http://%v/static/%v", ctx.HttpRequest().Host, filename)
	type renderResponse struct {
		Url           string  `json:"url"`
		Y_offset      float64 `json:"y_offset"`
		PixelsDensity int16   `json:"pixels_per_mm"`
	}
	//	fstat, err := os.Stat(fmt.Sprintf("./static-files/%v", filename))
	//	if err == nil {
	//		render_time := fstat.ModTime()
	//		if render_time.After(des.Updated) {
	//			log.Println("Returning cached render info")
	//			dinfo := renderResponse{url, 900 - origin, 10}
	//			return goweb.API.RespondWithData(ctx, dinfo)
	//		}
	//	}

	material, err := findMaterialById(materialId)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	// PNG image.  Dimensions by convention, correspond to 1mm : 10px
	im := image.NewRGBA(image.Rect(0, 0, 2000, 900))

	// Offset the frame so it just fits on the canvas
	_, miny := left.minValues()
	for i, pt := range left {
		left[i] = Point{pt[0] + 1000, pt[1] - miny}     // Center on graphic
		right[i] = Point{pt[0]*-1 + 1000, pt[1] - miny} // Center on graphic
	}

	dc := material.TopColor
	fillColor := color.RGBA{uint8(dc[0]), uint8(dc[1]), uint8(dc[2]), uint8(dc[3])}

	// Get the curves for the outer contour
	bzs := left.convertToBeziers(false, true)
	bzs_r := right.convertToBeziers(false, true)
	gc := draw2d.NewGraphicContext(im)
	gc.SetFillColor(fillColor)
	gc.SetStrokeColor(fillColor)

	gc.MoveTo(bzs[0][0][0], bzs[0][0][1])
	for _, bez := range bzs {
		gc.CubicCurveTo(bez[1][0], bez[1][1], bez[2][0], bez[2][1], bez[3][0], bez[3][1])
	}
	for i := len(bzs_r) - 1; i >= 0; i-- {
		bez := bzs_r[i]
		gc.CubicCurveTo(bez[2][0], bez[2][1], bez[1][0], bez[1][1], bez[0][0], bez[0][1])
	}
	gc.FillStroke()

	if len(material.TopTexture) > 0 {
		// load the image of top texture and apply it
		if imFile, err := os.Open(material.TopTexture); err == nil {
			defer imFile.Close()
			if textIm, _, err2 := image.Decode(imFile); err2 == nil {
				bounds := im.Bounds()
				for i := bounds.Min.X; i <= bounds.Max.X; i++ {
					for j := bounds.Min.Y; j <= bounds.Max.Y; j++ {
						if im.At(i, j) == fillColor {
							r, g, b, a := textIm.At(i, j).RGBA()
							im.Set(i, j, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a) - 20})
						}
					}
				}
			} else {
				log.Printf("Error! %v", err2.Error())
			}
		} else {
			log.Printf("Error! %v", err.Error())
		}

	}

	lensColor := color.RGBA{255, 255, 255, 255}
	gc.SetFillColor(lensColor)
	gc.SetStrokeColor(lensColor)
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
	for i := bounds.Min.X; i <= bounds.Max.X; i++ {
		for j := bounds.Min.Y; j <= bounds.Max.Y; j++ {
			if im.At(i, j) == lensColor {
				im.Set(i, j, image.Transparent)
			}
		}
	}

	saveToPngFile(filename, im)

	dinfo := renderResponse{url, 900 - origin, 10}
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
