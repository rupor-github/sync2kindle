package thumbs

import (
	"path/filepath"
	"runtime/debug"

	"go.uber.org/zap"

	"s2k/config"
	"s2k/thumbs/kfx"
	"s2k/thumbs/mobi"
)

func ExtractThumbnail(fname string, params *config.ThumbnailsConfig, log *zap.Logger) (name string) {

	defer func() {
		// there may exist files we cannot parse
		if rec := recover(); rec != nil {
			log.Debug("Thumbnail processing ended with panic", zap.String("file", fname), zap.Any("record", rec), zap.ByteString("stack", debug.Stack()))
		}
	}()

	switch filepath.Ext(fname) {
	case ".mobi", ".azw3":
		r, err := mobi.NewReader(fname, params.Width, params.Height, log)
		if err != nil {
			log.Warn("Thumbnail extraction failed", zap.String("file", fname), zap.Error(err))
		} else {
			name, err = r.SaveResult(params.Dir)
			if err != nil {
				log.Warn("Thumbnail saving failed", zap.String("file", fname), zap.Error(err))
			}
		}
	case ".kfx":
		r, err := kfx.NewReader(fname, params.Width, params.Height, log)
		if err != nil {
			log.Warn("Thumbnail extraction failed", zap.String("file", fname), zap.Error(err))
		} else {
			name, err = r.SaveResult(params.Dir)
			if err != nil {
				log.Warn("Thumbnail saving failed", zap.String("file", fname), zap.Error(err))
			}
		}
	default:
		// ignore - not supported
	}
	return
}
