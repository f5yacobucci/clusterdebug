package consensus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/f5yacobucci/clusterdebug/pkg/config"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type (
	Member struct {
		Name string `json:"name" validate:"required"`
		IP   string `json:"ip" validate:"required,ip4_addr"`
		Seen uint   `json:"seen"`
	}
	MembersList struct {
		Server string  `json:"server"`
		Items  Members `json:"items"`
	}
	Members []Member

	Validator struct {
		validator *validator.Validate
	}
)

const (
	clusterAPI = "/cluster"
	membersAPI = "/members"
)

var members Members = make(Members, 0)

func NewValidator() *Validator {
	return &Validator{
		validator: validator.New(),
	}
}

func (v *Validator) Validate(i interface{}) error {
	if err := v.validator.Struct(i); err != nil {
		return err
	}
	return nil
}

func receiveHello(c echo.Context) error {
	dump, _ := httputil.DumpRequest(c.Request(), true)
	c.Logger().Printj(log.JSON{
		"message": "received request",
		"method":  "POST",
		"dump":    fmt.Sprintf("%q", dump),
	})
	c.Logger().Printf("%q", dump)

	var data Member
	if err := c.Bind(&data); err != nil {
		c.Logger().Printj(log.JSON{
			"err": err,
		})
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(data); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	data.Seen = 1
	for i := range members {
		if members[i].Name == data.Name {
			members[i].Seen = members[i].Seen + 1
			data = members[i]
			goto found
		}
	}
	members = append(members, data)

found:
	c.Logger().Printj(log.JSON{
		"data": fmt.Sprintf("%v", data),
	})
	return c.JSON(http.StatusCreated, data)
}

func get(c echo.Context) error {
	cc := c.(*config.ClusterContext)
	cc.Logger().Printj(log.JSON{
		"message": "received request",
		"method":  "GET",
	})
	return cc.JSON(http.StatusOK, MembersList{
		Server: fmt.Sprintf("%s - {%s}", cc.Config().Name, cc.Config().IP),
		Items:  members,
	})
}

func Run(conf config.Config, e *echo.Echo) {
	for {
		// Disco
		cname, srvs, err := net.LookupSRV("http", "tcp", conf.Domain)
		if err != nil {
			e.Logger.Printj(log.JSON{
				"err":     err,
				"reason":  "failed SRV lookup",
				"message": "using listen only mode - will not register",
			})
			goto wait
		}
		e.Logger.Printj(log.JSON{
			"message": "SRV lookup answer succeeded",
			"cname":   cname,
		})
		for i := range srvs {
			e.Logger.Printj(log.JSON{
				"target":   srvs[i].Target,
				"port":     srvs[i].Port,
				"priority": srvs[i].Priority,
				"weight":   srvs[i].Weight,
			})

			m := Member{
				Name: conf.Name,
				IP:   conf.IP,
			}
			b, err := json.Marshal(m)
			if err != nil {
				e.Logger.Printj(log.JSON{
					"err":     err,
					"message": "could not serialize cluster member",
					"data":    m,
				})
				continue
			}

			body := bytes.NewBuffer(b)
			req, _ := http.NewRequest(http.MethodPost,
				fmt.Sprintf("http://%s:%d%s",
					strings.TrimRight(srvs[i].Target, "."),
					srvs[i].Port,
					clusterAPI),
				body)
			req.Header.Add("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				e.Logger.Printj(log.JSON{
					"err":     err,
					"message": "failed posting to cluster",
					"target":  srvs[i].Target,
					"port":    srvs[i].Port,
					"buffer":  body.String(),
				})
				continue
			}

			dump, _ := httputil.DumpResponse(resp, true)
			e.Logger.Printj(log.JSON{
				"status": resp.Status,
				"dump":   fmt.Sprintf("%q", dump),
			})
			resp.Body.Close()
		}

	wait:
		<-time.After(time.Second * 10)
	}
}

func RegisterEndpoints(e *echo.Echo) {
	e.POST(clusterAPI, receiveHello)
	e.GET(membersAPI, get)
}
