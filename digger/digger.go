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
)

const(
	config_path = "./config/"
)
var (
	randMachine *rand.Rand
	getNameMutex  *sync.Mutex
	updataSizeMutex *sync.Mutex
	shutdownsign	 chan os.Signal
	goingToStop		bool
	imgNumbers int		//how many images already download
	totalImgbytes uint64		//the total size of all images already download
)

//config values
var (
	source_path string
	thread_numbers int
	url_seed	string
	min_img_kb int
	max_img_mb int
	max_occupy_mb int
)

//init config values
func init(){
	imgNumbers = 0
	totalImgbytes = 0
	goingToStop = false
	shutdownsign = make(chan os.Signal, 10)
	ns := rand.NewSource(time.Now().UnixNano())
	randMachine = rand.New(ns)
	getNameMutex = &sync.Mutex{}
	updataSizeMutex = &sync.Mutex{}

	conf ,err := NewConfig(config_path)
	if err != nil {
		panic(err)
	} 
	conf.Register("source_path", "", true)
	conf.Register("url_seed", "", true )
	conf.Register("thread_numbers", 1, false)
	conf.Register("min_img_kb", 1, false)
	conf.Register("max_img_mb", 10, false)
	conf.Register("max_occupy_mb",1000, false)
	source_path, _ = conf.GetString("source_path")
	thread_numbers,_ = conf.GetInt("thread_numbers")
	url_seed,_ = conf.GetString("url_seed")
	min_img_kb, _ = conf.GetInt("min_img_kb")
	max_img_mb, _ = conf.GetInt("max_img_mb")
	max_occupy_mb, _ = conf.GetInt("max_occupy_mb")
	//conf. Display()
}


func Test(){
	go destructor()
	DigUrl(url_seed)
}


//visit an url and do somthing through the html text
func DigUrl(targetUrl string) error{
	fmt.Printf("Begin to analyze %s :     ", targetUrl)
	resp, err := http.Get(targetUrl)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	html := string(body)
	//get all img url from html code and colloct into a slice
	reg1, _ := regexp.Compile(`<img [^>]*>`) 
	imgTags := reg1.FindAllString(html, -1)
	imgSlice := make([]string,0)
	for _,j := range imgTags {
		imgSlice = append(imgSlice, getImgSlice(j, targetUrl)...) 
	}

	//create some imgages download workers
	fmt.Println("   images numbers: ", len(imgSlice))
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