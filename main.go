package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"typosquad/database"

	"github.com/gempir/go-twitch-irc/v3"
	"zntr.io/typogenerator"
	"zntr.io/typogenerator/strategy"
)

var opt struct {
	TMI struct {
		Username string
		Token    string
		Channel  string
	}
	GitToken string
}

func init() {
	// log handler to file and console out
	logFile, err := os.OpenFile("logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	flag.StringVar(&opt.GitToken, "git", "", "git token")
	flag.StringVar(&opt.TMI.Username, "username", "", "twitch username")
	flag.StringVar(&opt.TMI.Token, "token", "", "twitch token")
	flag.StringVar(&opt.TMI.Channel, "channel", "", "channel to join")
	flag.Parse()
}

func main() {
	db := database.InitDB()

	file, err := os.OpenFile("domains.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	client := twitch.NewClient(opt.TMI.Username, opt.TMI.Token)

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.User.ID == "19264788" {
			return
		}

		if strings.EqualFold(message.User.Name, message.Channel) || message.User.ID == "253366808" {
			if strings.HasPrefix(message.Message, "!export") {
				client.Say(message.Channel, "export started => generating typosquad lists")
				domains, err := db.CrawlMessagesByRegex(`(https?:\/\/)?((?:[-a-z0-9._~!$&\'()*+,;=]|%[0-9a-f]{2})+(?::(?:[-a-z0-9._~!$&\'()*+,;=]|%[0-9a-f]{2})+)?@)?(?:((?:(?:\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5])\.){3}(?:\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5]))|((?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z][a-z0-9-]*[a-z0-9]))(:\d+)?((?:\/(?:[-a-z0-9._~!$&\'()*+,;=:@]|%[0-9a-f]{2})+)*\/?)(\?(?:[-a-z0-9._~!$&\'()*+,;=:@\/?]|%[0-9a-f]{2})*)?(\#(?:[-a-z0-9._~!$&\'()*+,;=:@\/?]|%[0-9a-f]{2})*)?`)
				if err != nil {
					log.Println(err)
					return
				}
				if len(domains) > 0 {
					gist := struct {
						Description string `json:"description"`
						Public      bool   `json:"public"`
						Files       map[string]struct {
							Content string `json:"content"`
						} `json:"files"`
					}{
						Description: "PiHole Filter List raw good and bad squated",
						Public:      false,
						Files: map[string]struct {
							Content string "json:\"content\""
						}{},
					}

					domains = func(input []string) []string {
						var temp map[string]bool = make(map[string]bool)
						for _, v := range input {

							// remove known subdomains
							v = strings.TrimPrefix(v, "www.")

							temp[v] = true
						}

						input = []string{}
						for k := range temp {
							input = append(input, k)
						}

						return input
					}(domains)

					var count int = 0
					for _, chunk := range ChunkSlice(domains, 1000000) {
						gist.Files[fmt.Sprintf("raw_%06d.txt", count)] = struct {
							Content string "json:\"content\""
						}{
							Content: strings.Join(chunk, "\n"),
						}
						count += 1
					}

					count = 0
					for _, chunk := range ChunkSlice(gengen(domains), 1000000) {
						gist.Files[fmt.Sprintf("squat_%06d.txt", count)] = struct {
							Content string "json:\"content\""
						}{
							Content: strings.Join(chunk, "\n"),
						}
						count += 1
					}

					b, err := json.Marshal(gist)
					if err != nil {
						log.Println(err)
						return
					}
					buf := bytes.NewBuffer(b)

					c := http.Client{}
					req, err := http.NewRequest(http.MethodPost, "https://api.github.com/gists", buf)
					if err != nil {
						log.Println(err)
					}

					req.Header.Set("Accept", "application/vnd.github+json")
					req.Header.Set("Authorization", "Bearer "+opt.GitToken)

					resp, err := c.Do(req)
					if err != nil {
						log.Println(err)
					}
					defer resp.Body.Close()

					var responseObj map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&responseObj)
					if err != nil {
						log.Fatal("Response JSON Error: ", err)
					}
					log.Println(responseObj["html_url"])
					client.Say(message.Channel, fmt.Sprint(responseObj["html_url"]))
				}
				return
			}
		}

		err := db.AddMessage(message.ID, message.Time, message.User.ID, message.User.Name, message.Message)
		if err != nil {
			log.Println(err)
		}
		log.Println(message.User.Name, message.User.ID, message.Message)

	})

	client.OnClearMessage(func(message twitch.ClearMessage) {
		db.DeleteMessage(message.TargetMsgID)
	})

	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		fmt.Println(message)
	})

	client.Join(opt.TMI.Channel)

	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func gengen(input []string) []string {
	strategys := []strategy.Strategy{
		strategy.Transposition,
		strategy.Repetition,
		strategy.Addition,
		strategy.BitSquatting,
		//strategy.Homoglyph,
		strategy.Hyphenation,
		strategy.Omission,
		strategy.Prefix,
		//strategy.SubDomain,
		strategy.TLDRepeat,
	}

	var squats []string = make([]string, 0)
	for _, v := range input {
		typos, err := typogenerator.FuzzDomain(v, strategys...)
		if err != nil {
			log.Println(err)
		}
		for _, permus := range typos {
			squats = append(squats, permus.Permutations...)
		}
	}

	sort.Strings(squats)
	return squats
}

func ChunkSlice[Slice any](input []Slice, size int) [][]Slice {
	var chunk []Slice
	chunks := make([][]Slice, 0, len(input)/size+1)
	for len(input) >= size {
		chunk, input = input[:size], input[size:]
		chunks = append(chunks, chunk)
	}
	if len(input) > 0 {
		chunks = append(chunks, input)
	}
	return chunks
}
