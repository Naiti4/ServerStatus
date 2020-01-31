package serverstatus

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/anvie/port-scanner"
	"github.com/bwmarrin/discordgo"
	steam "github.com/kidoman/go-steam"
	"github.com/mgerb/ServerStatus/bot"
	"github.com/mgerb/ServerStatus/config"
)

const (
	red   = 0xf4425c
	green = 0x42f477
	blue  = 0x42adf4
)

// Start - start port scanner and bot listeners
func Start() {
	//set each server status as online to start
	for i := range config.Config.Servers {
		config.Config.Servers[i].Online = true
	}

	err := bot.Session.UpdateStatus(0, config.Config.GameStatus)

	sendMessageToRooms(blue, "Server Status", "Bot started! Type !ServerStatus to see the status of your servers :smiley:", false)

	if err != nil {
		log.Println(err)
	}

	//start a new go routine
	go scanServers()
}

func scanServers() {

	//check if server are in config file
	if len(config.Config.Servers) < 1 {
		log.Println("No servers in config file.")
		return
	}

	for {

		for index, server := range config.Config.Servers {
			prevServerUp := server.Online //set value to previous server status

			serverScanner := portscanner.NewPortScanner(server.Address, time.Second*2, 1)
			serverUp := serverScanner.IsOpen(server.Port) //check if the port is open

			// if server isn't up check RCON protocol (UDP)
			if !serverUp {
				host := server.Address + ":" + strconv.Itoa(server.Port)
				steamConnection, err := steam.Connect(host)
				if err == nil {
					defer steamConnection.Close()
					_, err := steamConnection.Ping()
					if err == nil {
						serverUp = true
					}
				}
			}

			if serverUp && serverUp != prevServerUp {
				sendMessageToRooms(green, server.Name, "(SERVER) Estado del servidor: Running", true)
			} else if !serverUp && serverUp != prevServerUp {
				sendMessageToRooms(red, server.Name, "(SERVER) Estado del servidor: Stopped", true)
			}

			config.Config.Servers[index].Online = serverUp
		}

		time.Sleep(time.Second * config.Config.PollingInterval)
	}
}

func sendMessageToRooms(color int, title, description string, mentionRoles bool) {
	for _, roomID := range config.Config.RoomIDList {
		if mentionRoles {
			content := strings.Join(config.Config.RolesToNotify, " ")
			bot.Session.ChannelMessageSend(roomID, content)
		}
		sendEmbededMessage(roomID, color, title, description)
	}
}

func sendEmbededMessage(roomID string, color int, title, description string) {

	embed := &discordgo.MessageEmbed{
		Color:       color,
		Title:       title,
		Description: description,
	}

	bot.Session.ChannelMessageSendEmbed(roomID, embed)
}

// MessageHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == bot.BotID {
		return
	}

	if m.Content == "!ServerStatus" {
		for _, server := range config.Config.Servers {
			if server.Online {
				sendEmbededMessage(m.ChannelID, green, server.Name, "Online!")
			} else {
				sendEmbededMessage(m.ChannelID, red, server.Name, "Offline!")
			}
		}
	}
}
