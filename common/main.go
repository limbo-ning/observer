package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"obsessiontech/common/img"
	"obsessiontech/common/img/chaostoken"
	"os"
	"strings"

	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
)

func cropImages() {
	fpi := 5
	height := 844
	width := 640

	// source, err := os.Open("/Users/limbo/Desktop/序列帧02.png")
	source, err := os.Open("animation.png")
	if err != nil {
		log.Panic(err)
	}
	defer source.Close()

	sourceImage, err := png.Decode(source)
	if err != nil {
		log.Panic(err)
	}

	frames := sourceImage.Bounds().Max.X / width

	for i := 0; i <= frames/fpi; i++ {
		target, err := os.Create(fmt.Sprintf("animation%d_%d.png", fpi, i))
		if err != nil {
			log.Panic(err)
		}

		var result *image.NRGBA
		if i == frames/fpi {
			result = image.NewNRGBA(image.Rect(0, 0, (frames%fpi)*width, height))
		} else {
			result = image.NewNRGBA(image.Rect(0, 0, fpi*width, height))
		}

		draw.Draw(result, result.Bounds(), sourceImage, image.Pt(i*fpi*width, 0), draw.Src)
		png.Encode(target, result)
	}
}

func mergeImages() {
	height := 468
	width := 396

	number := 13

	target, err := os.Create("kangaroo.png")
	if err != nil {
		log.Panic(err)
	}

	result := image.NewNRGBA(image.Rect(0, 0, number*width, height))

	for i := 0; i < number; i++ {
		source, err := os.Open(strings.Replace(fmt.Sprintf("外卖小哥跑/外卖小哥%d.png", 100+i), "1", "", 1))
		if err != nil {
			log.Panic(err)
		}
		defer source.Close()

		sourceImage, err := png.Decode(source)
		if err != nil {
			log.Panic(err)
		}

		draw.Draw(result, image.Rect(i*width, 0, (i+1)*width, height), sourceImage, image.Pt(0, 0), draw.Src)
	}

	png.Encode(target, result)
}

func convertPNG2JPG() {

	target, err := os.Create("gameEndBg.jpg")
	if err != nil {
		log.Panic(err)
	}
	defer target.Close()

	source, err := os.Open("gameEndBg.png")
	if err != nil {
		log.Panic(err)
	}
	defer source.Close()

	sourceImage, err := png.Decode(source)
	if err != nil {
		log.Panic(err)
	}

	result := image.NewNRGBA(sourceImage.Bounds())
	draw.Draw(result, sourceImage.Bounds(), sourceImage, image.Pt(0, 0), draw.Src)

	jpeg.Encode(target, result, nil)

}

func convertTxtToSQL() {
	sourceName := "workbench/TextToSQL/2.txt"
	targetName := "workbench/TextToSQL/2.sql"
	source, err := os.Open(sourceName)
	if err != nil {
		log.Panic(err)
	}
	defer source.Close()

	target, err := os.Create(targetName)
	writer := bufio.NewWriter(target)

	writer.WriteString("INSERT INTO nba_consumable (`type`, `value`) VALUES \n")

	isFirst := true

	br := bufio.NewReader(source)
	for {
		a, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}

		if isFirst {
			isFirst = false
		} else {
			writer.WriteString(",\n")
		}

		writer.WriteString(fmt.Sprintf(`("RED_COVER", "%s")`, a))
	}

	writer.WriteString(";")

	writer.Flush()
}

func greyscaleImage() {

	// source, err := os.Open("workbench/img/CheckCode.gif")
	source, err := os.Open("workbench/img/code.png")
	if err != nil {
		log.Panic(err)
	}
	defer source.Close()

	image, err := img.LoadImage(source, "png")
	if err != nil {
		log.Panic(err)
	}

	result, lightest, darkest := img.GrayscaleImage(image)

	log.Println("greyscale: ", lightest, darkest)

	result = img.ThresholdGrayscale(result, lightest+(darkest-lightest)/2)

	target, err := os.Create("workbench/img/code_result.jpg")
	if err != nil {
		log.Panic(err)
	}
	jpeg.Encode(target, result, nil)
}

func chaos() {

	result, err := chaostoken.ChaosToken("1234", "workbench/Arial.ttf", 300, 100)
	if err != nil {
		log.Panic(err)
	}
	target, err := os.Create("workbench/img/chaos_result.jpg")
	if err != nil {
		log.Panic(err)
	}
	_, err = io.Copy(target, bytes.NewReader(result))
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	// mergeImages()
	// cropImages()
	// convertPNG2JPG()
	// convertTxtToSQL()
	// greyscaleImage()
	chaos()
}
