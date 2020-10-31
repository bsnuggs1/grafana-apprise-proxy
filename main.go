package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type requestPayloadStruct struct {
	DashboardID *int `json:"dashboardId"`
	EvalMatches []struct {
		Value  int    `json:"value"`
		Metric string `json:"metric"`
		Tags   struct {
		} `json:"tags"`
	} `json:"evalMatches"`
	ImageURL string `json:"imageUrl"`
	Message  string `json:"message"`
	OrgID    int    `json:"orgId"`
	PanelID  int    `json:"panelId"`
	RuleID   int    `json:"ruleId"`
	RuleName string `json:"ruleName"`
	RuleURL  string `json:"ruleUrl"`
	State    string `json:"state"`
	Tags     struct {
		TagName string `json:"tag name"`
	} `json:"tags"`
	Title string `json:"title"`
}

type requestAppRise struct {
	Urls  string `json:"urls"`
	Body  string `json:"body"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type conf struct {
	PortOverride *int64 `yaml:"port"`
	URL          string `yaml:"url"`
}

func (c *conf) Port() int64 {
	if c.PortOverride == nil {
		return 1445
	}
	return *c.PortOverride
}

func (c *conf) load() error {
	yamlErr := c.loadByYaml()
	if yamlErr != nil {
		logrus.Info("Unable to load configuration file, falling back to environment variables.")
		envErr := c.loadByEnv()
		if envErr != nil {
			return fmt.Errorf("Fallback failed. Unable to load configuration from environment variables: \n %s \n\n %s", envErr.Error(), yamlErr.Error())
		}
	}

	return nil
}

func (c *conf) loadByYaml() error {
	path := "/etc/grafana-apprise-proxy/conf.yml"

	yamlFile, err := ioutil.ReadFile("conf.yml")
	if err != nil {
		yamlFile, err = ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to find configuration file at %s", path)
		}
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return err
	}

	if len(c.URL) == 0 {
		return fmt.Errorf("missing parameter 'url' from configuration file")
	}

	return nil
}

func (c *conf) loadByEnv() error {
	portOverrideStr := os.Getenv(environmentNamePortOverride)
	URL := os.Getenv(environmentNameURL)

	if len(URL) == 0 {
		return fmt.Errorf("unable to locate environment variable, %s", environmentNameURL)
	}
	c.URL = URL

	if len(portOverrideStr) > 0 {
		portOverrideInt, err := strconv.ParseInt(portOverrideStr, 10, 64)
		if err != nil {
			logrus.Error(err)
		} else {
			c.PortOverride = &portOverrideInt
		}

	}

	return nil
}

var configuration = conf{}
var environmentNamePortOverride = "GRAFANA_APPRISE_PROXY_TARGET_PORT"
var environmentNameURL = "GRAFANA_APPRISE_PROXY_TARGET_URL"

// Get a json decoder for a given requests body
func requestBodyDecoder(request *http.Request) *json.Decoder {
	// Read body to buffer
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logrus.Panicf("Error reading body: %v", err)
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any susequent calls
	// are unable to read the body again....
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(body)))
}

// parseRequestBody Parse the requests body
func parseRequestBody(request *http.Request) requestPayloadStruct {
	decoder := requestBodyDecoder(request)

	var requestPayload requestPayloadStruct
	err := decoder.Decode(&requestPayload)

	if err != nil {
		panic(err)
	}

	return requestPayload
}

func updatePayload(requestionPayload requestPayloadStruct) requestAppRise {
	newPayload := requestAppRise{}
	newPayload.Body = requestionPayload.Message
	newPayload.Title = requestionPayload.Title

	switch alertType := requestionPayload.State; alertType {
	case "ok":
		newPayload.Type = "success"
	case "alerting", "no_data":
		newPayload.Type = "failure"
	case "pending":
		newPayload.Type = "warning"
	default:
		newPayload.Type = "info"
	}

	return newPayload
}

// serveReverseProxy Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// handleRequestAndRedirect: Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	requestPayload := parseRequestBody(req)
	url := configuration.URL

	if requestPayload.DashboardID != nil {
		logrus.Infof("converting request from %s...", req.URL.Host)

		newPayload := updatePayload(requestPayload)
		payloadBytes, err := json.Marshal(newPayload)
		if err != nil {
			logrus.Error(err)
		}
		payloadBuffer := bytes.NewBuffer(payloadBytes)

		req.Body = ioutil.NopCloser(payloadBuffer)
		req.ContentLength = int64(payloadBuffer.Len())
	}

	serveReverseProxy(url, res, req)
}

// Get the port to listen on
func getListenAddress() string {
	return fmt.Sprintf(":%d", configuration.Port())
}

// Log the env variables required for a reverse proxy
func logSetup() {
	logrus.Infof("Listening on port: %d\n", configuration.Port())
	logrus.Infof("Redirecting all requests to url: %s\n", configuration.URL)
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		logrus.Panic(err)
	}
	exPath := filepath.Dir(ex)
	logrus.Infof("Executing from %s", exPath)
	os.Chdir(exPath)

	configErr := configuration.load()
	if configErr != nil {
		logrus.Fatalf("The Grafana-Apprise-Proxy needs at least a target URL to proxy against to run.\n %s", configErr.Error())
	}

	// Log setup values
	logSetup()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
