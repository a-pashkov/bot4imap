# bot4imap

**Docker**

CGO_ENABLED=0 go build ./cmd/bot4imap

sudo docker build -t bot4imap .

sudo docker run --restart unless-stopped -d --name bot4imap --env-file ./config/env.list bot4imap
