package app

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	emailCheckTime = 60 * time.Second
)

//Types

type config struct {
	botToken     string
	chatID       int64
	imapServer   string
	imapLogin    string
	imapPassword string
	imapMbox     string
}

func getConfig() (c *config) {
	c = new(config)

	c.botToken = os.Getenv("BOT_TOKEN")
	if c.botToken == "" {
		log.Fatal("Environment variable not found: BOT_TOKEN")
	}

	intChatID, err := strconv.Atoi(os.Getenv("CHAT_ID"))
	if err != nil {
		log.Fatal("Environment variable CHAT_ID should be number")
	}
	c.chatID = int64(intChatID)

	c.imapServer = os.Getenv("IMAP_SERVER")
	if c.imapServer == "" {
		log.Fatal("Environment variable not found: IMAP_SERVER")
	}

	c.imapLogin = os.Getenv("IMAP_LOGIN")
	if c.imapLogin == "" {
		log.Fatal("Environment variable not found: IMAP_LOGIN")
	}

	c.imapPassword = os.Getenv("IMAP_PASSWORD")
	if c.imapPassword == "" {
		log.Fatal("Environment variable not found: IMAP_PASSWORD")
	}

	c.imapMbox = os.Getenv("IMAP_MBOX")
	if c.imapMbox == "" {
		c.imapMbox = "INBOX"
	}

	// log.Printf("%+v\n", c)

	return
}

type telegram struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

func getTelegram(conf *config) (tg *telegram) {
	tg = new(telegram)
	bot, err := tgbotapi.NewBotAPI(conf.botToken)
	if err != nil {
		log.Panic(err)
	}

	// bot.Debug = true

	tg.bot = bot
	tg.chatID = conf.chatID

	return
}

func (t *telegram) send(msg string) {
	chmsg := tgbotapi.NewMessage(t.chatID, msg)
	_, err := t.bot.Send(chmsg)

	if err != nil {
		log.Panic(err)
	}
}

// Public

func Start() {
	conf := getConfig()

	startImap(conf)
}

// Private

func startImap(conf *config) {

	tg := getTelegram(conf)

	// Connect to server
	c, err := client.DialTLS(conf.imapServer, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected IMAP")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(conf.imapLogin, conf.imapPassword); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in IMAP")

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	// Select INBOX
	mbox, err := c.Select(conf.imapMbox, false)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Mailbox \"%+v\" was selected", mbox.Name)

	for {
		getNewMessages(tg, c)
		waitUpdates(c)
	}
}

func getNewMessages(tg *telegram, c *client.Client) {
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}

	//log.Println("IDs found:", ids)

	if len(ids) > 0 {
		seqset := new(imap.SeqSet)
		seqset.AddNum(ids...)

		var section imap.BodySectionName
		section.Peek = true                            // Don't mark as read
		items := []imap.FetchItem{section.FetchItem()} // [BODY.PEEK[]]

		messages := make(chan *imap.Message, 10)
		done := make(chan error, 1)
		go func() {
			done <- c.Fetch(seqset, items, messages)
		}()

		for msg := range messages {

			text := messageToText(msg)

			tg.send(text)

			// mark as read
			seqSet := new(imap.SeqSet)
			seqSet.AddNum(msg.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.SeenFlag}
			err = c.Store(seqSet, item, flags, nil)
			if err != nil {
				log.Fatal(err)
			}
		}

		if err := <-done; err != nil {
			log.Fatal(err)
		}
	}
}

func messageToText(msg *imap.Message) (text string) {
	var section imap.BodySectionName
	r := msg.GetBody(&section)
	if r == nil {
		log.Fatal("Server didn't returned message body")
	}

	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		log.Fatal(err)
	}

	header := mr.Header
	date, err := header.Date()
	if err == nil {
		text += date.String() + "\n"
	}

	subject, err := header.Subject()
	if err == nil && subject != "" {
		text += subject + "\n"
	}

	// Process each message's part
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			b, _ := ioutil.ReadAll(p.Body)
			//log.Printf("Got text: %v\n", string(b))
			if string(b) != "" {
				text += string(b) + "\n"
			}
		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := h.Filename()
			log.Printf("Got attachment: %v\n", filename)
		}
	}
	return
}

func waitUpdates(c *client.Client) {
	updates := make(chan client.Update)
	c.Updates = updates

	// Start idling
	stopped := false
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- c.Idle(stop, nil)
	}()

	// Listen for updates
	for {
		select {
		case _ = <-updates:
			//log.Printf("New update: %T", update)
			if !stopped {
				close(stop)
				stopped = true
			}
		case err := <-done:
			if err != nil {
				log.Fatal(err)
			}
			//log.Println("Not idling anymore")
			return
		case <-time.After(emailCheckTime):
			//log.Println("Timer")
			if !stopped {
				close(stop)
				stopped = true
			}
		}
	}
}
