package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/pranavmangal/grabotp/config"
	"github.com/pranavmangal/grabotp/gmail"

	"github.com/urfave/cli/v3"
	"golang.design/x/clipboard"
)

var rootCmd = &cli.Command{
	Name:                  "grabotp",
	Usage:                 "A simple tool to fetch recent OTPs from your Gmail accounts.",
	EnableShellCompletion: true,
	Action:                fetchOTPs,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output as JSON",
		},
	},
	Commands: []*cli.Command{
		{
			Name:  "account",
			Usage: "Manage your Gmail accounts",
			Commands: []*cli.Command{
				{
					Name:   "add",
					Usage:  "Add a new Gmail account",
					Action: addAccount,
				},
				{
					Name:   "list",
					Usage:  "List all configured Gmail accounts",
					Action: listAccounts,
				},
				{
					Name:      "remove",
					Usage:     "Remove a Gmail account",
					Action:    removeAccount,
					ArgsUsage: "<email>",
				},
			},
		},
		{
			Name:   "reset",
			Usage:  "Reset all your credentials",
			Action: resetCreds,
		},
	},
}

func addAccount(ctx context.Context, cmd *cli.Command) error {
	gmail.AddAccount()
	return nil
}

func listAccounts(ctx context.Context, cmd *cli.Command) error {
	accs, err := config.ListAccounts()
	if err != nil {
		return err
	}

	if len(accs) == 0 {
		fmt.Println("No accounts configured.")
		return nil
	}

	fmt.Println("Configured accounts:")
	for _, acc := range accs {
		fmt.Println("-", acc)
	}

	return nil
}

func removeAccount(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		return fmt.Errorf("Please provide an email to remove")
	}

	acc := cmd.Args().First()
	err := config.RemoveAccount(acc)
	if err != nil {
		return err
	}

	err = gmail.DeleteToken(acc)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully removed %s\n", acc)
	return nil
}

// Reset client ID and remove all accounts and tokens
func resetCreds(ctx context.Context, cmd *cli.Command) error {
	err := config.DeleteClientId()
	if err != nil {
		return err
	}

	accs, err := config.ListAccounts()
	if err != nil {
		return err
	}

	for _, acc := range accs {
		err := gmail.DeleteToken(acc)
		if err != nil {
			return err
		}
	}

	err = config.ResetAllAccounts()
	if err != nil {
		return err
	}

	fmt.Println("Reset successful!")
	return nil
}

func fetchOTPs(ctx context.Context, cmd *cli.Command) error {
	accounts, err := config.ListAccounts()
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		fmt.Println("No accounts configured. Please run `grabotp account add` to add one.")
		return nil
	}

	var wg sync.WaitGroup
	otpsChan := make(chan []gmail.ParsedOTP, len(accounts))

	for _, acc := range accounts {
		wg.Add(1)
		go func(acc string) {
			defer wg.Done()
			otpsChan <- gmail.FetchOTPs(acc)
		}(acc)
	}

	wg.Wait()
	close(otpsChan)

	allOtps := []gmail.ParsedOTP{}
	for otps := range otpsChan {
		allOtps = append(allOtps, otps...)
	}

	sort.Slice(allOtps, func(i, j int) bool {
		return allOtps[i].Timestamp.After(allOtps[j].Timestamp)
	})

	if cmd.Bool("json") {
		outputJson(allOtps)
	} else {
		outputTable(allOtps)

		if len(allOtps) > 0 {
			mostRecentOTP := allOtps[0].OTP
			copyLatestToClipboard(mostRecentOTP)
		}
	}

	return nil
}

func outputJson(otps []gmail.ParsedOTP) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(otps)
	if err != nil {
		return err
	}

	return nil
}

func outputTable(otps []gmail.ParsedOTP) {
	if len(otps) == 0 {
		fmt.Println("No OTPs found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	defer w.Flush()

	for _, otp := range otps {
		fmt.Fprintln(w, otp.From+"\t"+timeAgo(otp.Timestamp)+"\t"+otp.OTP)
	}
}

func copyLatestToClipboard(mostRecentOTP string) {
	err := clipboard.Init()
	if err == nil {
		clipboard.Write(clipboard.FmtText, []byte(mostRecentOTP))
	}
}

// Returns a human readable time string
func timeAgo(t time.Time) string {
	diff := time.Since(t)

	seconds := int(math.Round(diff.Seconds()))
	if seconds < 60 {
		return fmt.Sprintf("%ds ago", seconds)
	}

	minutes := int(math.Round(diff.Minutes()))
	return fmt.Sprintf("%dm ago", minutes)
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
