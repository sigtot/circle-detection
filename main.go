package main

import (
	"fmt"
	"github.com/sigtot/kalman"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/mat"
	"image"
	"image/color"
	"log"
	"time"
)

func main() {
	webcam, _ := gocv.OpenVideoCapture(1)
	defer webcam.Close()

	window := gocv.NewWindow("Circles")
	window2 := gocv.NewWindow("Canny")

	A := mat.NewDense(4, 4, []float64{1, 0, 0, 0.1677, 0, 1, 0.1677, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	B := mat.NewDense(4, 1, []float64{0, 0.0001406, 3 * 0.1677, 0})
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

	rot := float64(95)
	orangeBelow := gocv.NewScalar(rot, 50, 10, 0)
	orangeAbove := gocv.NewScalar(rot+20, 150, 150, 0)

	gocv.CvtColor(img, &cimg, gocv.ColorRGBToHLS)

	gocv.InRangeWithScalar(cimg, orangeBelow, orangeAbove, &cimg)

	gocv.Canny(cimg, &cimg, 5, 5)

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()
	gocv.Dilate(cimg, &cimg, kernel)

	contours := gocv.FindContours(cimg, gocv.RetrievalList, gocv.ChainApproxSimple)

	statusColor := color.RGBA{255, 0, 0, 0}
	for i, c := range contours {
		area := gocv.ContourArea(c)
		if area < 100 {
			continue
		}

		gocv.DrawContours(&cimg, contours, i, statusColor, 2)
	}

	circles := gocv.NewMat()
	defer circles.Close()

	gocv.HoughCirclesWithParams(
		cimg,
		&circles,
		gocv.HoughGradient,
		1, // dp
		float64(img.Rows()/8), // minDist
		75, // param1
		10, // param2
		25, // minRadius
		30, // maxRadius
	)

	blue := color.RGBA{0, 0, 255, 0}
	red := color.RGBA{255, 0, 0, 0}
	redPred := color.RGBA{255, 0, 0, 150}

	for i := 0; i < circles.Cols(); i++ {
		v := circles.GetVecfAt(0, i)
		// if circles are found
		if len(v) > 2 {
			x := int(v[0])
			y := int(v[1])
			r := int(v[2])

			output := mat.NewVecDense(2, []float64{float64(x), float64(y)})
			f.AddOutput(output)

			gocv.Circle(&img, image.Pt(x, y), r, blue, 2)
			gocv.Circle(&img, image.Pt(x, y), 2, red, 3)

			fmt.Printf("Found circle: (%d,%d)\n", x, y)
			// Draw 10 predicted positions
			for i := 0; i < 100; i++ {
				aPostStateEst := f.APostStateEst(f.CurrentK() + i)
				predX := aPostStateEst.At(0, 0)
				predY := aPostStateEst.At(1, 0)
				gocv.Circle(&img, image.Pt(int(predX), int(predY)), 2, redPred, 2)
			}
		}
	}
	window.IMShow(img)
	cannyWindow.IMShow(cimg)
	window.WaitKey(1)
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
