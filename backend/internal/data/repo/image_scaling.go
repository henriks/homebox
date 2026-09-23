package repo

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/evanoberholster/imagemeta"
	"github.com/gen2brain/avif"
	"github.com/gen2brain/heic"
	"github.com/gen2brain/jpegxl"
	"github.com/gen2brain/webp"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/pkgs/utils"
	"golang.org/x/image/draw"
)

// ErrInvalidPhoto is returned when a photo cannot be safely converted to WebP.
var ErrInvalidPhoto = errors.New("invalid or unsupported photo")

func scalePhoto(doc ItemCreateAttachment, settings config.ImageScaling) (ItemCreateAttachment, error) {
	content, err := io.ReadAll(doc.Content)
	if err != nil {
		return doc, err
	}
	if len(content) == 0 {
		return doc, fmt.Errorf("%w: empty file", ErrInvalidPhoto)
	}

	mimeType := http.DetectContentType(content[:min(512, len(content))])
	if mimeType == "application/octet-stream" {
		switch strings.ToLower(filepath.Ext(doc.Title)) {
		case ".heic", ".heif":
			mimeType = "image/heic"
		case ".avif":
			mimeType = "image/avif"
		case ".jxl":
			mimeType = "image/jxl"
		}
	}
	var dimensions image.Config
	switch mimeType {
	case "image/jpeg", "image/png":
		dimensions, _, err = image.DecodeConfig(bytes.NewReader(content))
	case "image/gif":
		dimensions, err = gif.DecodeConfig(bytes.NewReader(content))
	case "image/webp":
		dimensions, err = webp.DecodeConfig(bytes.NewReader(content))
	case "image/avif":
		dimensions, err = avif.DecodeConfig(bytes.NewReader(content))
	case "image/heic", "image/heif":
		dimensions, err = heic.DecodeConfig(bytes.NewReader(content))
	case "image/jxl":
		dimensions, err = jpegxl.DecodeConfig(bytes.NewReader(content))
	default:
		return doc, fmt.Errorf("%w: %s", ErrInvalidPhoto, mimeType)
	}
	if err != nil {
		return doc, fmt.Errorf("%w: %v", ErrInvalidPhoto, err)
	}
	if dimensions.Width < 1 || dimensions.Height < 1 || dimensions.Width > 10000 || dimensions.Height > 10000 {
		return doc, fmt.Errorf("%w: invalid image dimensions", ErrInvalidPhoto)
	}

	var img image.Image
	switch mimeType {
	case "image/jpeg", "image/png":
		img, _, err = image.Decode(bytes.NewReader(content))
	case "image/gif":
		var frames *gif.GIF
		frames, err = gif.DecodeAll(bytes.NewReader(content))
		if err == nil {
			if len(frames.Image) != 1 {
				return doc, fmt.Errorf("%w: animated GIF", ErrInvalidPhoto)
			}
			img = frames.Image[0]
		}
	case "image/webp":
		var frames *webp.WEBP
		frames, err = webp.DecodeAll(bytes.NewReader(content))
		if err == nil {
			if len(frames.Image) != 1 {
				return doc, fmt.Errorf("%w: animated WebP", ErrInvalidPhoto)
			}
			img = frames.Image[0]
		}
	case "image/avif":
		var frames *avif.AVIF
		frames, err = avif.DecodeAll(bytes.NewReader(content), avif.Options{AutoRotate: true})
		if err == nil {
			if len(frames.Image) != 1 {
				return doc, fmt.Errorf("%w: animated AVIF", ErrInvalidPhoto)
			}
			img = frames.Image[0]
		}
	case "image/heic", "image/heif":
		var frames *heic.HEIC
		frames, err = heic.DecodeAll(bytes.NewReader(content))
		if err == nil {
			if len(frames.Image) != 1 {
				return doc, fmt.Errorf("%w: animated HEIC", ErrInvalidPhoto)
			}
			img = frames.Image[0]
		}
	case "image/jxl":
		var frames *jpegxl.JXL
		frames, err = jpegxl.DecodeAll(bytes.NewReader(content))
		if err == nil {
			if len(frames.Image) != 1 {
				return doc, fmt.Errorf("%w: animated JPEG XL", ErrInvalidPhoto)
			}
			img = frames.Image[0]
		}
	}
	if err != nil {
		return doc, fmt.Errorf("%w: %v", ErrInvalidPhoto, err)
	}
	if img == nil || img.Bounds().Dx() < 1 || img.Bounds().Dy() < 1 || img.Bounds().Dx() > 10000 || img.Bounds().Dy() > 10000 {
		return doc, fmt.Errorf("%w: invalid image dimensions", ErrInvalidPhoto)
	}

	if mimeType != "image/avif" && mimeType != "image/jxl" {
		if meta, metaErr := imagemeta.Decode(bytes.NewReader(content)); metaErr == nil {
			img = utils.ApplyOrientation(img, uint16(meta.IFD0.Orientation))
		}
	}
	w, h := calculateThumbnailDimensions(img.Bounds().Dx(), img.Bounds().Dy(), settings.Width, settings.Height)
	if w < 1 || h < 1 {
		return doc, fmt.Errorf("invalid image scaling dimensions")
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := webp.Encode(&out, dst, webp.Options{Quality: settings.Quality, Lossless: false}); err != nil {
		return doc, fmt.Errorf("encode scaled photo: %w", err)
	}
	name := strings.TrimSuffix(doc.Title, filepath.Ext(doc.Title)) + ".webp"
	return ItemCreateAttachment{Title: name, Content: bytes.NewReader(out.Bytes())}, nil
}
