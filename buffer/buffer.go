package buffer

import (
	"bytes"
	"github.com/bwmarrin/discordgo"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
	ffmpeggo "github.com/u2takey/ffmpeg-go"
)

func createPionRTPPacket(p *discordgo.Packet) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			Version: 2,
			// Taken from Discord voice docs
			PayloadType:    0x78,
			SequenceNumber: p.Sequence,
			Timestamp:      p.Timestamp,
			SSRC:           p.SSRC,
		},
		Payload: p.Opus,
	}
}

type RtpBuffer struct {
	buf []*discordgo.Packet
	pos uint
}

func NewBuffer(size uint) RtpBuffer {
	return RtpBuffer{
		buf: make([]*discordgo.Packet, size),
		pos: 0,
	}
}

func (r *RtpBuffer) Push(p *discordgo.Packet) {
	r.buf[r.pos] = p
	if r.pos == uint(cap(r.buf)-1) {
		r.pos = 0
	} else {
		r.pos++
	}
}

func (r *RtpBuffer) DumpAudio() (*bytes.Buffer, error) {
	bIn := bytes.Buffer{}
	bOut := bytes.Buffer{}
	w, err := oggwriter.NewWith(&bIn, 48000, 2)
	for _, p := range r.buf[r.pos:] {
		if p != nil {
			err := w.WriteRTP(createPionRTPPacket(p))
			if err != nil {
				return nil, err
			}
		}
	}
	for _, p := range r.buf[:r.pos] {
		if p != nil {
			err := w.WriteRTP(createPionRTPPacket(p))
			if err != nil {
				return nil, err
			}
		}
	}
	err = ffmpeggo.
		Input("pipe:", ffmpeggo.KwArgs{"f": "ogg"}).
		Output("-", ffmpeggo.KwArgs{"f": "ogg", "af": "aresample=async=1"}).
		WithInput(&bIn).
		WithOutput(&bOut).
		OverWriteOutput().
		Run()
	if err != nil {
		return nil, err
	}
	return &bOut, nil
}
