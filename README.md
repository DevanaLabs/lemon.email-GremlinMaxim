# GremlinMaxim

SMTP/IMAP clients for load testing

## maxim usage

Maxim is an smtp stress testing client. Start the program with no arguments for a short usage message:

```
Usage: maxim [options] -d <msgs_directory> <smtp_host:smtp_port>

Options:

  -c int
        number of envelope recepients for rcpt format (default 1)
  -d string
        directory with email messages
  -f string
        envelope sender (default "from@domain.com")
  -h string
        smtp helo/ehlo identification string (default "localhost")
  -m int
        number of messages for recipient per smtp session (default 1)
  -r string
        string format of envelope recepient (default "rcptto@domain.com")
  -s int
        number of smtp sessions to start (default 1)
  -v    print what's happening
```

Create `msgs_directory` and fill it with files of SMTP messages you would like to be sent. It will randomly pick one to send, so control the probabilities with number of message (types).

## gremlin usage

Maxim is an imap stress testing client. Start the program with no arguments for a short usage message:

```
Usage: gremlin [options] <imap_host:imap_port>

Options:

  -n int
        number of imap scripts per session (default 1)
  -p string
        string format of password
  -s int
        number of parallel sessions to start (default 1)
  -u string
        string format of username
  -v    print what's happening
```

Gremlin uses an implemented "imap script" which connects to an imap server, selects the "INBOX" folder and fetches the last 10 messages.
