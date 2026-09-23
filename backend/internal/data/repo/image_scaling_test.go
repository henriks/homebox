package repo

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gen2brain/webp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/attachment"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestScalePhoto(t *testing.T) {
	settings := config.ImageScaling{Width: 100, Height: 100, Quality: 80}
	for _, tc := range []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{name: "large", width: 400, height: 200, wantWidth: 100, wantHeight: 50},
		{name: "small", width: 40, height: 20, wantWidth: 40, wantHeight: 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := scalePhoto(ItemCreateAttachment{Title: "photo.png", Content: bytes.NewReader(testPNG(t, tc.width, tc.height))}, settings)
			require.NoError(t, err)
			require.Equal(t, "photo.webp", doc.Title)
			img, err := webp.Decode(doc.Content)
			require.NoError(t, err)
			require.Equal(t, tc.wantWidth, img.Bounds().Dx())
			require.Equal(t, tc.wantHeight, img.Bounds().Dy())
		})
	}
}

func TestScalePhotoRejectsInvalidAndAnimated(t *testing.T) {
	settings := config.ImageScaling{Width: 100, Height: 100, Quality: 80}
	_, err := scalePhoto(ItemCreateAttachment{Title: "bad.png", Content: strings.NewReader("not an image")}, settings)
	require.ErrorIs(t, err, ErrInvalidPhoto)

	frames := &gif.GIF{Image: []*image.Paletted{
		image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black}),
		image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.White}),
	}, Delay: []int{10, 10}}
	var animated bytes.Buffer
	require.NoError(t, gif.EncodeAll(&animated, frames))
	_, err = scalePhoto(ItemCreateAttachment{Title: "animated.gif", Content: &animated}, settings)
	require.ErrorIs(t, err, ErrInvalidPhoto)
}

func TestScaledPhotoStoredOnlyWhenEnabled(t *testing.T) {
	ctx := context.Background()
	group, err := tRepos.Groups.GroupCreate(ctx, "scaled-photo", uuid.Nil)
	require.NoError(t, err)
	entity, err := tRepos.Entities.CreateContainer(ctx, group.ID, EntityCreate{Name: "photos"})
	require.NoError(t, err)
	input := testPNG(t, 200, 100)
	original, err := tRepos.Attachments.Create(ctx, entity.ID, ItemCreateAttachment{Title: "original.png", Content: bytes.NewReader(input)}, attachment.TypePhoto, false)
	require.NoError(t, err)
	require.Equal(t, "image/png", original.MimeType)
	require.Equal(t, "original.png", original.Title)

	enabled := true
	_, err = tRepos.Groups.GroupUpdate(ctx, group.ID, GroupUpdate{Name: group.Name, Currency: group.Currency, ScaleImages: &enabled})
	require.NoError(t, err)
	scaled, err := tRepos.Attachments.Create(ctx, entity.ID, ItemCreateAttachment{Title: "scaled.png", Content: bytes.NewReader(input)}, attachment.TypePhoto, false)
	require.NoError(t, err)
	require.Equal(t, "image/webp", scaled.MimeType)
	require.Equal(t, "scaled.webp", scaled.Title)
	content, err := os.ReadFile(filepath.Join(os.TempDir(), scaled.Path))
	require.NoError(t, err)
	_, err = webp.Decode(bytes.NewReader(content))
	require.NoError(t, err)

	other, err := tRepos.Attachments.Create(ctx, entity.ID, ItemCreateAttachment{Title: "other.png", Content: bytes.NewReader(input)}, attachment.TypeAttachment, false)
	require.NoError(t, err)
	require.Equal(t, "image/png", other.MimeType)
	require.Equal(t, "other.png", other.Title)

	before, err := tClient.Attachment.Query().Count(ctx)
	require.NoError(t, err)
	filesBefore, err := os.ReadDir(filepath.Join(os.TempDir(), group.ID.String(), "documents"))
	require.NoError(t, err)
	_, err = tRepos.Attachments.Create(ctx, entity.ID, ItemCreateAttachment{Title: "bad.png", Content: strings.NewReader("bad")}, attachment.TypePhoto, false)
	require.True(t, errors.Is(err, ErrInvalidPhoto))
	after, err := tClient.Attachment.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, before, after)
	filesAfter, err := os.ReadDir(filepath.Join(os.TempDir(), group.ID.String(), "documents"))
	require.NoError(t, err)
	require.Equal(t, len(filesBefore), len(filesAfter))

	t.Cleanup(func() {
		_ = tRepos.Groups.GroupDelete(ctx, group.ID)
	})
}
