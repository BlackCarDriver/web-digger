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
	"syscall"
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
		basehref := `http://pic.netbian.com/tupian/%d.html`
		startIndx := 14335
		endIndex :=  24335
		for i:= startIndx; i<=endIndex; i++ {
			tmpUrl := fmt.Sprintf(basehref, i)
			digAndSaveImgs(tmpUrl)
		}  
	}

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


//judge if a <a/> element have substring url_nextpage
func isNextPage(atag string) bool {
	if url_nextpage == "" {
		return false
	}
	if strings.Index(atag, url_nextpage) < 0  {
		return false
	}
	return true
}
//judge if a <a/> element have substring url_must_contain
func isContain(atag string) bool {
	if url_must_contain == "" {
		return true
	}
	if strings.Index(atag, url_must_contain) < 0  {
		return false
	}
	return true
}

//visit an url and get the html code
func digHtml(url string)(html string, err error){
	resp, err := mainClient.Get(url)
	if err != nil {
		return "",err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	html = string(body)
	return html, err
}

//find img url from html code and download some of then according to the config 
func digAndSaveImgs(url string) {
	//get all img link from html code
	html,err := digHtml(url)
	if err != nil {
		fmt.Println(err)
		fmt.Println()
	}
	reg1, _ := regexp.Compile(`<img [^>]*>`) 
	imgTags := reg1.FindAllString(html, -1)
	imgSlice := make([]string,0)
	for _,j := range imgTags {
		imgSlice = append(imgSlice, getImgSlice(j, url)...) 
	}
	if len(imgSlice) == 0 {
		return
	}
	
	fmt.Printf("url     [  %s  ]\n", url)
	fmt.Printf("<img>   [  %-6d  ]\n", len(imgSlice))
		
	//create some goroutine and distribute the workes
	urlChan := make(chan string, 100)
	resChan := make(chan int, 20)
	for i:=0; i<thread_numbers; i++ {
		go imgDownLoader(i, urlChan, resChan)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go showResult(len(imgSlice), resChan, &wg)
	for _,j := range imgSlice {		//begin to download images
		urlChan <- j
	}
	//wait for images download complete
	wg.Wait()
	close(urlChan)
	close(resChan)
}

//display the result of images download goroutine
//called by digAndSaveImgs()
func showResult(times int, res <-chan int, wg *sync.WaitGroup){
	counter := 0
	for tres :=  range res {
		counter ++
		fmt.Print(tres," ")
		if counter == times {
			fmt.Println()
			fmt.Printf("QueLen [ %-6d]    Pages [ %-6d]    ImgNum [ %-6d]    ImgSize [ %-6d]  \n\n\n", mylist.Len(), pagesNumber, imgNumbers, totalImgbytes/1048576 )
			wg.Done()
			return
		}
		if (counter%50) == 0 {
			fmt.Println()
		}
	}
}

func digAtags(url string)[]string{
	html, err := digHtml(url)
	if err!=nil {
		fmt.Println(err)
		return make([]string, 0)
	}
	//extract  <a/> tag from html code
	aReg,_ := regexp.Compile(`<a [^>]*>`)
	return aReg.FindAllString(html, -1)
}

//dig all can_be_used_url from an html text according to the config
func DigUrl(url string) []string {
	//get a tag text from html code
	aTags := digAtags(url)
	newUrls := make([]string, 0)
	if len(aTags)==0 {
		return newUrls
	}
	//extract usefully url from <a/>
	for _, a := range aTags {
		aurl := getHref(a, url)
		if !canUse(aurl) { 	//synax not right or already meet before or out of basehref
			if aurl != "" {
				errLog.Println(aurl)
			}		
			continue
		}
		newUrls = append(newUrls, aurl)
	}
	return newUrls
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

