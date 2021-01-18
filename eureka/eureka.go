package eureka

import (
	"bytes"
	"encoding/xml"
	"github.com/shumybest/ragnaros2/config"
	"github.com/shumybest/ragnaros2/feign"
	"github.com/shumybest/ragnaros2/log"
	"github.com/shumybest/ragnaros2/utils"
	"strings"
	"sync"
	"time"
)

var logger = log.GetLoggerInstance()

const (
	AppsUrl = "apps/"
)

type Client struct {
	Instance         InstanceConfig
	Status           string
}

var instance *Client
var once sync.Once
func GetClientInstance() *Client {
	once.Do(func() {
		instance = &Client{}
	})
	return instance
}

var eurekaServiceUrl string

func (c *Client) Register() {
	eurekaServiceUrl = config.GetConfigString("eureka.client.service-url.defaultZone")
	if eurekaServiceUrl == "" {
		logger.Warn("Eureka Service URL is empty, running into mono mode")
		c.Status = OUT_OF_SERVICE
		return
	}

	c.Instance = composeInstance()
	buf, _ := xml.Marshal(c.Instance)
	registerUrl := eurekaServiceUrl + AppsUrl + c.Instance.App

	logger.Info("trying to register to Eureka: " + registerUrl)

	resp, err := utils.RetryableClient().
		SetHeader("Content-Type", "application/xml").
		SetBody(bytes.NewBuffer(buf)).
		Post(registerUrl)

	if err != nil {
		logger.Error(err)
		c.Status = OUT_OF_SERVICE
		return
	}

	if resp.StatusCode() == 204 || resp.StatusCode() == 200 {
		logger.Info("Eureka Client Register Succeed")
		c.Status = UP
		go c.clientRefresh()
	} else {
		c.Status = OUT_OF_SERVICE
		logger.Warnf("Eureka Client Register Failed: %s %s", resp.StatusCode(), resp)
	}
}

func (c *Client) clientRefresh() {
	defer c.unRegister()
	appsUrl := eurekaServiceUrl + AppsUrl
	instanceUrl := appsUrl + c.Instance.App + "/" + c.Instance.InstanceId

	for {
		// heartbeat
		client := utils.RetryableClient()
		if resp, err := client.Put(instanceUrl); err == nil {
			if resp.StatusCode() != 204 && resp.StatusCode() != 200 {
				c.Status = UNKNOWN
				logger.Warnf("Eureka Client Renew Failed: %s %s\n", resp)

				// perform register again
				c.unRegister()
				c.Register()
				break
			}
		} else {
			logger.Error(err)
			c.Status = UNKNOWN
			return
		}

		// get apps
		if resp, err := client.Get(appsUrl); err == nil {
			if resp.StatusCode() == 200 {
				var apps ApplicationsResponse
				if err = xml.Unmarshal(resp.Body(), &apps); err == nil {
					for _, app := range apps.Applications {
						var instances []feign.Instance
						for _, inst := range app.Instances {
							instances = append(instances, feign.Instance{
								HomePageUrls: inst.HomePageUrl,
								Status:       inst.Status,
							})
						}
						feign.Applications[strings.ToLower(app.Name)] = instances
					}
				}
			}
		} else {
			logger.Error(err)
			c.Status = UNKNOWN
			return
		}

		logger.Debugf("application refresh: %v\n", feign.Applications)
		config.SetConfig("ragnaros.conf.applications", feign.Applications)
		c.Status = UP
		time.Sleep(10 * time.Second)
	}
}

func (c *Client) unRegister() {
	instanceUrl := eurekaServiceUrl + AppsUrl + c.Instance.App + "/" + c.Instance.InstanceId

	resp, err := utils.RetryableClient().Delete(instanceUrl)
	if err != nil {
		c.Status = UNKNOWN
		logger.Error(err)
		return
	}

	if resp.StatusCode() != 204 && resp.StatusCode() != 200 {
		c.Status = UNKNOWN
		logger.Warnf("Eureka Client Delete Failed: %s %s", resp.StatusCode(), resp)
	} else {
		c.Status = DOWN
	}
}