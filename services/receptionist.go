package services

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	Name           string
	Port           int
	ReverseProxy   *httputil.ReverseProxy
	RequestChannel chan RequestMessage
}

type ResponseMessage struct {
	Service *Service
	Err     error
}

type RequestMessage struct {
	Service         string
	ResponseChannel chan ResponseMessage
	Request         *http.Request
	Command         string
	Func            func(port int, rq string, elapsed int64)
	Elapsed         int64
	Rq              string
}

var RequestQueue = make(chan RequestMessage, 100)
var services = make(map[string]*Service)

func NewRequestMessage(svc string, rc chan ResponseMessage, r *http.Request) RequestMessage {
	return RequestMessage{Service: svc, ResponseChannel: rc, Request: r, Command: "", Func: nil}
}

func NewFunctionMessage(svc string, f func(int, string, int64), rq string, elapsed int64) RequestMessage {
	return RequestMessage{Service: svc, Command: "", Func: f, Elapsed: elapsed, Rq: rq}
}

func NewCommandMessage(cmd string) RequestMessage {
	return RequestMessage{Command: cmd}
}

func NewResponseMessage(s *Service, err error) ResponseMessage {
	return ResponseMessage{Service: s, Err: err}
}

func Receptionist() {
	for {
		msg := <-RequestQueue
		if msg.Command != "" {
			parts := strings.Split(msg.Command, ":")
			if parts[0] == "stop" {
				if parts[1] == "all" {
					for _, rs := range services {
						if rs.Name != strconv.Itoa(rs.Port) {
							delete(services, rs.Name)
							rs.Name = strconv.Itoa(rs.Port)
							services[rs.Name] = rs
							go stopService(rs)
						}
					}
					continue
				}

				s, ok := services[parts[1]]
				if ok {
					delete(services, parts[1])
					s.Name = strconv.Itoa(s.Port)
					services[s.Name] = s
					go stopService(s)
				}
			}
		} else {
			s, ok := services[msg.Service]
			if !ok {
				freePort := 9600
				for _, rs := range services {
					if rs.Port > freePort {
						freePort = rs.Port
					}
				}
				cs, err := createService(msg.Service, freePort+1)
				if err != nil {
					msg.ResponseChannel <- NewResponseMessage(nil, err)
					continue
				}
				services[msg.Service] = cs
				go ServiceWorker(cs, err)
				s = cs
			}
			s.RequestChannel <- msg
		}
	}
}

func ServiceWorker(s *Service, err error) {
	for {
		msg := <-s.RequestChannel
		if msg.Command == "stop" {
			return;
		} else if msg.Func == nil {
			msg.ResponseChannel <- NewResponseMessage(s, err)
		} else {
			go msg.Func(s.Port, msg.Rq, msg.Elapsed)
		}
	}
}

func getCommandString(svc string) (string, error) {
	files, err := ioutil.ReadDir("./" + svc)
	if err != nil {
		return "", err
	}

	ver := "def"
	for _, f := range files {
		if f.IsDir() {
			ver = f.Name()
		}
	}

	return "./" + svc + "/" + ver + "/" + svc, nil
}

func createService(svc string, port int) (*Service, error) {
	cmdStr, err := getCommandString(svc)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(cmdStr, strconv.Itoa(port))
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	if !isServiceReady(port) {
		return nil, errors.New("Service '" + svc + "' started but unavailable...")
	}

	p, err := makeProxy(port)
	if err != nil {
		return nil, err
	}

	s := new(Service)
	s.Port = port
	s.ReverseProxy = p
	s.Name = svc
	s.RequestChannel = make(chan RequestMessage, 100)

	return s, nil
}

func makeProxy(port int) (*httputil.ReverseProxy, error) {
	remote, err := url.Parse("http://localhost:" + strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	return proxy, nil
}

func pingService(port int) error {
	client := &http.Client{
		Timeout: time.Millisecond * 250,
	}

	resp, err := client.Get("http://localhost:" + strconv.Itoa(port) + "/echo")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), "OK") {
		return errors.New(string(body))
	}

	return nil
}

func stopService(s *Service) {
	time.Sleep(30 * time.Second)

	client := &http.Client{
		Timeout: time.Second * 120,
	}

	s.RequestChannel <- NewCommandMessage ("stop")

	resp, err := client.Get("http://localhost:" + strconv.Itoa(s.Port) + "/stop")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if !strings.Contains(string(body), "OK") {
		return
	}

	return
}

func isServiceReady(port int) bool {
	var err error
	for retry := 1; retry <= 50; retry++ {
		err = pingService(port)
		if err == nil {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
