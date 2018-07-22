# Scream

A demo of rendering a webcam stream through your terminal.

This is a quick and easy mashup of:

- github.com/blackjack/webcam for the webcam capture
- github.com/nsf/termbox-go for easy terminal rendering
- golang.org/x/image/draw for image scaling

Only works on Linux since the `github.com/blackjack/webcam` dependency only works with V4L2 (Linux media subsystem) webcam sources.

```
$ go get github.com/AstromechZA/scream
```

## Example

If it's not loading on Github, view the image directly.

![wave example](_examples/wave.svg)
