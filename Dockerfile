FROM alpine:latest

ADD ./bot4imap /opt/

CMD ["/opt/bot4imap"]
