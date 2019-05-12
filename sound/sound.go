package sound

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/wav"
)

var soundDir = "/home/seebs/src/go/src/seebs.net/modus/sound/sounds"

var ac *audio.Context

type Voice struct {
	tones [][]byte
	ap    []*audio.Player
}

func NewVoice(name string, polyphony int) (*Voice, error) {
	var err error
	if ac == nil {
		ac, err = audio.NewContext(48000)
		if err != nil {
			return nil, err
		}
	}
	dir, err := os.Open(soundDir)
	if err != nil {
		return nil, err
	}
	names, err := dir.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	max := 0
	tTones := make(map[int][]byte, len(names))
	for _, file := range names {
		if strings.HasPrefix(file, name) {
			raw, err := ioutil.ReadFile(filepath.Join(soundDir, file))
			if err != nil {
				return nil, err
			}
			ext := strings.LastIndex(file, ".")
			if ext != -1 {
				file = file[:ext]
			}
			fileIndex := strings.Replace(file, name, "", 1)
			idx, err := strconv.Atoi(fileIndex)
			if idx > max {
				max = idx
			}
			tTones[idx] = raw
		}
	}
	v := Voice{}
	v.tones = make([][]byte, max)
	for i := 0; i < max; i++ {
		// for historical reasons, tones are numbered 1-16
		s, err := wav.Decode(ac, audio.BytesReadSeekCloser(tTones[i+1]))
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(s)
		if err != nil {
			return nil, err
		}
		v.tones[i] = b
	}
	return &v, nil
}

func (v *Voice) Play(tone, volume int) {
	if v == nil {
		return
	}
	// NewPlayerFromBytes can't error these days
	ap, _ := audio.NewPlayerFromBytes(ac, v.tones[tone%len(v.tones)])
	ap.SetVolume(float64(volume) / 100)
	ap.Play()
}
