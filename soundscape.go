package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/mp3"
)

// Ambience from the public-domain recordings embedded in assets/sounds,
// looped seamlessly in memory; pink noise is synthesized. This is the native
// port of the former WebAudio Soundscape: same seams, fades and gains.

// seamHint places a loop's seam between beats or after a piece ends: start
// and loop are seconds into the decoded file, fade is the crossfade length.
type seamHint struct {
	start, loop, fade float64
	fixed             bool // false: start each session at a random point
}

var seamHints = map[string]seamHint{
	"clock": {start: 0.25, loop: 30.001, fade: 0.15},
	"knock": {start: 0.25, loop: 29.7, fade: 0.15},
	// The whole Canon, then about two seconds of silence before it starts again.
	"musicbox": {start: 0.1, loop: 226.9, fade: 0.2, fixed: true},
}

var soundStatusText = map[string]string{
	"waves": "海浪正輕輕陪著你", "rain": "窗外彷彿下著細雨", "bubbles": "像在安靜的水族箱旁", "wind": "風從遠處輕輕吹來",
	"fire": "篝火噼啪，安靜又暖", "stream": "溪水在不遠處流過", "pink": "平穩的底噪替你蓋住雜音", "clock": "滴答之間，時間在走",
	"boil": "水在咕嘟，慢慢開著", "knock": "一聲一聲，穩穩地", "thunder": "遠處有雷，雨聲更重了", "cafe": "像坐在安靜的咖啡館角落",
	"musicbox": "一首卡農，慢慢流淌", "morning": "清晨的風和鳥鳴", "summer": "蟬鳴陣陣，夏天正盛", "night": "夏夜蟲鳴，風也安靜",
	"forest": "林子裡，風穿過樹葉", "deepsea": "很深很深的海底", "desert": "風捲細沙，安靜遼闊", "traffic": "遠處有車流經過",
}

const (
	soundStatusInitial = "選一段聲音，安靜陪著你"
	soundStatusOff     = "聲音已關閉"
	soundLoadFailed    = "這段聲音暫時無法播放，計時不受影響"
	// outputGain is the WebAudio master gain the ambience went through.
	outputGain = 0.55
)

// loopedSound is a recording with its seam crossfaded in place.
type loopedSound struct {
	sound              *audio.Sound
	loopStart, loopEnd time.Duration
	random             bool
}

// seamSamples crossfades the fade seconds after the loop into its first fade
// seconds, in place and at equal power, so playback runs from loopEnd into
// loopStart without a seam. It returns the loop's bounds in frames.
func seamSamples(samples []float32, channels, rate int, start, loop, fade float64) (int, int, error) {
	frames := len(samples) / channels
	a := int(math.Round(start * float64(rate)))
	f := int(math.Round(fade * float64(rate)))
	n := 0
	if loop > 0 {
		n = int(math.Round(loop * float64(rate)))
	} else {
		n = frames - a - f - int(math.Round(0.25*float64(rate)))
	}
	if n <= f || a+n+f > frames {
		return 0, 0, fmt.Errorf("迴圈超出錄音長度：%.2f 秒", float64(frames)/float64(rate))
	}
	for c := 0; c < channels; c++ {
		for i := 0; i < f; i++ {
			angle := math.Pi / 2 * float64(i) / float64(f)
			end := (a+n+i)*channels + c
			begin := (a+i)*channels + c
			samples[end] = float32(float64(samples[end])*math.Cos(angle) + float64(samples[begin])*math.Sin(angle))
		}
	}
	return a + f, a + n + f, nil
}

func framesToDuration(frames, rate int) time.Duration {
	return time.Duration(float64(frames) / float64(rate) * float64(time.Second))
}

// decodeLoop reads a whole recording into memory and seams it.
func decodeLoop(data []byte, hint seamHint) (*loopedSound, error) {
	d, _, err := audio.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	rate, channels := d.SampleRate(), d.Channels()
	var samples []float32
	buf := make([]float32, 8192*channels)
	for {
		n, err := d.Read(buf)
		samples = append(samples, buf[:n]...)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	start, fade := hint.start, hint.fade
	if start == 0 && fade == 0 {
		start, fade = 0.25, 3
	}
	ls, le, err := seamSamples(samples, channels, rate, start, hint.loop, fade)
	if err != nil {
		return nil, err
	}
	return &loopedSound{
		sound: audio.NewSound(rate, channels, samples), random: !hint.fixed,
		loopStart: framesToDuration(ls, rate), loopEnd: framesToDuration(le, rate),
	}, nil
}

// pinkNoise is Paul Kellet's filter from a fixed seed: 18 s, seamed like the recordings.
func pinkNoise() *loopedSound {
	const rate = 48000
	length := rate * 18
	samples := make([]float32, length*2)
	seed := uint32(211)
	random := func() float64 {
		seed = seed*1664525 + 1013904223
		return float64(seed) / 4294967296
	}
	for c := 0; c < 2; c++ {
		var b0, b1, b2, b3, b4, b5, b6 float64
		for i := 0; i < length; i++ {
			white := random()*2 - 1
			b0 = 0.99886*b0 + white*0.0555179
			b1 = 0.99332*b1 + white*0.0750759
			b2 = 0.96900*b2 + white*0.153852
			b3 = 0.8665*b3 + white*0.3104856
			b4 = 0.55*b4 + white*0.5329522
			b5 = -0.7616*b5 - white*0.016898
			samples[i*2+c] = float32((b0 + b1 + b2 + b3 + b4 + b5 + b6 + white*0.5362) * 0.052)
			b6 = white * 0.115926
		}
	}
	ls, le, _ := seamSamples(samples, 2, rate, 0, 17.65, 0.35)
	return &loopedSound{sound: audio.NewSound(rate, 2, samples), random: true,
		loopStart: framesToDuration(ls, rate), loopEnd: framesToDuration(le, rate)}
}

// chimeSound is the completion chime: three soft sine notes, 160 ms apart.
var chimeSound = sync.OnceValue(func() *audio.Sound {
	const rate = 48000
	total := int(rate * (0.32 + 0.65))
	samples := make([]float32, total*2)
	for i, freq := range []float64{523.25, 659.25, 783.99} {
		start := float64(i) * 0.16
		for f := 0; f < int(0.65*rate); f++ {
			t := float64(f) / rate
			var g float64
			switch {
			case t < 0.025:
				g = 0.07 * t / 0.025
			case t < 0.6:
				g = 0.07 * math.Pow(0.001/0.07, (t-0.025)/(0.6-0.025))
			default:
				g = 0.001
			}
			v := float32(g * math.Sin(2*math.Pi*freq*t))
			at := (int(start*rate) + f) * 2
			samples[at] += v
			samples[at+1] += v
		}
	}
	return audio.NewSound(rate, 2, samples)
})

type playingSound struct {
	player *audio.Player
	kind   string
	ramp   int // bumps to cancel an in-flight volume ramp
}

// soundscape plays one looping ambience at a time with the WebAudio fades.
type soundscape struct {
	mu        sync.Mutex
	files     fs.FS
	current   *playingSound
	loops     map[string]*loopedSound
	order     []string
	sequence  int
	requested bool
	started   bool
	lastError string
	sound     string
	volume    float64
	onStatus  func(message string, isError bool)
	onChanged func()
}

func newSoundscape(files fs.FS) *soundscape {
	return &soundscape{files: files, loops: map[string]*loopedSound{}, sound: "waves", volume: 0.35}
}

func (s *soundscape) status(message string, isError bool) {
	if s.onStatus != nil {
		s.onStatus(message, isError)
	}
}

func (s *soundscape) notify() {
	if s.onChanged != nil {
		s.onChanged()
	}
}

// load keeps the two most recent sounds decoded; a failed load is forgotten.
func (s *soundscape) load(kind string) (*loopedSound, error) {
	s.mu.Lock()
	if l := s.loops[kind]; l != nil {
		s.touch(kind)
		s.mu.Unlock()
		return l, nil
	}
	s.mu.Unlock()
	var l *loopedSound
	if kind == "pink" {
		l = pinkNoise()
	} else {
		data, err := fs.ReadFile(s.files, kind+".mp3")
		if err == nil {
			l, err = decodeLoop(data, seamHints[kind])
		}
		if err != nil {
			log.Printf("sound %q failed to load: %v", kind, err)
			return nil, errors.New(soundLoadFailed)
		}
	}
	s.mu.Lock()
	s.loops[kind] = l
	s.touch(kind)
	for len(s.order) > 2 {
		delete(s.loops, s.order[0])
		s.order = s.order[1:]
	}
	s.mu.Unlock()
	return l, nil
}

func (s *soundscape) touch(kind string) {
	for i, k := range s.order {
		if k == kind {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	s.order = append(s.order, kind)
}

// rampLocked moves a player's volume linearly to target over d, then calls done.
func (s *soundscape) rampLocked(p *playingSound, target float64, d time.Duration, done func()) {
	p.ramp++
	id, from := p.ramp, p.player.Volume()
	go func() {
		const step = 20 * time.Millisecond
		steps := max(1, int(d/step))
		for i := 1; i <= steps; i++ {
			time.Sleep(step)
			s.mu.Lock()
			if p.ramp != id {
				s.mu.Unlock()
				return
			}
			p.player.SetVolume(from + (target-from)*float64(i)/float64(steps))
			s.mu.Unlock()
		}
		if done != nil {
			done()
		}
	}()
}

func (s *soundscape) fadeOutLocked(p *playingSound) {
	if p == nil {
		return
	}
	s.started = false
	s.rampLocked(p, 0, 450*time.Millisecond, func() { time.Sleep(50 * time.Millisecond); p.player.Stop() })
}

// cancel aborts any in-flight load so a late decode cannot silently start.
func (s *soundscape) cancel() {
	s.mu.Lock()
	s.sequence++
	s.started = false
	s.fadeOutLocked(s.current)
	s.current = nil
	s.mu.Unlock()
	s.notify()
}

// request plays (or silences) a sound at a volume. It orders requests as
// they come and decodes a new sound on a goroutine of its own, so the latest
// request always wins and the caller never waits.
func (s *soundscape) request(playing bool, sound string, volume float64) {
	s.mu.Lock()
	s.sequence++
	seq := s.sequence
	s.requested, s.sound, s.volume = playing, sound, volume
	if !playing {
		s.fadeOutLocked(s.current)
		s.current, s.started = nil, false
		s.mu.Unlock()
		s.status(soundStatusOff, false)
		return
	}
	if s.current != nil && s.current.kind == sound {
		// setTargetAtTime(volume, now, 0.08): about 0.3 s to settle.
		s.rampLocked(s.current, volume*outputGain, 240*time.Millisecond, nil)
		s.mu.Unlock()
		s.status(soundStatusText[sound], false)
		return
	}
	s.mu.Unlock()
	go s.start(seq, sound, volume)
}

func (s *soundscape) start(seq int, sound string, volume float64) {
	loop, err := s.load(sound)
	s.mu.Lock()
	if seq != s.sequence || !s.requested {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.fadeOutLocked(s.current)
		s.current, s.started = nil, false
		s.lastError = err.Error()
		s.mu.Unlock()
		s.status(err.Error(), true)
		s.notify()
		return
	}
	player := loop.sound.NewPlayer().SetLoopRange(loop.loopStart, loop.loopEnd).SetVolume(0)
	at := loop.loopStart
	if loop.random {
		at += time.Duration(rand.Float64() * float64(loop.loopEnd-loop.loopStart))
	}
	player.Seek(at)
	player.Play()
	next := &playingSound{player: player, kind: sound}
	s.started = true
	s.rampLocked(next, volume*outputGain, 600*time.Millisecond, nil)
	s.fadeOutLocked(s.current)
	s.started = true
	s.current = next
	s.mu.Unlock()
	s.status(soundStatusText[sound], false)
	s.notify()
}

// playing reports a requested sound that has actually started.
func (s *soundscape) playing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requested && s.started
}

func (s *soundscape) chime() { chimeSound().Play() }

func (s *soundscape) dispose() {
	s.mu.Lock()
	s.sequence++
	s.requested = false
	if s.current != nil {
		s.current.ramp++
		s.current.player.Stop()
		s.current = nil
	}
	s.mu.Unlock()
}
