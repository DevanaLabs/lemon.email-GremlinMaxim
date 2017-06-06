package main

import (
    "bytes"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "math/rand"
    "net/smtp"
    "os"
    "path/filepath"
    "strconv"
    "sync"
    "time"
    "github.com/DevanaLabs/lemon.email-GremlinMaxim/avg"
)

type Options struct {
    Sessions   int
    Messages   int
    From       string
    RcptCount  int
    RcptFormat string
    EmailDir   string
    Verbose    bool
    Helo      string
}

type EmailMap map[int]string

type BodyMap map[string][]byte

type SessionCtx struct {
    id     string
    addr   string
    wg     *sync.WaitGroup
    emails EmailMap
    bodies BodyMap
}

var options *Options

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage: %s [options] -d <msgs_directory> <smtp_host:smtp_port>\n", os.Args[0])
    fmt.Fprintln(os.Stderr, "\nOptions:\n")
    flag.PrintDefaults()
}

var bytesSentPerSec = avg.NewAvg()

func init() {
    var (
        helo       = flag.String("h", "localhost", "smtp helo/ehlo identification string")
        sessions   = flag.Int("s", 1, "number of smtp sessions to start")
        messages   = flag.Int("m", 1, "number of messags for recipient per smtp session")
        from       = flag.String("f", "from@domain.com", "envelope sender")
        rcptFormat = flag.String("r", "rcptto@domain.com", "string format of envelope recepient")
        rcptCount  = flag.Int("c", 1, "number of envelope recepients for rcpt format")
        emailDir   = flag.String("d", "", "directory with email messages")
        verbose    = flag.Bool("v", false, "print what's happening")
    )

    flag.Parse()

    options = &Options{
        Helo:       *helo,
        Sessions:   *sessions,
        Messages:   *messages,
        From:       *from,
        RcptFormat: *rcptFormat,
        RcptCount:  *rcptCount,
        EmailDir:   *emailDir,
        Verbose:    *verbose,
    }
}

func SendMail(c *smtp.Client, from string, rcptto string, body io.Reader) error {
    startTime := time.Now()

    c.Mail(from)
    c.Rcpt(rcptto)

    wc, err := c.Data()
    if err != nil {
        return err
    }
    defer wc.Close()

    size, err := io.Copy(wc, body)
    if err != nil {
        return err
    }

    bytesSentPerSec.AddValuePerTime(float64(size), startTime)

    return nil
}

// Allocates cache map for email sources (reducing io load on test machine)
func cacheBodies(ctx *SessionCtx) error {
    ctx.bodies = make(map[string][]byte)
    var err error

    for _, e := range ctx.emails {
        ctx.bodies[e], err = ioutil.ReadFile(e)
        if err != nil {
            return err
        }
    }

    return nil
}

// Single SMTP session goroutine
func Session(ctx *SessionCtx, opt *Options) {
    defer ctx.wg.Done()

    c, err := smtp.Dial(ctx.addr)
    if err != nil {
        log.Println(err)
        return
    }
    defer c.Close()

    num := len(ctx.emails)

    err = cacheBodies(ctx)
    if err != nil {
        log.Println(err)
        return
    }

    c.Hello(opt.Helo)

    var path string
    var recepient string

    for i := 0; i < opt.Messages; i++ {
        for j := 0; j < opt.RcptCount; j++ {
            path = ctx.emails[rand.Intn(num)]
            recepient = fmt.Sprintf(opt.RcptFormat, j)

            if opt.Verbose {
                fmt.Printf("[%s] Sending message %d [%s] for %s\n", ctx.id, i, path, recepient)
            }

            if err := SendMail(c, opt.From, recepient, bytes.NewReader(ctx.bodies[path])); err != nil {
                log.Println(err)
            }
        }
    }
}

// Create map for easy random-picking emails to send
func mapEmails(dir string) (EmailMap, error) {
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        return nil, err
    }

    msgs := make(map[int]string)

    for i, f := range files {
        msgs[i] = filepath.Join(dir, f.Name())
    }

    return msgs, nil
}

func main() {
    // reduce mem usage or eliminate contention, the question is now..

    addr := flag.Arg(0)

    if addr == "" || options.EmailDir == "" {
        Usage()
        return
    }

    emails, err := mapEmails(options.EmailDir)
    if err != nil {
        log.Println(err)
        return
    }

    var wg sync.WaitGroup

    rand.Seed(time.Now().Unix())

    for i := 0; i < options.Sessions; i++ {
        wg.Add(1)
        go Session(&SessionCtx{
            id:     strconv.Itoa(i),
            addr:   addr,
            wg:     &wg,
            emails: emails,
        }, options)
    }

    wg.Wait()

    fmt.Printf("Average DATA cmd speed: %s B/s\n", 
        strconv.FormatFloat(bytesSentPerSec.GetValue() * float64(options.Sessions), 'f', 2, 64))
}
