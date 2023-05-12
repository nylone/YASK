package recorder

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nylone/YASK/buffer"
	"os"
	"strconv"
	"sync"
)

type ssrc = uint32

type gidBuffersMap struct {
	m  map[string]*ssrcBuffersMap
	rw sync.RWMutex
}

type ssrcBuffersMap struct {
	m   map[ssrc]*buffer.RtpBuffer
	mut sync.Mutex
}

func (r *gidBuffersMap) getSSRCBuffersMap(gid string) *ssrcBuffersMap {
	// get read access and see if the gid is already mapped
	r.rw.RLock()
	m, ok := r.m[gid]
	if !ok {
		// add a new buffer to the map since gid is not mapped
		r.rw.RUnlock() // release read lock
		r.rw.Lock()    // get write lock
		m = new(ssrcBuffersMap)
		m.m = map[ssrc]*buffer.RtpBuffer{}
		r.m[gid] = m
		r.rw.Unlock() // release write lock
		return m
	}
	defer r.rw.RUnlock()
	return m
}

var (
	g2bm = gidBuffersMap{
		m: map[string]*ssrcBuffersMap{},
	} // guild to ssrc to audio buffers
)

func addPacket(gid string, ssrc ssrc, p *discordgo.Packet) {
	// get ssrc to audio buffer
	m := g2bm.getSSRCBuffersMap(gid)

	// lock the ssrc to audio buffer
	m.mut.Lock()
	defer m.mut.Unlock()

	// get relevant audio buffer
	b, ok := m.m[ssrc]
	if !ok {
		nb := buffer.NewBuffer()
		b = &nb
		m.m[ssrc] = b
	}

	b.Push(p)
}

func HandleVoice(gid string, c chan *discordgo.Packet) {
	for p := range c {
		addPacket(gid, p.SSRC, p)
	}
}

func DumpVoice(gid string) error {
	for ssrc, rtpBuffer := range g2bm.getSSRCBuffersMap(gid).m {
		out, err := rtpBuffer.DumpAudio()
		if err != nil {
			return err
		}
		// open output file
		fo, err := os.Create(strconv.Itoa(int(ssrc)) + ".ogg")
		if err != nil {
			return err
		}
		// close fo on exit and check for its returned error
		_, err = fo.Write(out.Bytes())
		if err != nil {
			return err
		}
		if err := fo.Close(); err != nil {
			return err
		}
	}
	return nil
}
