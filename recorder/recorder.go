package recorder

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nylone/YASK/buffer"
	"strconv"
	"sync"
)

type ssrc = uint32

type gidMap struct {
	m  map[string]*buffersMap
	rw sync.RWMutex
}

type buffersMap struct {
	m   map[ssrc]*buffer.RtpBuffer
	mut sync.Mutex
}

func (r *gidMap) getBuffersMap(gid string) *buffersMap {
	// get read access and see if the gid is already mapped
	r.rw.RLock()
	m, ok := r.m[gid]
	if !ok {
		// add a new buffer to the map since gid is not mapped
		r.rw.RUnlock() // release read lock
		r.rw.Lock()    // get write lock
		m = new(buffersMap)
		m.m = map[ssrc]*buffer.RtpBuffer{}
		r.m[gid] = m
		r.rw.Unlock() // release write lock
		return m
	}
	defer r.rw.RUnlock()
	return m
}

var (
	g2bm = gidMap{
		m: map[string]*buffersMap{},
	} // guild to ssrc to audio buffers
)

func addPacket(gid string, ssrc ssrc, p *discordgo.Packet) {
	// get ssrc to audio buffer
	bm := g2bm.getBuffersMap(gid)

	// lock the ssrc to audio buffer
	bm.mut.Lock()
	defer bm.mut.Unlock()

	// get relevant audio buffer
	b, ok := bm.m[ssrc]
	if !ok {
		nb := buffer.NewBuffer(1024)
		b = &nb
		bm.m[ssrc] = b
	}

	b.Push(p)
}

func HandleVoice(gid string, c chan *discordgo.Packet) {
	for p := range c {
		addPacket(gid, p.SSRC, p)
	}
}

func DumpVoice(gid string) ([]*discordgo.File, error) {
	// get ssrc to audio buffer
	bm := g2bm.getBuffersMap(gid)
	// lock the ssrc to audio buffer
	bm.mut.Lock()
	defer bm.mut.Unlock()
	var files []*discordgo.File
	for ssrc, rtpBuffer := range bm.m {
		out, err := rtpBuffer.DumpAudio()
		if err != nil {
			return nil, err
		}
		f := discordgo.File{
			Name:        strconv.Itoa(int(ssrc)) + ".ogg",
			ContentType: "audio/ogg",
			Reader:      out,
		}
		files = append(files, &f)
	}
	return files, nil
}
