// Example program that uses blakjack/webcam library
// for working with V4L2 devices.
// The application reads frames from device and writes them to stdout
// If your device supports motion formats (e.g. H264 or MJPEG) you can
// use it's output as a video stream.
// Example usage: go run stdout_streamer.go | vlc -
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"sort"

	"github.com/blackjack/webcam"
	termbox "github.com/nsf/termbox-go"

	"golang.org/x/image/draw"
)

func readChoice(s string) int {
	var i int
	for true {
		print(s)
		_, err := fmt.Scanf("%d\n", &i)
		if err != nil || i < 1 {
			println("Invalid input. Try again")
		} else {
			break
		}
	}
	return i
}

type FrameSizes []webcam.FrameSize

func (slice FrameSizes) Len() int {
	return len(slice)
}

//For sorting purposes
func (slice FrameSizes) Less(i, j int) bool {
	ls := slice[i].MaxWidth * slice[i].MaxHeight
	rs := slice[j].MaxWidth * slice[j].MaxHeight
	return ls < rs
}

//For sorting purposes
func (slice FrameSizes) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func main() {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		panic(err.Error())
	}
	defer cam.Close()

	formatDesc := cam.GetSupportedFormats()
	var formats []webcam.PixelFormat
	for f := range formatDesc {
		formats = append(formats, f)
	}

	println("Available formats: ")
	for i, value := range formats {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, formatDesc[value])
	}

	choice := readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(formats)))
	format := formats[choice-1]

	fmt.Fprintf(os.Stderr, "Supported frame sizes for format %s\n", formatDesc[format])
	frames := FrameSizes(cam.GetSupportedFrameSizes(format))
	sort.Sort(frames)

	for i, value := range frames {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, value.GetString())
	}
	choice = readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(frames)))
	size := frames[choice-1]

	f, w, h, err := cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))

	if err != nil {
		panic(err.Error())
	} else {
		fmt.Fprintf(os.Stderr, "Resulting image format: %s (%dx%d)\n", formatDesc[f], w, h)
	}

	println("Press Enter to start streaming")
	fmt.Scanf("\n")
	err = cam.StartStreaming()
	if err != nil {
		panic(err.Error())
	}

	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetOutputMode(termbox.Output256)
	defer termbox.Close()

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

	timeout := uint32(5) //5 seconds

A:
	for {
		select {
		case ev := <-eventQueue:
			if ev.Type == termbox.EventKey && ev.Key == termbox.KeyEsc {
				break A
			}
		default:
			err = cam.WaitForFrame(timeout)

			switch err.(type) {
			case nil:
			case *webcam.Timeout:
				fmt.Fprint(os.Stderr, err.Error())
				continue
			default:
				panic(err.Error())
			}

			frame, err := cam.ReadFrame()
			if len(frame) != 0 {

				img, err := jpeg.Decode(bytes.NewReader(frame))
				if err != nil {
					fmt.Fprint(os.Stderr, err.Error())
				} else {
					szx, szy := termbox.Size()
					img2 := image.NewRGBA(image.Rect(0, 0, szx, szy))
					draw.ApproxBiLinear.Scale(img2, img2.Bounds(), img, img.Bounds(), draw.Over, nil)
					for y := 0; y < szy; y++ {
						for x := 0; x < szx; x++ {
							cr, cg, cb, _ := img2.At(x, y).RGBA()
							termbox.SetCell(x, y, ' ', termbox.ColorDefault, termbox.Attribute(rgbaToAnsi(cr, cg, cb)))
						}
					}
					termbox.Flush()
				}

			} else if err != nil {
				panic(err.Error())
			}
		}
	}
}

func rgbaToAnsi(r, g, b uint32) uint16 {
	var fr, fg, fb float64
	fr = float64(r) / 65535.0
	fg = float64(g) / 65535.0
	fb = float64(b) / 65535.0

	if fr == fg && fr == fb && fr != 0.0 && fr != 1.0 {
		return 0xe9 + uint16(math.Round(24*fr))
	}

	return 0x11 + 36*uint16(math.Round(fr*5)) +
		6*uint16(math.Round(fg*5)) +
		uint16(math.Round(fb*5))
}
