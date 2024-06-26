package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	genimage "go-pratice-introduction/chapter-9-b/funcs"
	"html/template"
	"image"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"
	"time"
)

var TILESDB map[string][3]float64

func main() {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("/public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))
	mux.HandleFunc("/", upload)
	mux.HandleFunc("/mosaic", mosaic)
	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	TILESDB = genimage.TilesDB()
	fmt.Println("Mosaic server started.")
	server.ListenAndServe()
}

func upload(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("upload.html")
	t.Execute(w, nil)
}

func mosaic(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()

	r.ParseMultipartForm(10485760)
	file, _, _ := r.FormFile("image")
	defer file.Close()
	tileSize, _ := strconv.Atoi(r.FormValue("tile_size"))

	original, _, _ := image.Decode(file)
	bounds := original.Bounds()

	newimage := image.NewNRGBA(image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y))

	db := genimage.CloneTilesDB(TILESDB)

	sp := image.Point{0, 0}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += tileSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += tileSize {

			r, g, b, _ := original.At(x, y).RGBA()
			color := [3]float64{float64(r), float64(g), float64(b)}

			nearest := genimage.Nearest(color, &db)
			file, err := os.Open(nearest)
			if err == nil {
				img, _, err := image.Decode(file)
				if err == nil {
					t := genimage.Resize(img, tileSize)
					tile := t.SubImage(t.Bounds())
					tileBounds := image.Rect(sp.X, sp.Y, sp.X+tileSize, sp.Y+tileSize)

					draw.Draw(newimage, tileBounds, tile, image.Point{0, 0}, draw.Src)
				} else {
					fmt.Println("error:", err, nearest)
				}
			} else {
				fmt.Println("error:", nearest)
			}
			file.Close()
		}
		sp = image.Point{0, sp.Y + tileSize}
	}

	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())

	buf2 := new(bytes.Buffer)
	jpeg.Encode(buf2, newimage, nil)
	mosaic := base64.StdEncoding.EncodeToString(buf2.Bytes())
	t1 := time.Now()
	images := map[string]string{
		"original": originalStr,
		"mosaic":   mosaic,
		"duration": fmt.Sprintf("%v ", t1.Sub(t0)),
	}
	t, _ := template.ParseFiles("results.html")
	t.Execute(w, images)
}
