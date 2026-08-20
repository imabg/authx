package mail

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFormatSMTPMessage(t *testing.T) {
	got := formatSMTPMessage(Config{FromEmail: "from@example.com", FromName: "Acme"}, Message{
		To:      "to@example.com",
		Subject: "Hello",
		Text:    "Line 1\nLine 2\n",
	})
	for _, want := range []string{
		"From: Acme <from@example.com>\r\n",
		"To: to@example.com\r\n",
		"Subject: Hello\r\n",
		"Line 1\r\nLine 2\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestSMTPSenderSend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var got []string
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		br := bufio.NewReader(conn)
		writeSMTP(conn, "220 test ESMTP")
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.TrimSpace(line)
			mu.Lock()
			got = append(got, cmd)
			mu.Unlock()
			upper := strings.ToUpper(cmd)
			switch {
			case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
				writeSMTP(conn, "250-localhost")
				writeSMTP(conn, "250 OK")
			case strings.HasPrefix(upper, "MAIL"):
				writeSMTP(conn, "250 OK")
			case strings.HasPrefix(upper, "RCPT"):
				writeSMTP(conn, "250 OK")
			case strings.HasPrefix(upper, "DATA"):
				writeSMTP(conn, "354 End data with <CR><LF>.<CR><LF>")
				for {
					dataLine, err := br.ReadString('\n')
					if err != nil {
						return
					}
					mu.Lock()
					got = append(got, strings.TrimRight(dataLine, "\r\n"))
					mu.Unlock()
					if strings.TrimSpace(dataLine) == "." {
						break
					}
				}
				writeSMTP(conn, "250 OK")
			case strings.HasPrefix(upper, "QUIT"):
				writeSMTP(conn, "221 Bye")
				return
			default:
				writeSMTP(conn, "250 OK")
			}
		}
	}()

	sender := NewSMTPSender()
	err = sender.Send(context.Background(), Config{
		FromEmail: "noreply@example.com",
		FromName:  "Acme",
		SMTP: SMTPConfig{
			Host: "127.0.0.1",
			Port: ln.Addr().(*net.TCPAddr).Port,
		},
	}, Message{To: "user@example.com", Subject: "OTP", Text: "code 123456"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	<-done
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "MAIL FROM:<noreply@example.com>") {
		t.Fatalf("missing MAIL FROM in %s", joined)
	}
	if !strings.Contains(joined, "RCPT TO:<user@example.com>") {
		t.Fatalf("missing RCPT TO in %s", joined)
	}
	if !strings.Contains(joined, "Subject: OTP") {
		t.Fatalf("missing subject in %s", joined)
	}
	if strings.Contains(joined, "smtp-password") {
		t.Fatal("conversation leaked a password")
	}
}

func TestSMTPSenderRequiresHost(t *testing.T) {
	err := NewSMTPSender().Send(context.Background(), Config{FromEmail: "a@b.com"}, Message{To: "c@d.com"})
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("error = %v", err)
	}
}

func writeSMTP(w io.Writer, line string) {
	_, _ = io.WriteString(w, line+"\r\n")
}
