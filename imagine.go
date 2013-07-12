package main

import (
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"github.com/keep94/weblogs"
	"github.com/gorilla/context"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"runtime"
)

type Decoder func(r io.Reader) (image.Image, error)
type Encoder func(w io.Writer, m image.Image) error

func jpgEncoder(w io.Writer, m image.Image) error {
	return jpeg.Encode(w, m, nil)
}

func decoder(ext string) (Decoder, error) {
	if ext == "" {
		return nil, errors.New("extension is empty")
	}
	if ext == ".jpeg" || ext == ".jpg" {
		return jpeg.Decode, nil
	} else if ext == ".png" {
		return png.Decode, nil
	} else if ext == ".gif" {
		return gif.Decode, nil
	}
	return nil, errors.New(fmt.Sprintf("Unsupported type: %s", ext))
}

func encoder(format string) Encoder {
	if format == "jpeg" {
		return jpgEncoder
	} else if format == "png" {
		return png.Encode
	}
	return jpgEncoder
}

type ResizeRequest struct {
	URL     *url.URL
	Width   uint
	Height  uint
	Encoder Encoder
	Decoder Decoder
	Interp  resize.InterpolationFunction
}

func createRequest(r *http.Request) (*ResizeRequest, error) {
	request := new(ResizeRequest)
	source := r.URL.Query().Get("source")
	if source == "" {
		return nil, errors.New("No source defined in query params")
	}
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	width := r.URL.Query().Get("width")
	if width != "" {
		w, err := strconv.Atoi(width)
		if err != nil {
			return nil, err
		}
		if w >= 0 {
			request.Width = uint(w)
		} else {
			return nil, errors.New("Width must be a 0 or positive")
		}
	} else {
		request.Width = 0
	}
	height := r.URL.Query().Get("height")
	if height != "" {
		h, err := strconv.Atoi(height)
		if err != nil {
			return nil, err
		}
		if h >= 0 {
			request.Height = uint(h)
		} else {
			return nil, errors.New("Height must be a 0 or positive")
		}
	} else {
		request.Height = 0
	}
	request.URL = u
	request.Decoder, err = decoder(Ext(request.URL.Path))
	if err != nil {
		return nil, err
	}
	encodeType := r.URL.Query().Get("encode")
	request.Encoder = encoder(encodeType)
	request.Interp = resize.NearestNeighbor

	return request, nil
}

func Ext(path string) string {
	for i := len(path) - 1; i >= 0 && '/' != path[i]; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

func imagine(r *ResizeRequest, w http.ResponseWriter) error {
	resp, err := http.Get(r.URL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	img, err := r.Decoder(resp.Body)
	if err != nil {
		return err
	}
	m := resize.Resize(r.Width, r.Height, img, r.Interp)
	err = r.Encoder(w, m)
	if err != nil {
		return err
	}
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	request, err := createRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		if request.Width == 0 && request.Height == 0 {
			http.Redirect(w, r, request.URL.String(), http.StatusMovedPermanently)
			return
		}
		err = imagine(request, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	response := "<healthcheck>ok</healthcheck>"
	w.Header().Add("Content-Type", "text/xml")
	w.Header().Add("Content-Length", strconv.Itoa(len(response)))
	io.WriteString(w, response)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())


	http.HandleFunc("/imagine", handler)
	http.HandleFunc("/healthcheck", healthcheck)

	// http.DefaultServeMux has /imagine and /healthcheck defined already
	accessHandler := context.ClearHandler(weblogs.Handler(http.DefaultServeMux))
	http.ListenAndServe(":8080", accessHandler)
}
