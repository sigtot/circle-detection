package main

import (
	"fmt"
	"github.com/sigtot/kalman"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/mat"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"time"
)

const y0 = 50
const xLeft = 90
const xRight = 540
const clusterThresh = 10
const boxPadding = 30

func main() {
	webcam, _ := gocv.OpenVideoCapture(1)
	defer webcam.Close()

	window := gocv.NewWindow("Mask")
	window2 := gocv.NewWindow("Web camera")

	A := mat.NewDense(4, 4, []float64{1, 0, 0, 0.1677, 0, 1, 0.1677, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	B := mat.NewDense(4, 1, []float64{0, 0.0001406, 4 * 0.1677, 0})
	C := mat.NewDense(2, 4, []float64{1, 0, 0, 0, 0, 1, 0, 0})
	D := mat.NewDense(2, 1, []float64{0, 0})
	G := mat.NewDiagonal(4, []float64{0.2, 0.2, 0.1, 0.1})
	H := mat.NewDense(2, 2, []float64{0.1, 0.1, 0.2, 0.2})
	R := mat.NewDiagonal(2, []float64{10, 10})
	Q := mat.NewDiagonal(4, []float64{0.2, 0.2, 1, 1})

	aPriErrCovInit := mat.NewDense(4, 4, []float64{1, 0, 2, 0, 0, 1, 0, 2, 2, 0, 1, 0, 0, 2, 0, 1})
	aPriStateEstInit := mat.NewVecDense(4, []float64{300, 200, 0, 0})
	input := mat.NewVecDense(1, []float64{-4})
	outputInit := mat.NewVecDense(2, []float64{300, 200})

	f := kalman.NewFilter(A, B, C, D, H, G, R, Q, aPriErrCovInit, aPriStateEstInit, input, outputInit)

	ticker := time.NewTicker(16 * time.Millisecond)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			drawPredictImage(window, window2, webcam, &f)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func findGameBox(webcam *gocv.VideoCapture) {
	window := gocv.NewWindow("Game box Canny")
	img := gocv.NewMat()
	defer img.Close()

	window2 := gocv.NewWindow("Gray scale image")

	if ok := webcam.Read(&img); !ok {
		log.Fatal("Webcam closed")
	}

	if img.Empty() {
		log.Println("Warning: Read empty image when trying to find game box")
		return
	}

	grayImg := gocv.NewMat()
	gocv.CvtColor(img, &grayImg, gocv.ColorRGBAToGray)

	gocv.MedianBlur(grayImg, &grayImg, 3)
	canny := gocv.NewMat()
	defer canny.Close()

	gocv.Canny(grayImg, &canny, 3, 3)

	erodeKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(1, 3))
	gocv.Erode(canny, &canny, erodeKernel)

	dilateKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	gocv.Dilate(canny, &canny, dilateKernel)

	lines := gocv.NewMat()
	gocv.HoughLinesP(canny, &lines, 1, 3.14/180, 80)

	var xValues []int
	for i := 0; i < lines.Rows(); i++ {
		pt1 := image.Pt(int(lines.GetVeciAt(i, 0)[0]), int(lines.GetVeciAt(i, 0)[1]))
		pt2 := image.Pt(int(lines.GetVeciAt(i, 0)[2]), int(lines.GetVeciAt(i, 0)[3]))
		if math.Sqrt(math.Pow(float64(pt2.X-pt1.X), 2)+math.Pow(float64(pt2.Y-pt1.Y), 2)) > 30 {
			x0 := findIntersection(pt1, pt2, img.Rows()-y0)
			xValues = append(xValues, x0)
			gocv.Line(&img, pt1, image.Pt(x0, y0), color.RGBA{0, 255, 0, 50}, 2)
		}
	}

	xClusters := []int{0}
	var clusterValues []int
	if len(xValues) > 0 {
		sort.Ints(xValues)
		for i := 0; i < len(xValues); i++ {
			if xValues[i] > xClusters[len(xClusters)-1]+clusterThresh {
				xClusters = append(xClusters, xValues[i])
				clusterValues = []int{}
			} else {
				clusterValues = append(clusterValues, xValues[i])
				xClusters[len(xClusters)-1] = average(clusterValues)
			}
		}
	}

	blue := color.RGBA{0, 0, 255, 0}
	for _, x := range xClusters {
		gocv.Circle(&img, image.Pt(x, y0), 2, blue, 2)
	}

	window.IMShow(canny)
	window2.IMShow(img)
	window.WaitKey(1)
}

func average(values []int) int {
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum / len(values)
}

func findIntersection(pt1 image.Point, pt2 image.Point, y0 int) int {
	return pt1.X + (pt1.X-pt2.X)/(pt1.Y-pt2.Y)*(y0-pt1.Y)
}

func drawPredictImage(window *gocv.Window, cannyWindow *gocv.Window, webcam *gocv.VideoCapture, f *kalman.Filter) {
	img := gocv.NewMat()
	defer img.Close()
	if ok := webcam.Read(&img); !ok {
		log.Fatal("Webcam closed")
	}

	if img.Empty() {
		log.Println("Warning: Read empty image")
		return
	}

	cimg := gocv.NewMat()
	defer cimg.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	hsvImg := gocv.NewMat()
	defer hsvImg.Close()

	rot := float64(10)
	hlsBlueBelow := gocv.NewScalar(rot, 80, 80, 0)
	hlsBlueAbove := gocv.NewScalar(rot+15, 255, 255, 0)

	gocv.CvtColor(img, &hsvImg, gocv.ColorRGBAToBGR)
	gocv.CvtColor(hsvImg, &hsvImg, gocv.ColorBGRToHLS)
	gocv.InRangeWithScalar(hsvImg, hlsBlueBelow, hlsBlueAbove, &mask)

	c := gocv.FindContours(mask, gocv.RetrievalExternal, gocv.ChainApproxNone)
	largestContour := 0
	largestContourCount := 0
	for i := range c {
		length := len(c[i])
		if length > largestContourCount {
			largestContour = i
			largestContourCount = length
		}
	}

	red := color.RGBA{255, 0, 0, 0}
	green := color.RGBA{0, 255, 0, 0}
	blue := color.RGBA{0, 0, 255, 0}
	yellow := color.RGBA{255, 255, 0, 0}

	if len(c) > 0 {
		rect := gocv.MinAreaRect(c[largestContour])

		x := rect.Center.X
		y := rect.Center.Y

		gocv.Circle(&img, image.Pt(x, y), 7, red, 13)

		output := mat.NewVecDense(2, []float64{float64(x), float64(y)})
		f.AddOutput(output)

		// Draw 100 predicted positions
		lastPredY := f.APostStateEst(f.CurrentK()).At(1, 0)
		for i := 0; i < 100; i++ {
			aPostStateEst := f.APostStateEst(f.CurrentK() + i)
			predX := aPostStateEst.At(0, 0)
			predY := aPostStateEst.At(1, 0)
			gocv.Circle(&img, image.Pt(int(predX), int(predY)), 2, green, 2)

			// Ball crosses bottom line at this k
			if lastPredY > y0 && predY < y0 {
				cartReference := confinedToRange(int(predX), xLeft, xRight)
				gocv.Circle(&img, image.Pt(cartReference, (int(predY)+int(lastPredY))/2), 5, yellow, 5)
			}
			lastPredY = predY
		}
	}

	// Draw bottom line
	gocv.Line(&img, image.Pt(xLeft-boxPadding, y0), image.Pt(xRight+boxPadding, y0), blue, 2)
	gocv.Line(&img, image.Pt(xLeft, y0), image.Pt(xRight, y0), green, 2)

	window.IMShow(mask)
	cannyWindow.IMShow(img)
	window.WaitKey(1)
}

func confinedToRange(value, min, max int) int {
	if value < min {
		return min
	}

	if value > max {
		return max
	}

	return value
}
func printLocation(webcam *gocv.VideoCapture) {
	img := gocv.NewMat()
	if ok := webcam.Read(&img); !ok {
		log.Fatal("Webcam closed")
	}

	if img.Empty() {
		log.Println("Warning: Read empty image")
		return
	}

	gocv.CvtColor(img, &img, gocv.ColorRGBToGray)

	circles := gocv.NewMat()
	defer circles.Close()

	gocv.HoughCirclesWithParams(
		img,
		&circles,
		gocv.HoughStandard,
		1, // dp
		float64(img.Rows()/8), // minDist
		75, // param1
		10, // param2
		25, // minRadius
		28, // maxRadius
	)

	for i := 0; i < circles.Cols(); i++ {
		v := circles.GetVecfAt(0, i)
		// if circles are found
		if len(v) > 2 {
			x := int(v[0])
			y := int(v[1])
			r := int(v[2])
			fmt.Printf("pos=(%d, %d) r=%d\n", x, y, r)
		}
	}
}

func showImg(webcam *gocv.VideoCapture, window *gocv.Window) {
	img := gocv.NewMat()
	if ok := webcam.Read(&img); !ok {
		log.Fatal("Webcam closed")
	}

	if img.Empty() {
		log.Println("Warning: Read empty image")
		return
	}

	cimg := gocv.NewMat()
	defer cimg.Close()

	gocv.CvtColor(img, &img, gocv.ColorRGBToGray)
	gocv.CvtColor(img, &cimg, gocv.ColorGrayToBGR)

	circles := gocv.NewMat()
	defer circles.Close()

	gocv.HoughCirclesWithParams(
		img,
		&circles,
		gocv.HoughGradient,
		1, // dp
		float64(img.Rows()/8), // minDist
		75, // param1
		25, // param2
		25, // minRadius
		28, // maxRadius
	)

	blue := color.RGBA{0, 0, 255, 0}
	red := color.RGBA{255, 0, 0, 0}

	for i := 0; i < circles.Cols(); i++ {
		v := circles.GetVecfAt(0, i)
		// if circles are found
		if len(v) > 2 {
			x := int(v[0])
			y := int(v[1])
			r := int(v[2])

			gocv.Circle(&cimg, image.Pt(x, y), r, blue, 2)
			gocv.Circle(&cimg, image.Pt(x, y), 2, red, 3)
		}
	}

	window.IMShow(cimg)
	window.WaitKey(1)
}
