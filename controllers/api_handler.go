package controllers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/netrisai/netris-operator/configloader"
)

// HTTPReply struct
type HTTPReply struct {
	Data       []byte
	StatusCode int
	Status     string
	Error      error
}

// ConductorAddresses struct
type ConductorAddresses struct {
	General            string
	Auth               string
	Kubenet            string
	KubenetClusterInfo string
	KubenetNode        string
	KubenetLB          string
	KubenetAPIStatus   string
	L4LB               string
	VNet               string
}

var conductorAddresses = ConductorAddresses{
	General:            "/api/general",
	Auth:               "/api/auth",
	Kubenet:            "/api/kubenet",
	KubenetClusterInfo: "/api/kubenet/clusterinfo",
	KubenetNode:        "/api/kubenet/node",
	KubenetLB:          "/api/kubenet/l4lb/connect",
	KubenetAPIStatus:   "/api/kubenet/changeapistatus",
	L4LB:               "/api/kubenet/l4lb",
	VNet:               "/api/v-net",
}

// HTTPCred stores the credentials for connect to http server. User, Password, HTTP Clien e.t.c
type HTTPCred struct {
	sync.Mutex
	URL        url.URL
	LoginData  loginData
	Cookies    []http.Cookie
	ConnectSID string
	Timeout    int
}

type loginData struct {
	Login      string `json:"user"`
	Password   string `json:"password"`
	AuthScheme int    `json:"auth_scheme_id"`
}

var cred *HTTPCred

func init() {
	var err error
	cred, err = newHTTPCredentials(10)
	if err != nil {
		log.Panicf("newHTTPCredentials error %v", err)
	}
	err = cred.LoginUser()
	if err != nil {
		log.Printf("LoginUser error %v", err)
	}
	go cred.checkAuthWithInterval()
}

func newHTTPCredentials(timeout int) (*HTTPCred, error) {
	URL, err := url.Parse(configloader.Root.Controller.Host)
	if err != nil {
		return nil, fmt.Errorf("{newHTTPCredentials} %s", err)
	}
	if timeout == 0 {
		timeout = 5
	}
	return &HTTPCred{
		URL: *URL,
		LoginData: loginData{
			Login:    configloader.Root.Controller.Login,
			Password: configloader.Root.Controller.Password,
		},
		Timeout: timeout,
	}, nil
}

// CheckAuth checks the user authorized or not
func (cred *HTTPCred) CheckAuth() error {
	reply, err := cred.Get(cred.URL.String() + conductorAddresses.Auth)
	if err != nil {
		return fmt.Errorf("{CheckAuth} %s", err)
	}
	if reply.StatusCode == http.StatusOK {
		return nil
	} else if len(reply.Data) > 0 {
		return fmt.Errorf("{CheckAuth} %s", reply.Data)
	}
	return fmt.Errorf("{CheckAuth} not authorized")
}

// LoginUser login the user and get the cookies for future use
func (cred *HTTPCred) LoginUser() error {
	cred.Lock()
	defer cred.Unlock()
	reqData := fmt.Sprintf("user=%s&password=%s&auth_scheme_id=1", cred.LoginData.Login, cred.LoginData.Password)

	URL := cred.URL.String() + conductorAddresses.Auth

	req, err := http.NewRequest("POST", URL, bytes.NewBufferString(reqData))
	if err != nil {
		return fmt.Errorf("{LoginUser} %s", err)
	}

	client := http.Client{
		Timeout: time.Duration(cred.Timeout) * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("{LoginUser} %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("{LoginUser} Authentication failed")
	}

	var cookies []http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "connect.sid" {
			cred.ConnectSID = cookie.Value
		}
		cookies = append(cookies, *cookie)
	}
	cred.Cookies = cookies
	return nil
}

// Get custom request
func (cred *HTTPCred) Get(address string) (reply HTTPReply, err error) {
	request, err := http.NewRequest("GET", address, nil)
	if err != nil {
		return reply, fmt.Errorf("{Get} [%s] %s", address, err)
	}

	request.Header.Set("Content-type", "application/json")
	cred.Lock()
	for _, cookie := range cred.Cookies {
		cook := cookie
		request.AddCookie(&cook)
	}
	cred.Unlock()
	client := http.Client{
		Timeout: time.Duration(cred.Timeout) * time.Second,
	}

	resp, err := client.Do(request)
	if err != nil {
		return reply, fmt.Errorf("{Get} [%s] %s", address, err)
	}

	reply.StatusCode = resp.StatusCode
	reply.Status = resp.Status

	defer respBodyClose(resp)
	reply.Data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return reply, fmt.Errorf("{Get} [%s] %s", address, err)
	}
	return reply, err
}

// CustomBodyRequest impelements the POST, PUT, UPDATE requests
func (cred *HTTPCred) CustomBodyRequest(method string, address string, data []byte) (reply HTTPReply, err error) {
	requestBody := bytes.NewBuffer(data)
	request, err := http.NewRequest(method, address, requestBody)
	if err != nil {
		return reply, fmt.Errorf("{CustomBodyRequest} [%s] [%s] %s", method, address, err)
	}

	request.Header.Set("Content-type", "application/json")
	cred.Lock()
	for _, cookie := range cred.Cookies {
		cook := cookie
		request.AddCookie(&cook)
	}
	cred.Unlock()

	client := http.Client{
		Timeout: time.Duration(cred.Timeout) * time.Second,
	}

	resp, err := client.Do(request)
	if err != nil {
		return reply, fmt.Errorf("{CustomBodyRequest} [%s] [%s] %s", method, address, err)
	}

	reply.StatusCode = resp.StatusCode
	reply.Status = resp.Status

	defer respBodyClose(resp)
	reply.Data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return reply, fmt.Errorf("{CustomBodyRequest} [%s] [%s] %s", method, address, err)
	}

	return reply, nil
}

func respBodyClose(resp *http.Response) {
	err := resp.Body.Close()
	if err != nil {
		log.Printf("resp.Body.Close() error: %v", err)
	}
}

// Post custom request
func (cred *HTTPCred) Post(address string, data []byte) (reply HTTPReply, err error) {
	return cred.CustomBodyRequest("POST", address, data)
}

// Put custom request
func (cred *HTTPCred) Put(address string, data []byte) (reply HTTPReply, err error) {
	return cred.CustomBodyRequest("PUT", address, data)
}

// Delete custom request
func (cred *HTTPCred) Delete(address string, data []byte) (reply HTTPReply, err error) {
	return cred.CustomBodyRequest("DELETE", address, data)
}

func (cred *HTTPCred) checkAuthWithInterval() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			err := cred.CheckAuth()
			if err != nil {
				log.Println(err)
				err := cred.LoginUser()
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}
