package gmail

import (
	"encoding/base64"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pranavmangal/grabotp/otp"

	"golang.org/x/net/html"
	"google.golang.org/api/gmail/v1"
)

const (
	fetchWindowMin = 10
	maxResults     = 3
)

type ParsedOTP struct {
	To        string    `json:"to"`
	From      string    `json:"from"`
	Timestamp time.Time `json:"timestamp"`
	OTP       string    `json:"otp"`
}

func FetchOTPs(user string) []ParsedOTP {
	srv := GetGmailService(user)

	timeAgo := time.Now().Add(-time.Minute * fetchWindowMin).Unix()
	query := "after:" + strconv.FormatInt(timeAgo, 10)

	r, err := srv.Users.Messages.List(user).Q(query).MaxResults(maxResults).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve messages: %v", err)
	}

	if len(r.Messages) == 0 {
		return []ParsedOTP{}
	}

	var wg sync.WaitGroup
	otpsChan := make(chan ParsedOTP, len(r.Messages))

	for _, m := range r.Messages {
		wg.Add(1)

		go func(msgId string) {
			defer wg.Done()

			msg, err := srv.Users.Messages.Get(user, m.Id).Format("full").Do()
			if err != nil {
				log.Printf("Unable to retrieve message %v: %v", msgId, err)
				return
			}

			sender := getSender(msg)
			timestamp := getTimestamp(msg)
			body := getBody(msg)

			otp := otp.Extract(body)
			if len(otp) > 0 {
				otpsChan <- ParsedOTP{To: user, From: sender, Timestamp: timestamp, OTP: otp}
			}
		}(m.Id)
	}

	wg.Wait()
	close(otpsChan)

	res := []ParsedOTP{}
	for otp := range otpsChan {
		res = append(res, otp)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Timestamp.After(res[j].Timestamp)
	})

	return res
}

func getSender(msg *gmail.Message) string {
	for _, h := range msg.Payload.Headers {
		if h.Name == "From" {
			return h.Value
		}
	}

	return ""
}

func getTimestamp(msg *gmail.Message) time.Time {
	return time.UnixMilli(msg.InternalDate)
}

func getBody(msg *gmail.Message) string {
	var parts []*gmail.MessagePart
	if msg.Payload.Body.Data != "" {
		// Single-part email
		parts = append(parts, &gmail.MessagePart{
			MimeType: msg.Payload.MimeType,
			Body:     msg.Payload.Body,
		})
	} else if len(msg.Payload.Parts) > 0 {
		// Multipart email
		parts = msg.Payload.Parts
	}

	for _, part := range parts {
		if part.Body == nil || part.Body.Data == "" || isAttachment(part) {
			continue
		}

		switch part.MimeType {
		case "text/plain":
			if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
				return string(decoded)
			}
		case "text/html":
			if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
				return parseHTMLText(string(decoded))
			}
		}
	}

	return ""
}

func isAttachment(part *gmail.MessagePart) bool {
	for _, header := range part.Headers {
		if header.Name == "Content-Disposition" {
			return strings.Contains(header.Value, "attachment")
		}
	}

	return false
}

func parseHTMLText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	var text strings.Builder
	walkHTMLNodes(doc, &text)

	return strings.TrimSpace(text.String())
}

func walkHTMLNodes(n *html.Node, text *strings.Builder) {
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return
	}

	if n.Type == html.TextNode {
		content := strings.TrimSpace(n.Data)
		if content != "" {
			text.WriteString(" ")
			text.WriteString(content)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkHTMLNodes(c, text)
	}
}
