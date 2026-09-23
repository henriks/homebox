package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageScalingEnvironment(t *testing.T) {
	t.Setenv("HBOX_IMAGE_SCALING_WIDTH", "1280")
	t.Setenv("HBOX_IMAGE_SCALING_HEIGHT", "720")
	t.Setenv("HBOX_IMAGE_SCALING_QUALITY", "65")
	cfg, err := New("test", "test")
	require.NoError(t, err)
	require.Equal(t, ImageScaling{Width: 1280, Height: 720, Quality: 65}, cfg.ImageScaling)
}

func TestImageScalingRejectsInvalidQuality(t *testing.T) {
	t.Setenv("HBOX_IMAGE_SCALING_QUALITY", "101")
	_, err := New("test", "test")
	require.ErrorContains(t, err, "image scaling quality")
}
