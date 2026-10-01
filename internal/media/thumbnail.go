package media

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
)

const thumbMaxWidth = 320

func ffmpegPath() (string, error) {
	p, err := cmdutil.Resolve("ffmpeg")
	if errors.Is(err, cmdutil.ErrNotConfigured) {
		return "", err
	}
	if err != nil {
		return "", errors.New("ffmpeg não encontrado. Instale o ffmpeg para usar o vidctl.")
	}
	return p, nil
}

func thumbCacheDir() (string, error) {
	dir := filepath.Join(os.TempDir(), "vidctl-thumbs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func thumbCachePath(path string, size, mtimeUnix int64) (string, error) {
	dir, err := thumbCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", path, size, mtimeUnix)))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".jpg"), nil
}

// ThumbDataURL extracts a single frame from the video and returns it as a
// JPEG data-URL. The frame is cached in the system temp dir keyed by path,
// size and mtime, so re-selecting the same file skips ffmpeg.
func ThumbDataURL(path string, durationSec float64) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", errors.New("não foi possível ler o arquivo de vídeo: " + path)
	}

	cache, err := thumbCachePath(path, st.Size(), st.ModTime().Unix())
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cache); err != nil {
		bin, err := ffmpegPath()
		if err != nil {
			return "", err
		}
		seek := durationSec / 2
		if seek > 1 {
			seek = 1
		}
		out, err := cmdutil.Command(bin,
			"-v", "error",
			"-ss", fmt.Sprintf("%.3f", seek),
			"-i", path,
			"-frames:v", "1",
			"-vf", fmt.Sprintf("scale=%d:-2", thumbMaxWidth),
			"-y", cache,
		).CombinedOutput()
		if err != nil {
			return "", errors.New("não foi possível gerar a miniatura: " + string(out))
		}
	}

	data, err := os.ReadFile(cache)
	if err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data), nil
}
