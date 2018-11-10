// Example program that uses blakjack/webcam library
// for working with V4L2 devices.
// The application reads frames from device and writes them to stdout
// If your device supports motion formats (e.g. H264 or MJPEG) you can
// use it's output as a video stream.
// Example usage: go run stdout_streamer.go | vlc -
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/blackjack/webcam"
	"github.com/nsf/termbox-go"

	"golang.org/x/image/draw"
)

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

const mainUsage = `%s streams a given webcam device to your terminal as a mesh of pixels.

`

func mainInner() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	deviceFlag := fs.String("device", "/dev/video0", "The webcam device to open")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, mainUsage, filepath.Base(os.Args[0]))
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		fmt.Fprintf(os.Stderr, "\n")
		return fmt.Errorf("no positional arguments expected")
	}

	err := termbox.Init()
	if err != nil {
		return fmt.Errorf("failed to init termbox: %s", err)
	}
	termbox.SetOutputMode(termbox.Output256)
	defer termbox.Close()

	cam, err := webcam.Open(*deviceFlag)
	if err != nil {
		return fmt.Errorf("failed to open webcam '%s': %s", *deviceFlag, err)
	}
	defer cam.Close()

	formatDesc := cam.GetSupportedFormats()
	var mpegFormat *webcam.PixelFormat
	for f, v := range formatDesc {
		if v == "Motion-JPEG" {
			x := f
			mpegFormat = &x
		}
	}
	if mpegFormat == nil {
		return fmt.Errorf("webcam does not support Motion-JPEG mode (%#v)", formatDesc)
	}
	frames := FrameSizes(cam.GetSupportedFrameSizes(*mpegFormat))
	sort.Sort(frames)

	var chosenSize webcam.FrameSize
	termWidth, termHeight := termbox.Size()
	for _, value := range frames {
		chosenSize = value
		if int(value.MinWidth) > termWidth && int(value.MinHeight) > termHeight {
			break
		}
	}

	f, w, h, err := cam.SetImageFormat(*mpegFormat, uint32(chosenSize.MaxWidth), uint32(chosenSize.MaxHeight))

	if err != nil {
		return fmt.Errorf("failed to set image format: %s", err)
	} else {
		fmt.Fprintf(os.Stderr, "Resulting image format: %s (%dx%d)\n", formatDesc[f], w, h)
	}

	err = cam.StartStreaming()
	if err != nil {
		return fmt.Errorf("failed to start streaming: %s", err)
	}

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
			if ev.Type == termbox.EventKey {
				switch ev.Key {
				case termbox.KeyCtrlC:
					fallthrough
				case termbox.KeyEsc:
					break A
				}
			}
		default:
			err = cam.WaitForFrame(timeout)

			switch err.(type) {
			case nil:
			case *webcam.Timeout:
				fmt.Fprint(os.Stderr, err.Error())
				continue
			default:
				return fmt.Errorf("failed to get frame: %s", err)
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
				return fmt.Errorf("failed to decode frame: %s", err)
			}
		}
	}
	return nil
}

func main() {
	if err := mainInner(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
