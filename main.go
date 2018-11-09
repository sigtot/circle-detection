package main

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"log"
	"time"
)

func main() {
	webcam, _ := gocv.OpenVideoCapture(0)
	defer webcam.Close()

	//window := gocv.NewWindow("Circles")

	ticker := time.NewTicker(16 * time.Millisecond)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			printLocation(webcam)
		case <-quit:
			ticker.Stop()
			return
		}
	}
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
		gocv.HoughGradient,
		1, // dp
		float64(img.Rows()/8), // minDist
		75, // param1
		22, // param2
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
