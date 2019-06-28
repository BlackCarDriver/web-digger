package digger

import(
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"math/rand"
	"time"
	"os"
	"strings"
	"container/list"
	"log"
)

const(
	config_path = "./config/"
)

//galbol values
var (
	randMachine *rand.Rand
	getNameMutex  *sync.Mutex
	updataSizeMutex *sync.Mutex
	mainClient		http.Client
	shutdownsign	chan os.Signal
	urlLog			*log.Logger
	errLog			*log.Logger
	url_list		[]string	
	goingToStop		bool
	imgNumbers int		//how many images already download
	pagesNumber int		//how many pages already visit
	totalImgbytes uint64		//the total size of all images already download
	mylist	*list.List
	url_map		map[string]bool	//record all url have read from html
)

//config values
var (
	source_path string
	log_path	string
	thread_numbers int
	url_seed	string
	min_img_kb int
	max_img_mb int
	max_occupy_mb int
	max_pages_number int
	travel_method string
	url_prefix	string
	url_must_contain string
	max_wait_time_s int
	sleep_time_s	int
)

func init(){
	imgNumbers = 0
	totalImgbytes = 0
	pagesNumber = 0
	goingToStop = false
	shutdownsign = make(chan os.Signal, 10)
	url_map = make(map[string]bool)
	ns := rand.NewSource(time.Now().UnixNano())
	randMachine = rand.New(ns)
	getNameMutex = &sync.Mutex{}
	updataSizeMutex = &sync.Mutex{}

	conf ,err := NewConfig(config_path)
	if err != nil {
		panic(err)
	} 
	conf.Register("source_path", "", false)
	conf.Register("log_path", "", false)
	conf.Register("url_seed", "https://tieba.baidu.com", false )
	conf.Register("thread_numbers", 1, false)
	conf.Register("min_img_kb", 1, false)
	conf.Register("max_img_mb", 10, false)
	conf.Register("max_occupy_mb", 1000, false)
	conf.Register("travel_method", "bfs", false)
	conf.Register("max_pages_number", 50, false)
	conf.Register("url_list", make([]string,0), false)
	conf.Register("url_prefix", "", false)
	conf.Register("url_must_contain", "", false)
	conf.Register("max_wait_time_s", 10, false)
	conf.Register("sleep_time_s", 0, false)
	source_path, _ = conf.GetString("source_path")
	log_path, _ = conf.GetString("log_path")
	thread_numbers,_ = conf.GetInt("thread_numbers")
	url_seed,_ = conf.GetString("url_seed")
	min_img_kb, _ = conf.GetInt("min_img_kb")
	max_img_mb, _ = conf.GetInt("max_img_mb")
	max_occupy_mb, _ = conf.GetInt("max_occupy_mb")
	travel_method, _ = conf.GetString("travel_method")
	max_pages_number, _ = conf.GetInt("max_pages_number")
	url_list, _ = conf.GetStrings("url_list")
	url_prefix, _ = conf.GetString("url_prefix")
	url_must_contain, _ = conf.GetString("url_must_contain")
	max_wait_time_s, _ = conf.GetInt("max_wait_time_s")
	sleep_time_s,_ = conf.GetInt("sleep_time_s")

	//conf.display()

	mainClient = http.Client{
		Timeout : time.Second * time.Duration(max_wait_time_s),
	}

	//init logger
	log_path = strings.TrimRight(log_path, `\`)
	logfp, err := os.Create(log_path + `\urlQueue.log`)
	if err!=nil {
		panic(err)
	}
	logfp2, _ := os.Create(log_path + `\error.log`)
	urlLog = log.New(logfp, "", 0)
	errLog = log.New(logfp2, "", 0)
}


func Test(){
	go destructor()
	switch travel_method {
	case "list":
		for _,url := range url_list{
			DigUrl(url)
		}

	case "bfs":
		mylist = list.New()
		mylist.PushBack(url_seed)
		for url:=mylist.Front(); url!=nil ; url=url.Next() {
			urlstr := url.Value.(string)
			if pagesNumber >= max_pages_number{
				break
			}
			if goingToStop {
				break
			}
			pagesNumber ++
			DigUrl(urlstr)
			time.Sleep(time.Second * time.Duration(sleep_time_s))
		}

	case "dfs":
		DigUrl(url_seed)
	}
}

//visit an url and do somthing through the html text
func DigUrl(targetUrl string) error{
	fmt.Printf("url         [  %s  ]\n", targetUrl)
	resp, err := mainClient.Get(targetUrl)
	if err != nil {
		fmt.Println(targetUrl, "----------->" ,err)
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	html := string(body)
	

	//extract href from html and push then into map and queue
	aReg,_ := regexp.Compile(`<a [^>]*>`)
	aTags := aReg.FindAllString(html, -1)
	fmt.Printf("<a>         [  %-6d  ]\n", len(aTags))
	useLink := 0
	for _,j := range aTags {
		aurl := getHref(j, targetUrl)
		if !canUse(aurl) {
			errLog.Println(aurl)
			continue
		}
		urlLog.Printf("%d ---> %s", mylist.Len(), aurl)
		mylist.PushBack(aurl)
		useLink ++
	}
	fmt.Printf("<a>+        [  %-6d  ]\n", useLink)
	
	//get all img url from html code and colloct into a slice
	reg1, _ := regexp.Compile(`<img [^>]*>`) 
	imgTags := reg1.FindAllString(html, -1)
	imgSlice := make([]string,0)
	for _,j := range imgTags {
		imgSlice = append(imgSlice, getImgSlice(j, targetUrl)...) 
	}
	fmt.Printf("<img>       [  %-6d  ]\n", len(imgSlice))
	
	//create some goroutine and distribute the workes
	if len(imgSlice) == 0 {
		return nil
	}
	urlChan := make(chan string, 100)
	resChan := make(chan int, 20)
	for i:=0; i<thread_numbers; i++ {
		go imgDownLoader(i, urlChan, resChan)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go showResult(len(imgSlice), resChan, &wg)
	for _,j := range imgSlice {
		urlChan <- j
	}

	//wait for images download
	wg.Wait()
	close(urlChan)
	close(resChan)
	return nil
}

//display the result of work goroutine
func showResult(times int, res <-chan int, wg *sync.WaitGroup){
	counter := 0
	for tres :=  range res {
		counter ++
		fmt.Print(tres," ")
		if counter == times {
			fmt.Printf("\n urlQue length now is %d , it is the %d pages \n\n\n", mylist.Len(), pagesNumber )
			wg.Done()
			return
		}
		if (counter%50) == 0 {
			fmt.Println()
		}
	}
}

//giving a little time to downloaders before shut down the program 
func destructor(){
	<-shutdownsign
	goingToStop = true
	go func(){
		for {
			<-shutdownsign
		}
	}()
	time.Sleep(time.Second * 10)
	os.Exit(1)
}

