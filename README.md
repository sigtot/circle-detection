# Circle Detection
Detect circles seen by a webcam with OpenCV
![Detected ping pong balls](https://i.imgur.com/A9CQpPj.png)
## Requirements
* OpenCV
* GoCV

Both can be installed by following the steps below (taken from [this guide](https://gocv.io/getting-started/linux/))

#### Install gocv package
```bash
go get -u -d gocv.io/x/gocv
```

#### Install OpenCV, and do other important stuff
```bash
cd $GOPATH/src/gocv.io/x/gocv
make install
```

## What's going on here?
The webcam ID is hardcoded into the code and can be changed by changing the parameter in this line
```go
webcam, _ := gocv.OpenVideoCapture(1)
```

The [Circle Hough Transform](https://en.wikipedia.org/wiki/Circle_Hough_Transform) which is used to detect the circles
can be configured by changing the parameters in the following code block
```go
gocv.HoughCirclesWithParams(
    img,
    &circles,
    gocv.HoughGradient,
    1, // dp
    float64(img.Rows()/8), // minDist
    75, // param1
    40, // param2
    3, // minRadius
    60,  // maxRadius
)
```
An explanation of the different parameters can be found on the OpenCV documentation pages
[here](https://docs.opencv.org/master/dd/d1a/group__imgproc__feature.html#ga47849c3be0d0406ad3ca45db65a25d2d)  
