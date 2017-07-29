# llmatch
## Description
llmatch collects and analyzes "#なかよしマッチ" tweets and serve a simple web-app.

(llmatch means lovelive-match...)

## requirement
- redis
- go command

## Features
- data-collecting with twitter API
- http server

## How it work
By using twitter search streaming API, ```llmatch twitter``` collects tweets which contain the term "なかよしマッチ".

As an example, @paka3m's tweet is raised.

    "#なかよしマッチ 176315 2人！募集中！"

This tweet is collected and analyzed as following.
```json
{
    "screen_name": "paka3m",
    "needed": 2,
    "text":  "#なかよしマッチ 176315 3人！募集中！",
}
```

and saved into redis with key ```match:176315```.

## Installation and Usages
### install with go command:

    go get github.com/paka3m/llmatch

### Usage: twitter data-collecting
e.g.

    llmatch twitter --at [AccountToken] --ats [AccountTokenSecret] --ck [ConsumerKey] --cks [ConsumerKeySecret] --redis localhost:6379

### Usage: server
[!] you should run ```llmatch twitter``` with ```llmatch serve``` together in parallel.

e.g.

    llmatch serve --port 8315 --redis localhost:6379

now you can access localhost:8315/match.

and... connect other servers or reverse-proxy such as nginx.

## TODO
- to apply other cases.
- to improve analysis alogorithm.
- to dockerize
- ...

## Link
This app uses "Umi" as the Web interface the Bootstrap theme. 
https://github.com/NKMR6194/Umi

## Author
paka3m

## Licence
MIT