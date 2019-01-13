package main

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"io"
	"os"
	"strings"
	"strconv"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"time"
	"math/rand"
	"./ljapi"
	"syscall"
)

type settings struct {
	CertFile	string	`json:"site_cert"`
	KeyFile		string	`json:"site_key"`
	UseTLS 		bool		`json:"site_tls"`
	GroupID		int			`json:"gid"`
	UserID 		int			`json:"uid"`
}

var conf settings = settings {
	CertFile: "",
	KeyFile: "",
	UseTLS: false,
	GroupID: os.Getgid(),
	UserID: os.Getuid(),
}

func loadConfig(filename string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Print("Failed to read config file. Using default settings. ")
		log.Print(err)
		return
	}
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Print("Failed to parse config file. Using default settings. ")
		log.Print(err)
		return
	}
	log.Print("Config file successfuly loaded.")
}

func loadPage(response http.ResponseWriter, filename string) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		log.Print(err)
		f, err = os.Open("pages/500.html")
		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	io.Copy(response, f)
}

func loadStyleSheet(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "text/css; charset=utf-8")
        f, err := os.Open("pages/style.css")
		defer f.Close()
        if err != nil {
				log.Print(err)
                loadPage(response, "pages/500.html")
                return
        }
        io.Copy(response, f)
}

func loadOptionsPage(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := request.ParseForm()
	if err != nil {
		loadPage(response, "pages/500.html")
		return
	}
	user := request.Form.Get("user")
	password := request.Form.Get("password")
	email := request.Form.Get("email")

	buf := md5.Sum([]byte(password))
	passhash := hex.EncodeToString(buf[:])

	lj := ljapi.LJClient{User: user, PassHash: passhash}
	ok, err := lj.TryLogIn()
	if err != nil {
		log.Print(err)
		loadPage(response, "pages/500.html")
		return
	}
	if !ok {
		log.Print("loadOptionsPage(): wrong password")
		loadPage(response, "pages/403.html")
		return
	}

	if ((user == "") || (password == "") || (email == "")) {
		loadPage(response, "pages/400.html")
		return
	}
	content, err := ioutil.ReadFile("pages/options.html")
	if err != nil {
		loadPage(response, "pages/500.html")
		return
	}
	var str_content string = string(content)
	fmt.Fprintf(response, str_content, user, password, email, email)
	log.Print("loadOptionsPage(): password OK")
}

func getNonce() string {
	const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var res string = ""
	for i := 0; i < 8; i++ {
		res = res + string(ALPHABET[rand.Intn(len(ALPHABET))])
	}
	return res
}

func registerReuploadQuery(response http.ResponseWriter, request *http.Request) {
	type reuploadQuery struct {
		LJ		ljapi.LJClient	`json:"lj_client"`
		Email string					`json:"email"`
		Links []string				`json:"links"`
		Rules []string				`json:"rules"`
	}

	var taskfile string = strconv.Itoa(int(time.Now().Unix())) + "-" + getNonce()
	taskfile = "tasks/"+taskfile
	f, err := os.Create(taskfile)
	defer f.Close()
	err = request.ParseForm()
	if err != nil {
		log.Print(err)
		loadPage(response, "pages/500.html")
		return
	}
	lj_user := request.Form.Get("user")
	buf := md5.Sum([]byte(request.Form.Get("password")))
	lj_passhash := hex.EncodeToString(buf[:])
	email := request.Form.Get("email")
	links := strings.Split(request.Form.Get("links"), "\r\n")
	rules := strings.Split(request.Form.Get("rules"), "\r\n")
	if ((lj_user == "") || (email == "") || (len(links) == 0) || (len(rules) == 0)) {
		loadPage(response, "pages/400.html")
		return
	}
	query := reuploadQuery{
		LJ: ljapi.LJClient{User: lj_user, PassHash: lj_passhash},
		Email: email,
		Links: links,
		Rules: rules,
	}
	js_bytes, err := json.Marshal(query)
	if err != nil {
		log.Print(err)
		loadPage(response, "pages/500.html")
		return
	}
	fmt.Fprint(f, string(js_bytes))
	f.Close()
	os.Chown(taskfile, conf.UserID, conf.GroupID)
	os.Chmod(taskfile, 0660)
	loadPage(response, "pages/reupload.html")
	log.Printf("Registered a reupload query. Task file: %s", taskfile)
}

func loadFavicon(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "image/x-icon")
	f, err := os.Open("pages/favicon.ico")
	defer f.Close()
	if err != nil {
		response.WriteHeader(http.StatusNotFound)
		return
	}
	io.Copy(response, f)
}

func handler(response http.ResponseWriter, request *http.Request) {
	var url string = request.URL.Path
	log.Printf("Request to %s from %s", url, request.RemoteAddr)
	switch url {
		case "/400": loadPage(response, "pages/400.html")
		case "/403": loadPage(response, "pages/403.html")
		case "/404": loadPage(response, "pages/404.html")
		case "/500": loadPage(response, "pages/500.html")
		case "/": loadPage(response, "pages/welcome.html")
		case "/lj_auth": loadPage(response, "pages/lj_auth.html")
		case "/rules": loadPage(response, "pages/rules.html")
		case "/style.css": loadStyleSheet(response)
		case "/reupload": registerReuploadQuery(response, request)
		case "/options": loadOptionsPage(response, request)
		case "/favicon.ico": loadFavicon(response)
		default: loadPage(response, "pages/404.html")
	}
}

func main() {
	log.Print("LJIR Online Front-End")

	loadConfig("ljir.conf")

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	rand.Seed(int64(time.Now().Unix()))

	os.RemoveAll("tasks/")
	os.Mkdir("tasks/", 0770)
	os.Chown("tasks/", conf.UserID, conf.GroupID)

	http.HandleFunc("/", handler)
	if conf.UseTLS {
		log.Fatal(http.ListenAndServeTLS(":443", conf.CertFile, conf.KeyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(":80", nil))
	}
}
