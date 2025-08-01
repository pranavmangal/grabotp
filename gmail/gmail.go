package gmail

import (
	"encoding/base64"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pranavmangal/grabotp/otp"

	"google.golang.org/api/gmail/v1"
)

type ParsedOTP struct {
	Sender    string
	Timestamp time.Time
	OTP       string
}

func FetchOTPs() []ParsedOTP {
	srv := GetClient()

	user := "me"
	halfHourAgo := time.Now().Add(-time.Minute * 30).Unix()
	query := "after:" + strconv.FormatInt(halfHourAgo, 10)

	r, err := srv.Users.Messages.List(user).Q(query).MaxResults(5).Do()
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
				otpsChan <- ParsedOTP{Sender: sender, Timestamp: timestamp, OTP: otp}
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
	if msg.Payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		if err == nil {
			return string(decoded)
		}

	} else {
		// Handle multipart emails
		for _, part := range msg.Payload.Parts {
			if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
				decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return string(decoded)
				}
			}
		}
	}

	return ""
}
