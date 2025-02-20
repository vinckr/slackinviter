package main

import (
	"bytes"
	"context"
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-recaptcha/recaptcha"
	"github.com/gorilla/handlers"
	"github.com/kelseyhightower/envconfig"
	badge "github.com/narqo/go-badge"
	"github.com/nlopes/slack"
	"github.com/paulbellamy/ratecounter"
)

var indexTemplate = template.Must(template.New("index.tmpl").ParseFiles("templates/index.tmpl"))
var redirectTemplate = template.Must(template.New("redirect.tmpl").ParseFiles("templates/redirect.tmpl"))

var (
	api     *slack.Client
	captcha *recaptcha.Recaptcha
	counter *ratecounter.RateCounter

	ourTeam = new(team)

	m *expvar.Map
	hitsPerMinute,
	requests,
	inviteErrors,
	missingFirstName,
	missingLastName,
	missingEmail,
	missingCoC,
	successfulCaptcha,
	failedCaptcha,
	invalidCaptcha,
	successfulInvites,
	userCount,
	activeUserCount expvar.Int
)

var c Specification

// wrapper for Session Data
type SessionResponse struct {
	SessionData SessionData `json:"sessionData"`
}

type SessionData struct {
	Identity struct {
		Traits struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"traits"`
	} `json:"identity"`
}

// Specification is the config struct
type Specification struct {
	Port           string        `envconfig:"PORT" required:"true"`
	CaptchaSitekey string        `required:"true"`
	CaptchaSecret  string        `required:"true"`
	SlackToken     string        `required:"true"`
	CocUrl         string        `required:"false" default:"http://coc.golangbridge.org/"`
	SessionData    []SessionData `json:"sessionData"`
	EnforceHTTPS   bool
	Debug          bool // toggles nlopes/slack client's debug flag
}

type contextKey string

func init() {
	var showUsage = flag.Bool("h", false, "Show usage")
	flag.Parse()

	if *showUsage {
		err := envconfig.Usage("slackinviter", &c)
		if err != nil {
			log.Fatal(err.Error())
		}
		os.Exit(0)
	}

	err := envconfig.Process("slackinviter", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	counter = ratecounter.NewRateCounter(1 * time.Minute)
	m = expvar.NewMap("metrics")
	m.Set("hits_per_minute", &hitsPerMinute)
	m.Set("requests", &requests)
	m.Set("invite_errors", &inviteErrors)
	m.Set("missing_first_name", &missingFirstName)
	m.Set("missing_last_name", &missingLastName)
	m.Set("missing_email", &missingEmail)
	m.Set("missing_coc", &missingCoC)
	m.Set("failed_captcha", &failedCaptcha)
	m.Set("invalid_captcha", &invalidCaptcha)
	m.Set("successful_captcha", &successfulCaptcha)
	m.Set("successful_invites", &successfulInvites)
	m.Set("active_user_count", &activeUserCount)
	m.Set("user_count", &userCount)

	captcha = recaptcha.New(c.CaptchaSecret)
	api = slack.New(c.SlackToken, slack.OptionDebug(c.Debug))
}

func handleBadge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	users := userCount.String()
	if activeUserCount.Value() > 0 {
		users = activeUserCount.String() + "/" + userCount.String()
	}

	var buf bytes.Buffer
	if err := badge.Render("slack", users, "#E01563", &buf); err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	buf.WriteTo(w)
}

func main() {
	go pollSlack()
	mux := http.NewServeMux()
	mux.HandleFunc("/invite/", handleInvite)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", enforceHTTPSFunc(redirectPage))
	mux.HandleFunc("/badge.svg", handleBadge)
	mux.Handle("/debug/vars", http.DefaultServeMux)
	mux.HandleFunc("/invitation", handleSession)
	err := http.ListenAndServe(":"+c.Port, handlers.CombinedLoggingHandler(os.Stdout, mux))
	if err != nil {
		log.Fatal(err.Error())
	}
}

func enforceHTTPSFunc(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if xfp := r.Header.Get("X-Forwarded-Proto"); c.EnforceHTTPS && xfp == "http" {
			u := *r.URL
			u.Scheme = "https"
			if u.Host == "" {
				u.Host = r.Host
			}
			http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			return
		}
		h(w, r)
	}
}

// Updates the globals from the slack API
// returns the length of time to sleep before the function
// should be called again
func updateFromSlack() time.Duration {
	var (
		err            error
		p              slack.UserPagination
		uCount, aCount int64 // users and active users
	)

	ctx := context.Background()
	for p = api.GetUsersPaginated(
		slack.GetUsersOptionPresence(true),
		slack.GetUsersOptionLimit(500),
	); !p.Done(err); p, err = p.Next(ctx) {
		if err != nil {
			if rle, ok := err.(*slack.RateLimitedError); ok {
				fmt.Printf("Being Rate Limited by Slack: %s\n", rle)
				time.Sleep(rle.RetryAfter)
				continue
			}
		}
		for _, u := range p.Users {
			if u.ID != "USLACKBOT" && !u.IsBot && !u.Deleted {
				uCount++
				if u.Presence == "active" {
					aCount++
				}
			}
		}
		fmt.Println("User Count:", uCount)
		fmt.Println("Active Count:", aCount)
	}
	userCount.Set(uCount)
	activeUserCount.Set(aCount)
	if err != nil && !p.Done(err) {
		log.Println("error polling slack for users:", err)
		return time.Minute
	}

	st, err := api.GetTeamInfo()
	if err != nil {
		log.Println("error polling slack for team info:", err)
		return time.Minute
	}
	ourTeam.Update(st)
	return time.Hour
}

// pollSlack over and over again
func pollSlack() {
	for {
		time.Sleep(updateFromSlack())
	}
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	var sessionData SessionData

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body",
			http.StatusInternalServerError)
	}
	// convert body to a string and trim the leading "sessionData="
	bodyString := strings.TrimPrefix(string(body), "sessionData=")
	// Decode the URL-encoded string
	decodedString, err := url.QueryUnescape(bodyString)
	if err != nil {
		log.Println("Error decoding URL-encoded string:", err)
		return
	}
	// Unmarshal the JSON-encoded decodedString into sessionData
	err = json.Unmarshal(([]byte(decodedString)), &sessionData)
	if err != nil {
		log.Println("Error unmarshalling JSON into sessionData:", err)
		return
	}
	// set other template variables
	counter.Incr(1)
	hitsPerMinute.Set(counter.Rate())
	requests.Add(1)
	// Render the index template with sessionData
	var buf bytes.Buffer
	errRender := indexTemplate.Execute(
		&buf,
		struct {
			SiteKey,
			UserCount,
			ActiveCount string
			Team        *team
			CocUrl      string
			SessionData SessionData
		}{
			c.CaptchaSitekey,
			userCount.String(),
			activeUserCount.String(),
			ourTeam,
			c.CocUrl,
			sessionData,
		},
	)
	if errRender != nil {
		log.Println("error rendering template:", err)
		http.Error(w, "error rendering template :-(", http.StatusInternalServerError)
		return
	}

	// Set the header and write the buffer to the http.ResponseWriter
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// redirectPage renders the redirect page
func redirectPage(w http.ResponseWriter, r *http.Request) {
	counter.Incr(1)
	hitsPerMinute.Set(counter.Rate())
	requests.Add(1)
	redirectTemplate.Execute(w, nil)
}

// ShowPost renders a single post
func handleInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	successfulCaptcha.Add(1)
	fname := r.FormValue("fname")
	lname := r.FormValue("lname")
	email := r.FormValue("email")
	coc := r.FormValue("coc")
	if email == "" {
		missingEmail.Add(1)
		http.Error(w, "Missing email", http.StatusPreconditionFailed)
		return
	}
	if fname == "" {
		missingFirstName.Add(1)
		http.Error(w, "Missing first name", http.StatusPreconditionFailed)
		return
	}
	if lname == "" {
		missingLastName.Add(1)
		http.Error(w, "Missing last name", http.StatusPreconditionFailed)
		return
	}
	if coc != "1" {
		missingCoC.Add(1)
		http.Error(w, "You need to accept the code of conduct", http.StatusPreconditionFailed)
		return
	}
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		failedCaptcha.Add(1)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	captchaResponse := r.FormValue("g-recaptcha-response")
	valid, err := captcha.Verify(captchaResponse, remoteIP)
	if err != nil {
		failedCaptcha.Add(1)
		http.Error(w, "Error validating recaptcha.. Did you click it?", http.StatusPreconditionFailed)
		return
	}
	if !valid {
		invalidCaptcha.Add(1)
		http.Error(w, "Invalid recaptcha", http.StatusInternalServerError)
		return

	}
	// all is well, let's try to invite someone!
	err = api.InviteToTeam(ourTeam.Domain(), fname, lname, email)
	if err != nil {
		log.Println("InviteToTeam error:", err)
		inviteErrors.Add(1)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	successfulInvites.Add(1)
}
