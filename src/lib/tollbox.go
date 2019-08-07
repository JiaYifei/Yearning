// Copyright 2019 HenryYee.
//
// Licensed under the AGPL, Version 3.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"Yearning-go/src/modal"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"gopkg.in/ldap.v3"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"
)

func SuperAuth(c echo.Context, r string) bool {
	var p modal.CoreGrained
	var px modal.PermissionList
	user, role := JwtParse(c)
	if role == "admin" {
		modal.DB().Model(&modal.CoreGrained{}).Where("username =?", user).First(&p)

		if err := json.Unmarshal(p.Permissions, &px); err != nil {
			c.Logger().Error(err.Error())
			return false
		}

		switch r {
		case "user":
			return px.User == "1"
		case "db":
			return px.Base == "1"
		}

	}
	return false
}

func ResearchDel(s []string, p string) []string {
	for in := 0; in < len(s); in++ {
		if s[in] == p {
			s = append(s[:in], s[in+1:]...)
			in--
		}
	}
	return s
}

func Paging(page interface{}, total int) (start int, end int) {
	var i int
	switch v := page.(type) {
	case string:
		i, _ = strconv.Atoi(v)
	case int:
		i = v
	}
	start = i*total - total
	end = total
	return
}

func LdapConnenct(c echo.Context, l *modal.Ldap, user string, pass string, isTest bool) bool {

	var s string

	ld, err := ldap.Dial("tcp", l.Url)
	if err != nil {
		c.Logger().Error(err.Error())
		return false
	}
	defer ld.Close()

	if ld != nil {
		if err := ld.Bind(l.User, l.Password); err != nil {
			return false
		}
		if isTest {
			return true
		}

	}

	if l.Type == 1 {
		s = fmt.Sprintf("(sAMAccountName=%s)", user)
	} else if l.Type == 2 {
		s = fmt.Sprintf("(uid=%s)", user)
	} else {
		s = fmt.Sprintf("(cn=%s)", user)
	}

	searchRequest := ldap.NewSearchRequest(
		l.Sc,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=organizationalPerson)%s)", s),
		[]string{"dn"},
		nil,
	)

	sr, err := ld.Search(searchRequest)

	if err != nil {
		log.Println(err.Error())
		return false
	}

	if len(sr.Entries) != 1 {
		log.Println("User does not exist or too many entries returned")
		return false
	}

	userdn := sr.Entries[0].DN

	if err := ld.Bind(userdn, pass); err != nil {
		c.Logger().Error(err.Error())
		return false
	}
	return true
}

func Axis() []string {
	var s []string
	currentTime := time.Now()
	for a := 0; a < 7; a++ {
		oldTime := currentTime.AddDate(0, 0, -a)
		s = append(s, oldTime.Format("2006-01-02"))
	}
	return s
}

func GenWorkid() string {
	rand.Seed(time.Now().Unix())
	a := rand.Intn(1000)
	c := strconv.Itoa(a)
	now := time.Now()
	return now.Format("20060102150405") + c
}

func Intersect(o, n []string) []string {
	m := make(map[string]int)
	var arr []string
	for _, v := range o {
		m[v]++
	}
	for _, v := range n {
		m[v]++
		if m[v] > 1 {
			arr = append(arr, v)
		}
	}
	return arr
}

func NonIntersect(o, n []string) []string {
	m := make(map[string]int)
	var arr []string
	for _, v := range o {
		m[v]++
	}
	for _, v := range n {
		m[v]++
		if m[v] == 1 {
			arr = append(arr, v)
		}
	}
	return arr
}

func MessagePush(c echo.Context, workid string, t uint, reject string) {
	var user modal.CoreAccount
	var o modal.CoreSqlOrder
	var ding, mail string
	modal.DB().Select("work_id,username,text,assigned").Where("work_id =?", workid).First(&o)
	modal.DB().Select("email").Where("username =?", o.Username).First(&user)
	s := modal.GloMessage
	s.ToUser = user.Email

	if t == 0 {
		ding = fmt.Sprintf(TmplRejectDing, o.WorkId, o.Username, o.Assigned, modal.Host, o.Text, reject)
		mail = fmt.Sprintf(TmplRejectMail, o.WorkId, o.Username, modal.Host, modal.Host, reject)
	}

	if t == 1 {
		ding = fmt.Sprintf(TmplSuccessDing, o.WorkId, o.Username, o.Assigned, modal.Host, o.Text)
		mail = fmt.Sprintf(TmplMail, "执行", o.WorkId, o.Username, modal.Host, modal.Host, "执行成功")
	}

	if t == 2 {
		modal.DB().Select("email").Where("username =?", o.Assigned).First(&user)
		s.ToUser = user.Email
		ding = fmt.Sprintf(TmplReferDing, o.WorkId, o.Username, o.Assigned, modal.Host, o.Text)
		mail = fmt.Sprintf(TmplMail, "提交", o.WorkId, o.Username, modal.Host, modal.Host, "已提交")
		s.ToUser = user.Email
	}

	if t == 4 {
		ding = fmt.Sprintf(TmplFailedDing, o.WorkId, o.Username, o.Assigned, modal.Host, o.Text)
		mail = fmt.Sprintf(TmplMail, "执行", o.WorkId, o.Username, modal.Host, modal.Host, "执行失败")
	}

	if t == 5 {
		ding = fmt.Sprintf(TmplPerformDing, o.WorkId, o.Username, o.Assigned, modal.Host, o.Text)
		mail = fmt.Sprintf(TmplMail, "转交", o.WorkId, o.Username, modal.Host, modal.Host, "等待执行人执行")
	}

	if modal.GloOther.Query {
		var op modal.CoreQueryOrder
		modal.DB().Select("work_id,username,text,assigned").Where("work_id =?", workid).First(&op)
		modal.DB().Select("email").Where("username =?", op.Username).First(&user)
		s.ToUser = user.Email
		if t == 6 {
			modal.DB().Select("email").Where("username =?", op.Assigned).First(&user)
			s.ToUser = user.Email
			ding = fmt.Sprintf(TmplQueryRefer, op.WorkId, op.Username, op.Assigned, modal.Host, op.Text)
			mail = fmt.Sprintf(TmplMail, "查询申请", op.WorkId, op.Username, modal.Host, modal.Host, "已提交")
		}
		if t == 7 {
			ding = fmt.Sprintf(TmplSuccessQuery, op.WorkId, op.Username, op.Assigned, modal.Host)
			mail = fmt.Sprintf(TmplMail, "查询申请", op.WorkId, op.Username, modal.Host, modal.Host, "已同意")
		}
		if t == 8 {
			ding = fmt.Sprintf(TmplRejectQuery, op.WorkId, op.Username, op.Assigned, modal.Host)
			mail = fmt.Sprintf(TmplMail, "查询申请", op.WorkId, op.Username, modal.Host, modal.Host, "已驳回")
		}
	}

	if t == 9 || t == 10 || t == 11 {
		var op modal.CoreGroupOrder
		modal.DB().Select("work_id,username").Where("work_id =?", workid).First(&op)
		modal.DB().Select("email").Where("username =?", op.Username).First(&user)
		s.ToUser = user.Email
		if t == 9 {
			modal.DB().Select("email").Where("username =?", "admin").First(&user)
			s.ToUser = user.Email
			ding = fmt.Sprintf(TmplGroupRefer, op.WorkId, op.Username, "admin", modal.Host)
			mail = fmt.Sprintf(TmplMail, "权限申请", op.WorkId, op.Username, modal.Host, modal.Host, "已提交")
		}

		if t == 10 {
			ding = fmt.Sprintf(TmplSuccessGroup, op.WorkId, op.Username, "admin", modal.Host)
			mail = fmt.Sprintf(TmplMail, "权限申请", op.WorkId, op.Username, modal.Host, modal.Host, "已同意")
		}

		if t == 11 {
			ding = fmt.Sprintf(TmplRejectGroup, op.WorkId, op.Username, "admin", modal.Host)
			mail = fmt.Sprintf(TmplMail, "权限申请", op.WorkId, op.Username, modal.Host, modal.Host, "已驳回")
		}
	}

	if modal.GloMessage.Mail {
		if user.Email != "" {
			go SendMail(c, s, mail)
		}
	}
	if modal.GloMessage.Ding {
		if modal.GloMessage.WebHook != "" {
			go SendDingMsg(c, s, ding)

		}
	}
}

func TimerEx(order *modal.CoreSqlOrder) time.Duration {
	if order.Delay != "none" {
		now := time.Now()
		dt, _ := time.ParseInLocation("2006-01-02 15:04 ", order.Delay, time.Local)
		after := dt.Sub(now)
		if after+1 > 0 {
			return after
		}
	}
	return 0
}

func TimeDifference(t string) bool {
	dt, _ := time.ParseInLocation("2006-01-02 15:04 ", t, time.Local)
	f := dt.Sub(time.Now())
	if math.Abs(f.Minutes()) > float64(modal.GloOther.ExQueryTime) && float64(modal.GloOther.ExQueryTime) > 0 {
		return true
	}
	return false
}

type querydata struct {
	Field []map[string]string
	Data  []map[string]interface{}
}

func QueryMethod(source *modal.CoreDataSource, req *modal.Queryresults, wordList []string) (querydata, error) {

	var qd querydata

	ps := Decrypt(source.Password)

	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", source.Username, ps, source.IP, source.Port, req.Basename))
	if err != nil {
		return qd, err
	}

	defer db.Close()

	rows, err := db.Queryx(req.Sql)

	if err != nil {
		return qd, err
	}

	cols, err := rows.Columns()

	if err != nil {
		return qd, err
	}

	for rows.Next() {

		results := make(map[string]interface{})

		rows.MapScan(results)

		for idx := range results {
			switch r := results[idx].(type) {
			case []uint8:
				if len(r) > 10000 {
					results[idx] = "blob字段无法显示"
				} else {
					results[idx] = string(r)
				}
			}
		}

		if len(wordList) > 0 {
			for ok := range results {
				for _, exclude := range wordList {
					if ok == exclude {
						results[ok] = "****脱敏字段"
					}
				}
			}
		}

		qd.Data = append(qd.Data, results)
	}

	for _, cv := range cols {
		qd.Field = append(qd.Field, map[string]string{"title": cv, "key": cv, "width": "200"})
	}
	qd.Field[0]["fixed"] = "left"

	return qd, nil
}

func OrderDelayKill(w string) bool {
	var c modal.CoreSqlOrder
	modal.DB().Where("work_id =?", w).First(&c)
	if c.IsKill == 1 {
		return true
	}
	return false
}
