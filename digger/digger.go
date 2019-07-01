package digger

import(
	"github.com/BlackCarDriver/config"
	"fmt"
	"net/http"
	"sync"
	"math/rand"
	"time"
	"os"
	"strings"
	"container/list"
	"log"
	"syscall"
	"strconv"
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
	url_nextpage string
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
	mylist = list.New()
	randMachine = rand.New(ns)
	getNameMutex = &sync.Mutex{}
	updataSizeMutex = &sync.Mutex{}

	conf ,err := config.NewConfig(config_path)
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
	conf.Register("url_nextpage", "", false)
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
	url_nextpage, _  = conf.GetString("url_nextpage")
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
	case "test":
		digAndSaveImgs(url_seed)

	case "list":
		for _,url := range url_list{
			pagesNumber ++
			digAndSaveImgs(url)
		}

	case "bfd":	
		bfDig(url_seed)

	case "dfs":
		break

	case "forward":
		forwardDig()
	}

	//wait for a while
	shutdownsign <-syscall.Signal(2) 
	time.Sleep(time.Second * 60) 
}

// breadth first dig
func bfDig(seed string){
	mylist.PushBack(url_seed)
	for url:=mylist.Front(); url!=nil ; url=url.Next() {
		if 	url.Prev() != nil {
			mylist.Remove(url.Prev())
		}
		if pagesNumber >= max_pages_number{
			break
		}
		if goingToStop {
			break
		}
		turl := url.Value.(string)		//going to dig it url
		allATags := digAtags( turl )
		for _, atag := range allATags {
			//extrat url and check whether can be used
			href := getHref(atag, turl)
			if !canUse(href) {
				if href != ""{
					errLog.Println(href)
				}
				continue
			}
			//check which type it href is and do something
			if isNextPage(atag){
				mylist.PushBack(href)
				urlLog.Printf("%d ---> %s", mylist.Len(), href)
				continue
			}
			if isContain(atag) {
				digAndSaveImgs(href)
				pagesNumber ++
				time.Sleep(time.Second * time.Duration(sleep_time_s))
			}
		} 
	}
}

// specially use to dig some regular change url
func forwardDig(){
	basehref := `http://tieba.baidu.com/f?kw=%E6%A0%A1%E8%8A%B1&ie=utf-8&pn=`
		startIndx := 22050
		gap := 50
		endIndex :=  314400
		for i := startIndx; i <= endIndex; i += gap {
			//tmpUrl := fmt.Sprintf(basehref, i)
			tmpUrl := basehref + strconv.Itoa(i)
			digAndSaveImgs(tmpUrl)
			pagesNumber ++
			if goingToStop {
				break
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
	fmt.Println("The program is stop down safely ....")
	time.Sleep(time.Second)
	os.Exit(1)
}

