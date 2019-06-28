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
	"container/list"
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
	fmt.Printf("Begin to analyze %s :     ", targetUrl)
	//fmt.Println(targetUrl)
	resp, err := mainClient.Get(targetUrl)
	if err != nil {
		fmt.Println(targetUrl, "----------->" ,err)
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	html := string(body)
	//fmt.Println(html)
	//return nil
	//extract href from html and push then into map and queue
	aReg,_ := regexp.Compile(`<a [^>]*>`)
	aTags := aReg.FindAllString(html, -1)
	for _,j := range aTags {
		aurl := getHref(j, targetUrl)
		//fmt.Println("####### ", aurl)
		if !canUse(aurl) {
			continue
		}
		//fmt.Println("inqueue  ------------>  ", aurl)
		mylist.PushBack(aurl)
	}
	
	//return nil
	//get all img url from html code and colloct into a slice
	reg1, _ := regexp.Compile(`<img [^>]*>`) 
	imgTags := reg1.FindAllString(html, -1)
	imgSlice := make([]string,0)
	for _,j := range imgTags {
		imgSlice = append(imgSlice, getImgSlice(j, targetUrl)...) 
	}

	//create some imgages download workers
	fmt.Println("   images numbers: ", len(imgSlice))
	if len(imgSlice) == 0 {
		return nil
	}
	urlChan := make(chan string, 100)
	resChan := make(chan int, 20)
	for i:=0; i<thread_numbers; i++ {
		go imgDownLoader(i, urlChan, resChan)
	}
	
	//distribute the work to downloaders
	var wg sync.WaitGroup
	wg.Add(1)
	go showResult(len(imgSlice), resChan, &wg)
	for _,j := range imgSlice {
		urlChan <- j
	}

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
		fmt.Print(tres)
		if counter == times {
			fmt.Println(" \n the work is complete !")
			wg.Done()
			return
		}
		if (counter%100) == 0 {
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

