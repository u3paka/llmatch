package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"gopkg.in/redis.v5"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/k0kubun/pp"
)

func TestRegisterMatch(t *testing.T) {
	txt := "#なかよしマッチ 632910!あと3人"
	r := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	reg := regexp.MustCompile(`(?P<id>\d{6})((@|.*(あと|残|残り|のこり))(?P<want>[1-3])(人|にん|名))*`)

	if strings.Contains(txt, "#なかよしマッチ") {
		exs := reg.FindAllStringSubmatch(txt, 1)
		if len(exs) != 0 {
			ex := exs[len(exs)-1]
			pp.Println(ex[1], ex[5])
			if ex[1] != "" {
				k := "match:" + ex[1]
				// r.Set("match:"+ex[1], ex[5], time.Second*60)
				pipe := r.Pipeline()
				pipe.HSet(k, "screen_name", "poipoi")
				pipe.HSet(k, "text", txt)
				pipe.HSet(k, "needed", ex[5])
				pipe.PExpire(k, time.Second*5)
				pipe.Exec()
			}
		}
	}
	// pp.Println()
}

func TestCallMatch(t *testing.T) {
	// txt := "#なかよしマッチ 632910!あと3人"
	r := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	res, err := r.Keys("match:*").Result()
	if err != nil || len(res) == 0 {
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
	pp.Println(d)
}

func TestSearchMatch(t *testing.T) {
	// txt := "#なかよしマッチ 632910!あと3人"
	r := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cks := "XXXXXXXXXXXXXXXXXXXXXXXXXX"
	ck := "XXXXXXXXXXXXXXXXXXXXXXX"
	at := "XXXXXXXXXXXXXXXXXXXX"
	ats := "XXXXXXXXXXXXXXXXXXXXXXXX"
	config := oauth1.NewConfig(ck, cks)
	token := oauth1.NewToken(at, ats)
	hcli := config.Client(oauth1.NoContext, token)
	twc := twitter.NewClient(hcli)
	stream, err := twc.Streams.Filter(&twitter.StreamFilterParams{
		Track: []string{"なかよしマッチ"},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	reg := regexp.MustCompile(`(?P<id>\d{6})((@|.*(あと|残|残り|のこり))(?P<want>[1-3])(人|にん|名))*`)

	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(tweet *twitter.Tweet) {
		txt := tweet.Text
		if strings.Contains(txt, "なかよしマッチ") {
			exs := reg.FindAllStringSubmatch(txt, 1)
			ex := exs[len(exs)-1]
			pp.Println(ex[1], ex[5])
			if ex[1] != "" {
				want, err := strconv.Atoi(ex[5])
				if err != nil {
					want = 0
				}
				r.Set("match:"+ex[1], want, time.Minute*5)
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
	return
}
