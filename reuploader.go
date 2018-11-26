package main

import (
	"time"
	"os"
	"log"
	"io/ioutil"
	"./ljapi"
	"./imgurapi"
	"./sender"
	"fmt"
	"net/url"
	"net/http"
	"encoding/json"
	"strings"
	"strconv"
	"path"
	"errors"
)

var imgur imgurapi.ImgurClient = imgurapi.ImgurClient{Locked: false, ClientID: "PRIVATE DATA", ClientSecret: "PRIVATE DATA"}

type task struct {
	LJ ljapi.LJClient	`json:"lj_client"`
	Email string		`json:"email"`
	Links []string		`json:"links"`
	Rules []string		`json:"rules"`
	Filename string
}

type image struct {
	URL, Domain string
	Size int
}

func (i *image) GetImageInfo() error {
	u, err := url.Parse(i.URL)
	if (err != nil) {
		return err
	}
	i.Domain = u.Host
	if (err != nil) {
		return err
	}
	head, err := http.Head(i.URL)
	if (err != nil) {
		return err
	}
	if (head.StatusCode != http.StatusOK) {
		return errors.New("Unknown error")
	}
	i.Size, err = strconv.Atoi(head.Header.Get("Content-Length"));
	if (err != nil) {
		return err
	}
	return nil
}

func (i *image) CheckImage(rules []string) bool {
	var rules_map map[string][]string = make(map[string][]string)
	for _, rule := range rules {
		entry := strings.Split(rule, " ")
		rules_map[entry[0]] = entry[1:]
	}
	if (rules_map["MORETHAN"] != nil) {
		if size, err := strconv.Atoi(rules_map["MORETHAN"][0]); (err != nil || i.Size <= size) {
			return false
		}
	}
	if (rules_map["LESSTHAN"] != nil) {
		if size, err := strconv.Atoi(rules_map["LESSTHAN"][0]); (err != nil || i.Size <= size) {
			return false
		}
	}
	if (rules_map["EXCLUDE"] != nil) {
		for _, domain := range rules_map["EXCLUDE"] {
			if (i.Domain == domain) {
				return false
			}
		}
	}
	if (rules_map["INCLUDE"] != nil) {
		var found bool = false
		for _, domain := range rules_map["INCLUDE"] {
			if (domain == "*" || i.Domain == domain) {
				found = true
				break
			}
		}
		if (!found) {
			return false
		}
	}
	return true
}

type reporter struct {
	File *os.File
}

func (r *reporter) Begin() {
	r.File, _ = os.Create("report/report.txt")
}

func (r *reporter) Add(msg string) {
	fmt.Fprintf(r.File, "[%s] > %s", time.Now().Format("15:04:05"), msg)
}

func (r *reporter) Finish() {
	r.File.Close();
}

var main_report reporter

func loadTask(filename string) task {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Failed to load task %s", filename)
		log.Print(err)
		os.Remove(filename)
		return task{}
	}
	var result task
	err = json.Unmarshal(content, &result)
	if err != nil {
		log.Printf("Failed to parse task %s", filename)
		log.Print(err)
		os.Remove(filename)
		return task{}
	}
	result.Filename = filename
	log.Printf("Loaded task %s", filename)
	return result
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func processPost(post ljapi.LJPost, rules []string) (ljapi.LJPost, error) {
	var tokens = [...]string {"<", "img", "src=\"", "\""}
	var token_index int = 0
	var url_begin, url_end int = 0,0
	
	var edited_content string
	edited_content = post.Content
	
	for i := 0; i < len(post.Content); i++ {
		if post.Content[i] == '>' {
			token_index = 0
			url_begin = 0
			url_end = 0
			continue
		}
		possible_token := post.Content[i*boolToInt(i + len(tokens[token_index]) < len(post.Content)) : (i + len(tokens[token_index]))*boolToInt(i + len(tokens[token_index]) < len(post.Content))]
		if possible_token == tokens[token_index] {
			if token_index == 2 {
				url_begin = i + 5
			}
			if token_index == 3 {
				url_end = i - 1
			}
			i += len(tokens[token_index]) - 1
			token_index++
			if token_index >= len(tokens) {
				image_url := post.Content[url_begin:url_end+1]
				img := image{URL: image_url}
				err := img.GetImageInfo()
				if err != nil {
					log.Printf("%s : error : %s", image_url, err)
					main_report.Add(fmt.Sprintf("%s : error : %s\n", image_url, err))
				}
				if img.CheckImage(rules) {
					if imgur.Locked {
						log.Printf("Imgur is locked, waiting %d seconds", imgur.ResetTime)
						time.Sleep(time.Duration(int64(imgur.ResetTime + 1) * 1000000000))
					}
					imgur.Locked = false
					new_image_url, err := imgur.UploadImage(image_url)
					if err == nil {
						edited_content = strings.Replace(edited_content, image_url, new_image_url, -1)
						log.Printf("%s -> %s", image_url, new_image_url)
						main_report.Add(fmt.Sprintf("%s -> %s\n", image_url, new_image_url))
					} else {
						log.Printf("%s : error : %s", image_url, err)
						main_report.Add(fmt.Sprintf("%s : error : %s\n", image_url, err))
					}
				} else {
					log.Printf("Skipped %s due to rules", image_url)
					main_report.Add(fmt.Sprintf("Skipped %s due to rules\n", image_url))
				}
				token_index = 0
				url_begin = 0
				url_end = 0
			}
		}
	}
	
	post.Content = edited_content
	return post, nil
}

func backupPost(link string, post ljapi.LJPost) error {
	_, filename := path.Split(link)

	f, err := os.Create("report/" + filename + ".txt")
	defer f.Close()	
	if err != nil {
		return err
	}
	fmt.Fprint(f, post.Header, "\n\n\n", post.Content)
	f.Close()

	post.Content = url.PathEscape(post.Content)
	post.Header = url.PathEscape(post.Header)

	buf, err := json.Marshal(post)
	if err != nil {
		return err
	}

	f, err = os.Create("report/" + filename + ".json")
	if err != nil {
		return err
	}
	fmt.Fprint(f, string(buf))
	return nil
}

func initReportDir() {
	os.RemoveAll("report/")
	os.Mkdir("report/", 0777)
}

func executeTask(subject task) {
	initReportDir()
	main_report.Begin()
	defer main_report.Finish()
	main_report.Add(fmt.Sprintf("Started executing task for %s\n", subject.LJ.User))
	for _, link := range subject.Links {
		main_report.Add(fmt.Sprintf("Started reuploading for post %s\n", link))
		post, err := subject.LJ.GetPost(link)
		if err != nil {
			log.Printf("Failed to get post %s", link)
			main_report.Add(fmt.Sprintf("Failed to get post %s\n", link))
			log.Print(err)
			continue
		}
		err = backupPost(link, post)
		if err != nil {
			log.Printf("Failed to backup post %s", link)
			main_report.Add(fmt.Sprintf("Failed to backup post %s\n", link))
			log.Print(err)
			continue
		}
		post, err = processPost(post, subject.Rules)
		if err != nil {
			log.Printf("Failed to process post %s", link)
			main_report.Add(fmt.Sprintf("Failed to process post %s\n", link))
			log.Print(err)
			continue
		}
		err = subject.LJ.EditPost(post)
		if err == nil {
			log.Printf("%s : done", link)
			main_report.Add(fmt.Sprintf("%s : done\n", link))
		} else {
			log.Printf("%s : error : %s", link, err)
			main_report.Add(fmt.Sprintf("%s : error : %s\n", link, err))
		}
	}
	if !imgur.Locked {
		err := sender.SendReport(subject.Email, subject.LJ.User)
		if err != nil {
			log.Print(err)
		} else {
			log.Printf("Successfuly sent email to %s", subject.Email)
		}
		main_report.Finish()
		os.Remove(subject.Filename)
	}
}

func main() {
	initReportDir()
	var check_id int = -1
	for true {
		if imgur.Locked {
			log.Printf("Imgur is locked, waiting %d seconds", imgur.ResetTime)
			time.Sleep(time.Duration(int64(imgur.ResetTime + 1) * 1000000000))
		}
		imgur.Locked = false
		time.Sleep(5 * time.Second)
		check_id++
		tasks, err := ioutil.ReadDir("tasks/")
		if err != nil {
			log.Printf("Check #%d: Failed to check tasks", check_id)
			log.Print(err)
			continue
		}
		if len(tasks) == 0 {
			log.Printf("Check #%d: No tasks were found", check_id)
			continue
		}
		executeTask(loadTask("tasks/" + tasks[0].Name()))
		log.Printf("Imgur reset time: %d", imgur.ResetTime)
	}
}