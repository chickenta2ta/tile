package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	"github.com/bamiaux/rez"
)

func isBackground(m image.Image, threshold float64) bool {
	whiteThreshold := 240
	blackThreshold := 15

	whitePixels := 0
	blackPixels := 0
	totalPixels := 0

	bounds := m.Bounds()
	for x := bounds.Min.X; x <= bounds.Max.X; x++ {
		for y := bounds.Min.Y; y <= bounds.Max.Y; y++ {
			r, g, b, _ := m.At(x, y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			if int(r8) >= whiteThreshold && int(g8) >= whiteThreshold && int(b8) >= whiteThreshold {
				whitePixels++
			}
			if int(r8) <= blackThreshold && int(g8) <= blackThreshold && int(b8) <= blackThreshold {
				blackPixels++
			}
			totalPixels++
		}
	}

	return float64(whitePixels+blackPixels)/float64(totalPixels) >= threshold
}

func crop(m, n image.Image, x, y, size int, threshold, scale float64, imageId string) {

	thumbScale := float64(n.Bounds().Dx()) / float64(m.Bounds().Dx())

	thumbX := int(float64(x) * thumbScale)
	thumbY := int(float64(y) * thumbScale)
	thumbSize := int(float64(size) * thumbScale)

	rect := image.Rect(thumbX, thumbY, thumbX+thumbSize, thumbY+thumbSize)
	subimage := n.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(rect)

	if isBackground(subimage, threshold) {
		subimage = nil
		return
	}

	rect = image.Rect(x, y, x+size, y+size)
	subimage = m.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(rect)

	rect = image.Rect(0, 0, int(float64(size)*scale), int(float64(size)*scale))
	newrgba := image.NewRGBA(rect)

	err := rez.Convert(newrgba, subimage, rez.NewBilinearFilter())
	if err != nil {
		fmt.Println(err)
		return
	}
	subimage = nil

	name := filepath.Join("/kaggle/working/patches", fmt.Sprintf("%s_%d_%d.png", imageId, x, y))
	f, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	err = png.Encode(f, newrgba)
	if err != nil {
		fmt.Println(err)
		return
	}
	newrgba = nil
}

func main() {
	imagePath := flag.String("p", "image_path", "")
	thumbnailPath := flag.String("t", "thumbnail_path", "")
	imageId := flag.String("i", "image_id", "")
	size := flag.Int("s", 2048, "")
	scale := flag.Float64("x", 0.25, "")
	threshold := flag.Float64("h", 0.125, "")

	flag.Parse()

	f, err := os.Open(filepath.Join(*imagePath, fmt.Sprintf("%s.png", *imageId)))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	g, err := os.Open(filepath.Join(*thumbnailPath, fmt.Sprintf("%s_thumbnail.png", *imageId)))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer g.Close()

	m, _, err := image.Decode(f)
	if err != nil {
		fmt.Println(err)
		return
	}

	n, _, err := image.Decode(g)
	if err != nil {
		fmt.Println(err)
		return
	}

	bounds := m.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var wg sync.WaitGroup
	for x := 0; x <= width-*size; x += *size {
		for y := 0; y <= height-*size; y += *size {
			wg.Add(1)
			go func(x, y int) {
				defer wg.Done()
				crop(m, n, x, y, *size, *threshold, *scale, *imageId)
			}(x, y)
		}
	}
	wg.Wait()
}
