package main

import (
	"./socks5"
	"container/list"
	"encoding/json"
	"io/ioutil"
	"log"
	"regexp"
)

type Users struct {
	User		string `json:"user"`
	Pass		string `json:"pass"`
}

type Config struct {
	Sock5Addr	string `json:"addr"`
	UserList	[]Users `json:"userlist"`
	Pattern		[]string `json:"pattern"`
}

func (c *Config) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func LoadConfig(s string) (*Config, error) {
	data, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, err
	}
	cConfig := &Config{}
	if err = json.Unmarshal(data, cConfig); err != nil {
		return nil, err
	}
	return cConfig, nil
}

func main() {
	conf, err := LoadConfig("socks5-proxy.config")
	if err != nil {
		log.Println("load configuration failed, err:", err)
		return
        }

	
	patterns := list.New()
	for _, pattern := range conf.Pattern {
		r, _ := regexp.Compile(pattern)
		patterns.PushBack(r)
	}

	srv := socks5.New()
	srv.AuthUsernamePasswordCallback = func(c *socks5.Conn, username, password []byte) error {
		user := string(username)
		pass := string(password)
		for _, uid := range conf.UserList {
			if user == uid.User {
				if pass != uid.Pass {
					log.Printf("User Refused/Password mismatched: '%v'", user)
					return socks5.ErrAuthenticationFailed
				} else {
					log.Printf("User Connected: '%v'", user)
					c.Data = user
					return nil
				}
			}
		}
		log.Printf("User Refused/No such user: '%v'", user)
		return socks5.ErrAuthenticationFailed
	}

	srv.HandleConnectFunc(func(c *socks5.Conn, host string) (newHost string, err error) {
		for e := patterns.Front(); e != nil; e = e.Next() {
			pattern := e.Value.(*regexp.Regexp)
			if pattern.MatchString(host) {
				return host, socks5.ErrConnectionNotAllowedByRuleset
			}
		}

		if user, ok := c.Data.(string); ok {
			log.Printf("%v connecting to %v", user, host)
		}
		return host, nil
	})

	srv.HandleCloseFunc(func(c *socks5.Conn) {
		if user, ok := c.Data.(string); ok {
			log.Printf("Goodbye %v!", user)
		}
	})

	srv.ListenAndServe(conf.Sock5Addr)
}