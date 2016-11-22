package main

import(
	"menteslibres.net/gosexy/redis"
	"log"
	"os"
  	"strconv"
  	"net/http"
  	"io/ioutil"
  	"crypto/tls"
  	"net/url"
  	"net/http/cookiejar"
  	"strings"
  	"net/smtp"
  	"net/mail"
  	"fmt"
  	"net"
)

const(
	email_user_key = "ticket_reminder_email_user_key"
	email_password_key = "ticket_reminder_email_password_key"
	email_to_key="ticket_reminder_email_to_key"
	email_host_key = "ticket_reminder_email_host_key"
	email_subject = "动车票购买通知"
)


var client *redis.Client
var cJar *cookiejar.Jar

func init() {
	client = redis.New()
	b,err:=strconv.Atoi(os.Args[2])
	checkErr(err)
	err= client.Connect(os.Args[1],uint(b))
	checkErr(err)
	if len(os.Args)>3{
		client.Auth(os.Args[3])
	}
	init_cookie()
	
}

func init_cookie() {
	cJar,_= cookiejar.New(nil)
	var cookies []*http.Cookie
	cookie := &http.Cookie{
		Name:   "_jc_save_fromDate",
		Value:  "2116-12-31",
		Path:   "/",
		Domain: ".12306.cn",
	}
	cookies = append(cookies, cookie)
	cookie = &http.Cookie{
		Name:   "_jc_save_fromStation",
		Value:  "%u676D%u5DDE%u4E1C%2CHGH",
		Path:   "/",
		Domain: ".12306.cn",
	}
	cookies = append(cookies, cookie)
	cookie = &http.Cookie{
		Name:   "_jc_save_toStation",
		Value:  "%u5B81%u6D77%2CNHH",
		Path:   "/",
		Domain: ".12306.cn",
	}
	cookies = append(cookies, cookie)
	cookie = &http.Cookie{
		Name:   "_jc_save_wfdc_flag",
		Value:  "dc",
		Path:   "/",
		Domain: ".12306.cn",
	}

	cookies = append(cookies, cookie)
	u, _ := url.Parse("https://kyfw.12306.cn/otn/lcxxcx/query?purpose_codes=ADULT&queryDate=2016-12-30&from_station=HGH&to_station=NHH")
	cJar.SetCookies(u, cookies)
}
func getHtml() string{
	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
	client := &http.Client{Transport: tr,Jar: cJar}
    resp, err := client.Get("https://kyfw.12306.cn/otn/lcxxcx/query?purpose_codes=ADULT&queryDate=2016-12-30&from_station=HGH&to_station=NHH")
    checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)
	return string(body)
}

func main() {
	defer client.Quit()
	html:=getHtml()
	date:=getDate(html)
	email_user,err:=client.Get(email_user_key)
	checkErr(err)
	email_password,err:=client.Get(email_password_key)
	checkErr(err)
	email_host,err:=client.Get(email_host_key)
	checkErr(err)
	email_to,err:=client.LRange(email_to_key,0,-1)
	checkErr(err)
	body:="今天最新可买到"+date+"的动车票，快行动吧!"
	SendToMail(email_user,email_password,email_host,email_to,email_subject,body,"")
	log.Println("finished")
}
func getDate(html string) string{
	key:="\"note\":\"暂售至<br/>"
	index:=strings.Index(html,key)
	return html[index+len(key):index+len(key)+10]
}

func SendToMail(fromAddr string, password string, servername string, toAddrs []string, subj string, body string, mailtype string) {
	from := mail.Address{"", fromAddr}
    // Setup headers
    headers := make(map[string]string)
    headers["From"] = from.String()
    
    headers["Subject"] = subj
    // Setup message



    // Connect to the SMTP Server

    host, _, _ := net.SplitHostPort(servername)

    auth := smtp.PlainAuth("",fromAddr, password, host)

    // TLS config
    tlsconfig := &tls.Config {
        InsecureSkipVerify: true,
        ServerName: host,
    }
    // Here is the key, you need to call tls.Dial instead of smtp.Dial
    // for smtp servers running on 465 that require an ssl connection
    // from the very beginning (no starttls)
    conn, err := tls.Dial("tcp", servername, tlsconfig)
    checkErr(err)
    
    c, err := smtp.NewClient(conn, host)
    checkErr(err)

    // Auth
    if err = c.Auth(auth); err != nil {
        checkErr(err)
    }

    // To && From
    if err = c.Mail(from.Address); err != nil {
        checkErr(err)
    }
    for _,toAddr :=range toAddrs{
    	to   := mail.Address{"", toAddr}
	    headers["To"] = to.String()
	    message := ""
	    for k,v := range headers {
	        message += fmt.Sprintf("%s: %s\r\n", k, v)
	    }
	    message += "\r\n" + body
	    log.Println(message)
	    if err = c.Rcpt(to.Address); err != nil {
	        checkErr(err)
	    }
	    // Data
	    w, err := c.Data()
	    checkErr(err)

	    _, err = w.Write([]byte(message))
	    checkErr(err)

	    err = w.Close()
	    checkErr(err)
    }

    c.Quit()
}


func checkErr(err error){
	if(err!=nil){
		panic(err)
	}
}