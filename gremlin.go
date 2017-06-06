package main

import (
    "bytes"
    "fmt"
    "net/mail"
    "time"
    "crypto/tls"
    "github.com/mxk/go-imap/imap"
    "sync"
    "os"
    "flag"
    "strconv"
    "strings"
    "github.com/DevanaLabs/lemon.email-GremlinMaxim/avg"
)

type Options struct {
    Sessions  int
    Scripts   int
    Usernames string
    Passwords string
    Verbose   bool
}

var options *Options

type SessionCtx struct {
    id     string
    addr   string
    wg     *sync.WaitGroup
}

type ScriptCtx struct {
    addr   string
    username string
    password string
    verbose bool
}

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage: %s [options] <imap_host:imap_port>\n", os.Args[0])
    fmt.Fprintln(os.Stderr, "\nOptions:\n")
    flag.PrintDefaults()
}

var bytesRcvdPerSec = avg.NewAvg()

func init() {
    var (
        sessions   = flag.Int("s", 1, "number of parallel sessions to start")
        scripts    = flag.Int("n", 1, "number of imap scripts per session")
        usernames  = flag.String("u", "", "string format of username")
        passwords  = flag.String("p", "", "string format of password")
        verbose    = flag.Bool("v", false, "print what's happening")
    )

    flag.Parse()

    options = &Options{
        Sessions:   *sessions,
        Scripts:    *scripts,
        Usernames:  *usernames,
        Passwords:  *passwords,
        Verbose:    *verbose,
    }
}

func clientScript(ctx *ScriptCtx) {
    var (
        c   *imap.Client
        cmd *imap.Command
        rsp *imap.Response
        err error
    )

    cfg := &tls.Config{
        InsecureSkipVerify: true,
    }

    // Connect to the server
    c, err = imap.DialTLS(ctx.addr, cfg)
    if err != nil {
        fmt.Println("imap.DialTLS", err)
        return
    }

    // Remember to log out and close the connection when finished
    defer func() {
        c.Logout(30 * time.Second)
    }()

    // Print server greeting (first response in the unilateral server data queue)
    if ctx.verbose {
        fmt.Println("Server says hello:", c.Data[0].Info)
    }
    c.Data = nil

    // Authenticate
    if c.State() == imap.Login {
        cmd, err = c.Login(ctx.username, ctx.password)
        if err != nil {
            fmt.Println("c.Login", err)
            return
        }
    }

    // Open a mailbox (synchronous command - no need for imap.Wait)
    c.Select("INBOX", true)
    //fmt.Print("\nMailbox status:\n", c.Mailbox)

    // Fetch the headers of the 10 most recent messages
    set, _ := imap.NewSeqSet("")
    if c.Mailbox.Messages >= 10 {
        set.AddRange(c.Mailbox.Messages-9, c.Mailbox.Messages)
    } else {
        set.Add("1:*")
    }
    cmd, _ = c.Fetch(set, "RFC822.HEADER", "RFC822.TEXT")
    if err != nil {
        fmt.Println("c.Fetch", err)
        return
    }

    startTime := time.Now()
    totalSize := 0

    // Process responses while the command is running
    //fmt.Println("\nMost recent messages:")
    for cmd.InProgress() {
        // Wait for the next response (no timeout)
        c.Recv(-1)

        // Process command data
        for _, rsp = range cmd.Data {
            header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
            body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.TEXT"])
            if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
                //fmt.Println("|--", msg.Header.Get("Subject"))
            }

            size := len(body)
            
            if ctx.verbose {
                fmt.Printf("Msg size: %d\n", size)
            }

            totalSize += size
        }
        cmd.Data = nil

        // Process unilateral server data
        for _, rsp = range c.Data {
            //fmt.Println("Server data:", rsp)
        }
        c.Data = nil
    }

    bytesRcvdPerSec.AddValuePerTime(float64(totalSize), startTime)

    //fmt.Printf("TotalSize %d, Elapsed %f, average %f\n", totalSize, elapsed, average)

    // Check command completion status
    if rsp, err := cmd.Result(imap.OK); err != nil {
        if err == imap.ErrAborted {
            fmt.Println("Fetch command aborted")
        } else {
            fmt.Println("Fetch error:", rsp.Info)
        }
    }
}

func optSprintf(format string, i int) string {
    if strings.Contains(format, "%") {
        return fmt.Sprintf(format, i)
    }

    return format
}

func session(ctx *SessionCtx, opt *Options) {
    defer ctx.wg.Done()

    var username string
    var password string

    for i := 0; i < opt.Scripts; i++ {
        username = optSprintf(opt.Usernames, i)
        password = optSprintf(opt.Passwords, i)

        clientScript(&ScriptCtx{
            addr: ctx.addr,
            username: username,
            password: password,
            verbose: opt.Verbose,
        })
    }
}

func main() {
    addr := flag.Arg(0)

    if addr == "" || options.Usernames == "" || options.Passwords == "" {
        Usage()
        return
    }

    var wg sync.WaitGroup

    for i := 0; i < options.Sessions; i++ {
        wg.Add(1)
        go session(&SessionCtx{
            id:     strconv.Itoa(i),
            addr:   addr,
            wg:     &wg,
        }, options)
    }

    wg.Wait()

    fmt.Printf("Average msg dl speed: %s B/s\n", 
        strconv.FormatFloat(bytesRcvdPerSec.GetValue() * float64(options.Sessions), 'f', 2, 64))
}
