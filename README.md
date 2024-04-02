# bot4imap

**Docker**

CGO_ENABLED=0 go build ./cmd/bot4imap

sudo docker build -t bot4imap .

sudo docker run --restart unless-stopped -d --name bot4imap --env-file ./config/env.list bot4imap

**Systemd**

adduser --no-create-home --no-user-group --system --shell /usr/sbin/nologin bot4imap

ln -s /opt/bot4imap/config/bot4imap.service /etc/systemd/system/

systemctl start bot4imap

systemctl enable bot4imap
