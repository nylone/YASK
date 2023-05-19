package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/nylone/YASK/recorder"
	"time"
)

// Variables used for command line parameters
var (
	Token         string
	ChannelID     string
	TextChannelID string
	GuildID       string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&GuildID, "g", "", "Guild in which voice channel exists")
	flag.StringVar(&ChannelID, "c", "", "Voice channel to connect to")
	flag.StringVar(&TextChannelID, "o", "", "Text channel to dump to")
	flag.Parse()
}

func main() {
	var err error

	var s *discordgo.Session // bot session

	// connect to discord
	{
		s, err = discordgo.New("Bot " + Token)
		if err != nil {
			fmt.Println("error creating Discord session:", err)
			return
		}
		defer func(s *discordgo.Session) {
			err := s.Close()
			if err != nil {
				panic(err)
			}
		}(s)

		// We only really care about receiving voice state updates.
		s.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildVoiceStates)

		err = s.Open()
		if err != nil {
			fmt.Println("error opening connection:", err)
			return
		}
	}

	// join a voice channel
	v, err := s.ChannelVoiceJoin(GuildID, ChannelID, true, false)
	if err != nil {
		fmt.Println("failed to join voice channel:", err)
		return
	}

	// close voice channel on timeout
	go func() {
		time.Sleep(10 * time.Second)
		close(v.OpusRecv)
		v.Close()
	}()

	recorder.HandleVoice(GuildID, v.OpusRecv)
	f, err := recorder.DumpVoice(GuildID)
	if err != nil {
		return
	}
	ms := discordgo.MessageSend{
		Content: "here's your dump!",
		Files:   f,
	}
	_, err = s.ChannelMessageSendComplex(TextChannelID, &ms)
	if err != nil {
		return
	}
}
