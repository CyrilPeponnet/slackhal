package pluginarchiver

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

const (
	chanActive   = "ACTIVE"
	chanArchived = "ARCHIVED"
)

// ChatBot is the db model of a bot
type ChatBot struct {
	ID          int    `gorm:"not null;AUTO_INCREMENT;primary_key"`
	Active      bool   `gorm:"not null;column:is_active"`
	Server      string `gorm:"not null"`
	ServerPW    string `gorm:"column:server_password"`
	ServerID    string `gorm:"not null;column:server_identifier"`
	Nick        string `gorm:"not null"`
	Password    string
	RealName    string
	Slug        string `gorm:"not null"`
	MaxChannels int    `gorm:"not null;column:max_channels"`
}

// TableName set the table name containing chatbots
func (ChatBot) TableName() string { return "bots_chatbot" }

// GetChatBotFromDB will return the ChatBot or created it if needed
func GetChatBotFromDB(db *gorm.DB, name string) *ChatBot {
	bot := ChatBot{}
	rr := db.Where(&ChatBot{Nick: name}).First(&bot)
	if rr.Error != nil && rr.Error.Error() == "record not found" {
		// Create a ChatBot
		bot.Active = true
		bot.Nick = name
		bot.Server = "no server"
		bot.MaxChannels = 1000
		bot.ServerID = "no server id"
		bot.Slug = name
		db.NewRecord(bot)
		db.Create(&bot)
	}
	return &bot
}

// Channel is the db model of channel
type Channel struct {
	ID      int       `gorm:"not null;AUTO_INCREMENT;primary_key"`
	Created time.Time `gorm:"not null"`
	Updated time.Time `gorm:"not null"`
	// This will be the diplay name of the slack channel
	Name string `gorm:"not null"`
	// This will be the coded name of the slack channel
	Slug        string `gorm:"not null"`
	PrivateSlug string `gorm:"column:private_slug"`
	// Password    string
	Public   bool `gorm:"not null;column:is_public"`
	Featured bool `gorm:"not null;column:is_featured"`
	// Fingerprint string
	Kudos   bool   `gorm:"not null;column:public_kudos"`
	Notes   string `gorm:"not null"`
	ChatBot int    `gorm:"not null;column:chatbot_id"`
	// Status set to ACTIVE or ARCHIVED
	Status string `gorm:"not null"`
}

// GetChannelFromDB get or create a new chan
func GetChannelFromDB(db *gorm.DB, id int, name string, slug string, public bool) *Channel {
	c := Channel{}
	fmt.Printf("%v/%v", name, slug)
	rr := db.Where(&Channel{Name: name}).First(&c)
	if rr.Error != nil && rr.Error.Error() == "record not found" {
		// Create new chan
		c.ChatBot = id
		c.Created = time.Now()
		c.Featured = false
		c.Public = public
		c.Kudos = false
		c.Name = name
		c.Slug = slug
		c.PrivateSlug = slug
		c.Status = chanActive
		db.Save(&c)
	}
	return &c
}

// TableName set the table name containing chatbots
func (Channel) TableName() string { return "bots_channel" }

// Log is the db model of logs
type Log struct {
	ID        int       `gorm:"not null;AUTO_INCREMENT;primary_key"`
	Timestamp time.Time `gorm:"not null"`
	Nick      string    `gorm:"not null"`
	Text      string    `gorm:"not null"`
	Action    bool      `gorm:"not null"`
	// Command must be PRIVMSG for log to appear.
	Command string `gorm:"not null"`
	// Host      string
	// Raw       string
	// Room    string
	Bot     int `gorm:"column:bot_id"`
	Channel int `gorm:"column:channel_id"`
}

// NewLogToDB create a new log for a given bot, channel
func NewLogToDB(db *gorm.DB, botID int, chanID int, user string, message string) {
	// Create sone logs
	log := Log{}
	log.Action = false
	log.Bot = botID
	log.Channel = chanID
	log.Command = "PRIVMSG"
	log.Nick = user
	log.Text = message
	log.Timestamp = time.Now()
	db.Create(&log)

	// Create the search_index field using direct exec command as tsvector is not a known type
	db.Exec("update logs_log set search_index = to_tsvector(text) where id = (?)", log.ID)

}

// TableName set the table name containing chatbots
func (Log) TableName() string { return "logs_log" }
