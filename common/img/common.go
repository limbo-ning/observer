package img

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"
)

func DrawPoint(img *image.RGBA, clr color.Color, x, y int) {
	img.Set(x, y, clr)
}
func DrawLine(img *image.RGBA, clr color.Color, width, startX, startY, endX, endY int) {

	if width == 0 {
		return
	}

	if startX > endX {
		tmp := endX
		endX = startX
		startX = tmp

		tmp = endY
		endY = startY
		startY = tmp
	}

	var points [][2]int

	points = getPointsInLine(startX, endX, startY, endY)

	if width > 1 {
		var widthStartX, widthStartY, widthEndX, widthEndY int
		if startX == endX {
			widthStartX = startX - width/2
			widthEndX = startX + width/2
			widthStartY = startY
			widthEndY = startY
		} else {
			angle := math.Atan(float64(endY-startY) / float64(endX-startX))
			widthStartX = startX - int(math.Round(float64(width/2)*math.Sin(angle)))
			widthStartY = startY + int(math.Round(float64(width/2)*math.Cos(angle)))
			widthEndX = startX + int(math.Round(float64(width/2)*math.Sin(angle)))
			widthEndY = startY - int(math.Round(float64(width/2)*math.Cos(angle)))
		}

		for _, p := range append(getPointsInLine(widthStartX, startX, widthStartY, startY), getPointsInLine(startX, widthEndX, startY, widthEndY)...) {
			offsetX := p[0] - startX
			offsetY := p[1] - startY
			points = append(points, getPointsInLine(p[0], endX+offsetX, p[1], endY+offsetY)...)
		}
	}

	painted := make(map[string]bool)
	for _, p := range points {
		img.Set(p[0], p[1], clr)
		painted[fmt.Sprintf("%d,%d", p[0], p[1])] = true
	}

	//fill gaps
	for _, p := range points {
		for i := -1; i <= 1; i++ {
			for j := -1; j <= 1; j++ {
				if i == 0 && j == 0 {
					continue
				}
				if painted[fmt.Sprintf("%d,%d", p[0]+i, p[1]+j)] {
					continue
				}
				gapX := p[0] + i
				gapY := p[1] + j

				switch {
				case painted[fmt.Sprintf("%d,%d", gapX-1, gapY)] && painted[fmt.Sprintf("%d,%d", gapX+1, gapY)]:
				case painted[fmt.Sprintf("%d,%d", gapX, gapY-1)] && painted[fmt.Sprintf("%d,%d", gapX, gapY+1)]:
				case painted[fmt.Sprintf("%d,%d", gapX-1, gapY-1)] && painted[fmt.Sprintf("%d,%d", gapX+1, gapY+1)]:
				case painted[fmt.Sprintf("%d,%d", gapX-1, gapY+1)] && painted[fmt.Sprintf("%d,%d", gapX+1, gapY-1)]:
				default:
					continue
				}

				img.Set(gapX, gapY, clr)
				painted[fmt.Sprintf("%d,%d", gapX, gapY)] = true
			}
		}
	}
}

func getDistanceSqrPointToPoint(aX, aY, bX, bY int) float64 {
	return float64((bX-aX)*(bX-aX) + (bY-aY)*(bY-aY))
}

func getDistancePointToPoint(aX, aY, bX, bY int) float64 {
	return math.Sqrt(float64((bX-aX)*(bX-aX) + (bY-aY)*(bY-aY)))
}

func getDistancePointToLine(pointX, pointY, lineX, lineY int, lineAngle float64) float64 {
	return math.Abs(math.Cos(lineAngle) * (float64(pointY-lineY) - math.Tan(lineAngle)*float64(pointX-lineX)))
}

func getPointsInLine(startX, endX, startY, endY int) [][2]int {
	if startX > endX {
		tmp := endX
		endX = startX
		startX = tmp

		tmp = endY
		endY = startY
		startY = tmp
	}

	result := make([][2]int, 0)
	if startX == endX {
		if endY > startY {
			for i := startY; i <= endY; i++ {
				result = append(result, [2]int{startX, i})
			}
		} else {
			for i := endY; i <= startY; i++ {
				result = append(result, [2]int{startX, i})
			}
		}
		return result
	}

	angle := math.Atan(float64(endY-startY) / float64(endX-startX))

	exists := make(map[string]byte)

	currentX := startX
	currentY := startY

	for {
		result = append(result, [2]int{currentX, currentY})
		exists[fmt.Sprintf("%d,%d", currentX, currentY)] = 1

		if currentX == endX && currentY == endY {
			break
		}

		var closestDis float64
		var nextX, nextY int

		closestDis = -1
		possibleMoves := [][2]int{
			{currentX + 1, currentY},
		}

		if endY > startY {
			possibleMoves = append(possibleMoves, [2]int{currentX, currentY + 1}, [2]int{currentX + 1, currentY + 1})
		} else if endY < startY {
			possibleMoves = append(possibleMoves, [2]int{currentX, currentY - 1}, [2]int{currentX + 1, currentY - 1})
		}

		for _, p := range possibleMoves {
			if _, ok := exists[fmt.Sprintf("%d,%d", p[0], p[1])]; ok {
				continue
			}
			dis := getDistancePointToLine(p[0], p[1], startX, startY, angle)
			if closestDis == -1 || dis < closestDis {
				closestDis = dis
				nextX = p[0]
				nextY = p[1]
			}
		}

		currentX = nextX
		currentY = nextY
	}

	return result
}

func LoadImage(source io.Reader, imgType string) (image.Image, error) {

	switch strings.ToLower(imgType) {
	case "png":
		return png.Decode(source)
	case "gif":
		return gif.Decode(source)
	case "jpg":
		fallthrough
	case "jpeg":
		return jpeg.Decode(source)
	}

	return nil, errors.New("unsupport image")
}

func GrayscaleImage(img image.Image) (*image.Gray16, uint16, uint16) {

	lightest := uint16(math.MaxUint16)
	var darkest uint16

	result := image.NewGray16(img.Bounds())

	for i := img.Bounds().Min.X; i < img.Bounds().Dx(); i++ {
		for j := img.Bounds().Min.Y; j < img.Bounds().Dy(); j++ {
			result.Set(i, j, color.Gray16Model.Convert(img.At(i, j)))

			c := result.Gray16At(i, j).Y

			if c > darkest {
				darkest = c
			}
			if c < lightest {
				lightest = c
			}
		}
	}

	return result, lightest, darkest
}

func ThresholdGrayscale(img *image.Gray16, threshold uint16) *image.Gray16 {
	result := image.NewGray16(img.Bounds())

	for i := img.Bounds().Min.X; i < img.Bounds().Dx(); i++ {
		for j := img.Bounds().Min.Y; j < img.Bounds().Dy(); j++ {
			if img.Gray16At(i, j).Y <= threshold {
				result.SetGray16(i, j, color.Black)
			} else {
				result.SetGray16(i, j, color.White)
			}
		}
	}

	return result
}

// func TrimDistraction(img *image.Gray16, size int) *image.Gray16 {

// 	result := image.NewGray16(img.Bounds())

// 	areas := make([]image.Rectangle, 0)

// 	anchorX := img.Bounds().Min.X
// 	anchorY := img.Bounds().Min.Y

// 	for j := anchorY; j < img.Bounds().Dy(); j++ {
// 		for i := anchorX; i < img.Bounds().Dx(); i++ {
// 			line := findHorizontalContinuousLine(img, i, j)
// 			if line > 0 {

// 			}
// 		}
// 	}

// 	return result
// }

// func findContinuousArea(img *image.Gray16, x, y int) *image.Rectangle {

// 	var result *image.Rectangle

// 	for j := y; j < img.Bounds().Dy(); j++ {
// 		for i := x; i < img.Bounds().Dx(); i++ {

// 		}
// 	}

// 	return nil
// }

// func findHorizontalContinuousLine(img *image.Gray16, x, y int) (begin int, end int) {

// 	if img.Gray16At(x, y).Y != color.Black.Y {
// 		return -1, -1
// 	}

// 	begin = x
// 	end = x

// 	for i := x; i >= img.Bounds().Min.X; i-- {
// 		if img.Gray16At(i, y).Y == color.Black.Y {
// 			begin--
// 		}
// 	}

// 	for i := x; i < img.Bounds().Dx(); i++ {
// 		if img.Gray16At(i, y).Y == color.Black.Y {
// 			end++
// 		}
// 	}

// 	return
// }

// func findVerticalContinuousLine(img *image.Gray16, x, y int) int {
// 	length := 0
// 	for j := y; j < img.Bounds().Dy(); j++ {
// 		if img.Gray16At(x, j).Y == color.Black.Y {
// 			length++
// 		}
// 	}

// 	return length
// }
