package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/hajimehoshi/oto/v2"
	"github.com/go-audio/wav"
)

const (
	threshold      = 3000
	sampleRate     = 44100
	channels       = 1
	bufferSize     = 1024
	triggerTimeout = 2 * time.Second
)

var (
	context *oto.Context
	once    sync.Once
)

func main() {
	once.Do(func() {
		var ready chan struct{}
		var err error
		context, ready, err = oto.NewContext(sampleRate, channels, 2)
		if err != nil {
			log.Fatalf("Failed to initialize oto context: %v", err)
		}
		<-ready
	})
	portaudio.Initialize()
	defer portaudio.Terminate()

	input := make([]int16, bufferSize)
	stream, err := portaudio.OpenDefaultStream(channels, 0, sampleRate, len(input), func(in []int16) {
		copy(input, in)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}
	defer stream.Stop()

	var lastTriggerTime time.Time

	log.Println("マイク入力を監視中...")

	for {
		volume := calculateVolume(input)
		if volume > threshold && time.Since(lastTriggerTime) > triggerTimeout {
			log.Printf("音量検出: %d (閾値: %d)", volume, threshold)
			playAudio("maou_se_system46.wav")
			lastTriggerTime = time.Now()
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func calculateVolume(data []int16) float64 {
	var sum int64
	for _, v := range data {
		sum += int64(v) * int64(v)
	}
	mean := float64(sum) / float64(len(data))
	return math.Sqrt(mean)
}

func playAudio(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		log.Fatal("Invalid WAV file")
	}

	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatal(err)
	}

	data := convertIntToBytes(buffer.Data)
	reader := bytes.NewReader(data)

	player := context.NewPlayer(reader)
	if player == nil {
		log.Fatal("Failed to create player")
	}
	defer player.Close()

	player.Play()
	for player.IsPlaying() {
	}
}

func convertIntToBytes(data []int) []byte {
	buf := new(bytes.Buffer)
	for _, v := range data {
		err := binary.Write(buf, binary.LittleEndian, int16(v))
		if err != nil {
			log.Fatal("Failed to convert PCM data:", err)
		}
	}
	return buf.Bytes()
}
