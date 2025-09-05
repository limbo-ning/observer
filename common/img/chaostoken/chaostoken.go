package chaostoken

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"obsessiontech/common/config"
	"obsessiontech/common/img"
	"obsessiontech/common/random"

	"github.com/golang/freetype"
)

var Config struct {
	DefaultFontFile string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}

func ChaosToken(token, fontFile string, width, height int) ([]byte, error) {

	if width == 0 || height == 0 {
		return nil, errors.New("invalid size")
	}

	buff := make([]byte, 0)
	buffer := bytes.NewBuffer(buff)

	result := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(result, result.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	wordCount := len(token)
	boxSize := width / (wordCount)
	fontSize := float64(boxSize) * 0.75

	if fontFile == "" {
		fontFile = Config.DefaultFontFile
	}

	if fontFile == "" {
		return nil, errors.New("font file not set")
	}

	// 读字体数据
	fontBytes, err := ioutil.ReadFile(fontFile)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetFontSize(fontSize)

	for i, _ := range token {

		canvas := image.NewRGBA(image.Rect(0, 0, boxSize, boxSize))

		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

		c.SetClip(canvas.Bounds())
		c.SetDst(canvas)
		c.SetSrc(image.NewUniform(color.RGBA{R: 50 + uint8(random.GetRandomNumber(110)), G: 50 + uint8(random.GetRandomNumber(110)), B: 50 + uint8(random.GetRandomNumber(110)), A: 1}))

		pt := freetype.Pt(int(0.125*float64(boxSize)), int(float64(boxSize)-0.125*float64(boxSize))) // 字出现的位置

		advance, err := c.DrawString(token[i:i+1], pt)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		direction := random.GetRandomNumber(2)
		skewX := random.GetRandomNumber(advance.X.Round()) * (direction - 1)

		var offsetX, offsetY int
		offsetX = i*boxSize + (boxSize-advance.X.Round())/2
		offsetY = height/2 - boxSize/2 - (boxSize-advance.X.Round())/2
		for i := 0; i < canvas.Bounds().Dx(); i++ {
			for j := 0; j < canvas.Bounds().Dy(); j++ {
				skewXOffset := skewX * (canvas.Bounds().Dy()/2 - j) / (canvas.Bounds().Dy() / 2)
				clr := canvas.At(i, j)
				if _, _, _, a := clr.RGBA(); a > 0 {
					result.Set(offsetX+i+skewXOffset, offsetY+j, clr)
				}
			}
		}

	}

	// lineCount := wordCount + 2
	pointCount := wordCount * 5

	// for i := 0; i < lineCount; i++ {
	// 	img.DrawLine(result, color.RGBA{R: 40 + uint8(random.GetRandomNumber(140)), G: 40 + uint8(random.GetRandomNumber(140)), B: 40 + uint8(random.GetRandomNumber(140)), A: 1}, 1+random.GetRandomNumber(5), i*width/lineCount+random.GetRandomNumber((i+1)*width/lineCount), random.GetRandomNumber(height/2), random.GetRandomNumber(width), height/2+random.GetRandomNumber(height/2))

	// }

	for i := 0; i < pointCount; i++ {
		clr := color.RGBA{R: 40 + uint8(random.GetRandomNumber(140)), G: 40 + uint8(random.GetRandomNumber(140)), B: 40 + uint8(random.GetRandomNumber(140)), A: 1}
		x := random.GetRandomNumber(width)
		y := random.GetRandomNumber(height)

		img.DrawPoint(result, clr, x, y)
		img.DrawPoint(result, clr, x-1, y)
		img.DrawPoint(result, clr, x+1, y)
		img.DrawPoint(result, clr, x, y-1)
		img.DrawPoint(result, clr, x, y+1)
		img.DrawPoint(result, clr, x-1, y-1)
		img.DrawPoint(result, clr, x+1, y+1)
		img.DrawPoint(result, clr, x+1, y-1)
		img.DrawPoint(result, clr, x-1, y+1)
	}

	jpeg.Encode(buffer, result, nil)

	return buffer.Bytes(), nil
}
