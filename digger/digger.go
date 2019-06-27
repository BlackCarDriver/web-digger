package digger

import(
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
)

const(
	config_path = "./config/"
)

var (
	source_path string
	thread_numbers int
	url_seed	string
	target_url string
)

//init config values
func init(){
	conf ,err := NewConfig(config_path)
	if err != nil {
		panic(err)
	} 
	conf.Register("source_path", "", true)
	conf.Register("thread_numbers", 0, true)
	conf.Register("url_seed", "", true )
	source_path, _ = conf.GetString("source_path")
	thread_numbers,_ = conf.GetInt("thread_numbers")
	url_seed,_ = conf.GetString("url_seed")
}


func Test(){
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
		imgSlice = append(imgSlice, getImgSlice(j)...) 
	}

	//create some imgages download workers
	fmt.Println("   images numbers: ", len(imgSlice))
	urlChan := make(chan string, 100)
	resChan := make(chan bool, 20)
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
func showResult(times int, res <-chan bool, wg *sync.WaitGroup){
	counter := 0
	for tmp :=  range res {
		counter ++
		if tmp {
			fmt.Print("0")
		}else{
			fmt.Print("1")
		}
		if counter == times {
			fmt.Println(" \n the work is complete !")
			wg.Done()
			return
		}
	}
}