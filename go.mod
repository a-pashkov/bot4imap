module github.com/a-pashkov/bot4imap

go 1.18

require internal/app v0.0.0

require (
	github.com/emersion/go-imap v1.2.1 // indirect
	github.com/emersion/go-message v0.16.0 // indirect
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1 // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace internal/app => ./internal/app
