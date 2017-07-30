package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/urfave/cli"
	redis "gopkg.in/redis.v5"
)

type Match struct {
	ScreenName string
	ID         string
	Needed     int
	TTL        time.Duration
}

func main() {

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "redis",
			Value: "localhost:6379",
			Usage: "redis address",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "serve",
			Aliases: []string{"s"},
			Usage:   "serve",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port,p",
					Value: 8089,
					Usage: "a port number to serve",
				},
				cli.StringFlag{
					Name:  "template",
					Value: "static/index.html",
					Usage: "a template to serve",
				},
				cli.StringFlag{
					Name:  "crt",
					Value: "crt/index.html",
					Usage: "a server crt file",
				},
				cli.StringFlag{
					Name:  "key",
					Value: "static/index.html",
					Usage: "a server crt key file",
				},
			},
			Action: func(clc *cli.Context) error {
				// txt := "#なかよしマッチ 632910!あと3人"
				r := redis.NewClient(&redis.Options{
					Addr: clc.GlobalString("redis"),
				})

				t := template.Must(template.ParseGlob(clc.String("template")))
				http.HandleFunc("/html/index.html", func(w http.ResponseWriter, rq *http.Request) {
					t.ExecuteTemplate(w, "index", nil)
				})
				http.HandleFunc("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)
				http.HandleFunc("/match", func(w http.ResponseWriter, rq *http.Request) {
					bufb := new(bytes.Buffer)
					bufb.ReadFrom(rq.Body)
					body := bufb.String()
					fmt.Println(body)

					res, err := r.Keys("match:*").Result()
					if err != nil {
						fmt.Println(err)
						return
					}
					d := make([]Match, len(res))
					for i, k := range res {
						d[i].TTL = r.PTTL(k).Val()
						d[i].ID = strings.TrimPrefix(k, "match:")
						km, _ := r.HGetAll(k).Result()
						d[i].Needed, err = strconv.Atoi(km["needed"])

						if err != nil {
							d[i].Needed = 3
						}
						d[i].ScreenName = km["screen_name"]
					}

					sort.Slice(d, func(i, j int) bool {
						if d[i].Needed < d[j].Needed {
							return true
						}
						if d[i].TTL < d[j].TTL {
							return true
						}
						return false
					})

					t.ExecuteTemplate(w, "match", d)
				})
				http.ListenAndServe(":"+strconv.Itoa(clc.Int("port")), nil)
				// err := http.ListenAndServeTLS(":"+strconv.Itoa(clc.Int("port")), clc.String("crt"), clc.String("key"), nil)
				// if err != nil {
				// 	log.Fatal("ListenAndServe: ", err)
				// }
				return nil
			},
		},
		{
			Name:    "twitter",
			Aliases: []string{"t"},
			Usage:   "collect data with twitter stream",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "atoken,at",
					Value: "",
					Usage: "account token",
				},
				cli.StringFlag{
					Name:  "atokensecret,ats",
					Value: "",
					Usage: "account token secret",
				},
				cli.StringFlag{
					Name:  "ckey,ck",
					Value: "",
					Usage: "consumer key",
				},
				cli.StringFlag{
					Name:  "ckeysecret,cks",
					Value: "",
					Usage: "consumer key secret",
				},
			},
			Action: func(clc *cli.Context) error {
				r := redis.NewClient(&redis.Options{
					Addr: clc.GlobalString("redis"),
				})
				config := oauth1.NewConfig(clc.String("ck"), clc.String("cks"))
				token := oauth1.NewToken(clc.String("at"), clc.String("ats"))
				hcli := config.Client(oauth1.NoContext, token)
				twc := twitter.NewClient(hcli)
				stream, err := twc.Streams.Filter(&twitter.StreamFilterParams{
					Track: []string{"なかよしマッチ"},
				})
				if err != nil {
					fmt.Println(err)
					return err
				}
				reg := regexp.MustCompile(`(?P<id>\d{6})((@|.*(あと|残|残り|のこり))(?P<want>[1-3])(人|にん|名))*`)

				demux := twitter.NewSwitchDemux()
				demux.Tweet = func(tweet *twitter.Tweet) {
					txt := tweet.Text
					if strings.Contains(txt, "なかよしマッチ") {
						exs := reg.FindAllStringSubmatch(txt, 1)
						if len(exs) == 0 {
							return
						}
						ex := exs[len(exs)-1]
						fmt.Println(tweet.User.ScreenName, ex[1], ex[5])
						if ex[1] != "" {
							k := "match:" + ex[1]
							pipe := r.Pipeline()
							pipe.HSet(k, "screen_name", tweet.User.ScreenName)
							pipe.HSet(k, "text", tweet.Text)
							pipe.HSet(k, "needed", ex[5])
							pipe.PExpire(k, time.Minute*5)
							pipe.Exec()
						}
					}
				}
				go demux.HandleChan(stream.Messages)

				// Wait for SIGINT and SIGTERM (HIT CTRL-C)
				ch := make(chan os.Signal)
				signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
				select {
				case <-ch:
					fmt.Println("Stopping Stream...")
					stream.Stop()
				}
				os.Exit(1)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
