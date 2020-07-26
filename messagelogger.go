package main

import (
	"fmt"
	"log"
	"strings"
	"trup/db"

	"github.com/bwmarrin/discordgo"
)

type messageCache struct {
	queue []string
	m     map[string]discordgo.Message
}

func newMessageCache(size int) *messageCache {
	return &messageCache{make([]string, size), make(map[string]discordgo.Message, size)}
}

func (c *messageCache) add(k string, value discordgo.Message) {
	delete(c.m, c.queue[0])
	c.m[k] = value
	c.queue[0] = ""
	c.queue = c.queue[1:]
	c.queue = append(c.queue, k)
}

func messageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in messageDelete", r)
		}
	}()

	const dateFormat = "2006-01-02T15:04:05.0000Z"
	messageCreationDate, _ := discordgo.SnowflakeTimestamp(m.ID)
	message, inCache := cache.m[m.ID]
	if !inCache {
		log.Printf("Unknown user deleted message %s(not in cache), message creation date: %s\n", m.ID, messageCreationDate.UTC().Format(dateFormat))
		return
	}

	var footer *discordgo.MessageEmbedFooter
	if channel, err := s.State.Channel(m.ChannelID); err == nil {
		footer = &discordgo.MessageEmbedFooter{
			Text: "#" + channel.Name,
		}
	}

	messageEmbed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Message Delete",
			IconURL: message.Author.AvatarURL("128"),
		},
		Title:       fmt.Sprintf("%s#%s(%s)", message.Author.Username, message.Author.Discriminator, message.Author.ID),
		Description: message.Content,
		Timestamp:   messageCreationDate.UTC().Format(dateFormat),
		Footer:      footer,
	}

	mediaFiles, finish, err := db.GetStoredAttachments(m.ChannelID, m.Message.ID)
	defer finish()
	if err != nil || len(mediaFiles) == 0 {
		s.ChannelMessageSendEmbed(env.ChannelBotlog, messageEmbed)
		return
	}

	discordFiles := []*discordgo.File{}
	for _, file := range mediaFiles {
		discordFiles = append(discordFiles, &discordgo.File{
			Name:        file.Filename,
			Reader:      file.Reader,
			ContentType: file.GetContentType(),
		})
	}

	if strings.Split(discordFiles[0].ContentType, "/")[0] == "video" {
		messageEmbed.Video = &discordgo.MessageEmbedVideo{URL: "attachment://" + mediaFiles[0].Filename}
	} else if strings.Split(discordFiles[0].ContentType, "/")[0] == "image" {
		messageEmbed.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + mediaFiles[0].Filename}
	}

	_, err = s.ChannelMessageSendComplex(env.ChannelBotlog, &discordgo.MessageSend{
		Embed: messageEmbed,
		Files: discordFiles,
	})
	if err != nil {
		log.Printf("Error writing message with attachments to botlog: %s", err)
	}
}

func messageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in messageUpdate", r)
		}
	}()

	const dateFormat = "2006-01-02T15:04:05.0000Z"
	messageCreationDate, _ := discordgo.SnowflakeTimestamp(m.ID)

	cached, inCache := cache.m[m.ID]
	if !inCache {
		return
	}
	before := cached.Content
	cached.Content = m.Content
	cache.m[m.ID] = cached

	var footer *discordgo.MessageEmbedFooter
	if channel, err := s.State.Channel(m.ChannelID); err == nil {
		footer = &discordgo.MessageEmbedFooter{
			Text: "#" + channel.Name,
		}
	}

	s.ChannelMessageSendEmbed(env.ChannelBotlog, &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Message Edit",
			IconURL: m.Author.AvatarURL("128"),
		},
		Title:       fmt.Sprintf("%s#%s(%s)", m.Author.Username, m.Author.Discriminator, m.Author.ID),
		Description: "**Before:**\n" + before + "\n\n**Now:**\n" + m.Content,
		Timestamp:   messageCreationDate.UTC().Format(dateFormat),
		Footer:      footer,
	})
}
