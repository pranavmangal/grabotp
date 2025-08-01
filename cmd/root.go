package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"text/tabwriter"
	"time"

	"github.com/pranavmangal/grabotp/gmail"

	"github.com/urfave/cli/v3"
	"golang.design/x/clipboard"
)

var rootCmd = &cli.Command{
	Name:   "grabotp",
	Usage:  "A simple tool to fetch recent OTPs from your Gmail account.",
	Action: fetchOTPs,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output as JSON",
		},
	},
}

func fetchOTPs(ctx context.Context, cmd *cli.Command) error {
	otps := gmail.FetchOTPs()
	if len(otps) == 0 {
		if cmd.Bool("json") {
			fmt.Println("[]")
		} else {
			fmt.Println("No OTPs found.")
		}

		return nil
	}

	if cmd.Bool("json") {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(otps)
		if err != nil {
			return err
		}

		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	defer w.Flush()

	for _, otp := range otps {
		fmt.Fprintln(w, otp.Sender+"\t"+timeAgo(otp.Timestamp)+"\t"+otp.OTP)
	}

	mostRecentOTP := otps[0].OTP

	err := clipboard.Init()
	if err == nil {
		clipboard.Write(clipboard.FmtText, []byte(mostRecentOTP))
	}

	return nil
}

func timeAgo(t time.Time) string {
	diff := time.Since(t)

	seconds := int(math.Round(diff.Seconds()))
	if seconds < 60 {
		return fmt.Sprintf("%ds ago", seconds)
	}

	minutes := int(math.Round(diff.Minutes()))
	return fmt.Sprintf("%dm ago", minutes)
}

func Execute() {
	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
