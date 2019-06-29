package digger

import(
	"fmt"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"strconv"
	"syscall"
	"sync"
)

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

//used to distribute download_workes for mutil goroutine
//called by digAndSaveImgs()
func imgDownLoader(no int, urlChan <-chan string , resChan chan<- int){
	for url := range urlChan {
		if uint64(max_occupy_mb * 1048576) < totalImgbytes || goingToStop == true {
			signal := syscall.Signal(2)
			shutdownsign <- signal
			resChan <- 9
			continue
		}
		resChan <- downLoadImages(url)	
	}
}

//download an image specied by url
//called by imgDownLoader
func downLoadImages(imgUrl string)int{
	if !isImgUrl(imgUrl) {
		return 1
	}
	resp, err := http.Get(imgUrl)
	if err != nil{
		return 2
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 3
	}
	imgSize := len(body)
	if imgSize == 0 {
		return 4
	}
	if imgSize < min_img_kb * 1024 {
		return 5
	}
	if imgSize > max_img_mb * 1048576{
		return 6
	}
	imgName := getName(imgUrl)
	go updateTotalSize(uint64(imgSize))
	imgPath := fmt.Sprint(source_path, string(os.PathSeparator), imgName)
	out, err := os.Create(imgPath)
	defer out.Close()
	if err != nil {
		errLog.Printf("%s  ---->  %v \n", imgPath, err)
		return 7
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return 8
	}
	return 0
}


//=============================== the following is tools functions ================================

//get a file name for download images
func getName(name string) string{
	tmp := strings.LastIndex(name, ".")
	suffix := name[tmp:]
	getNameMutex.Lock()
	newName := strconv.Itoa(imgNumbers)+suffix
	imgNumbers++
	getNameMutex.Unlock()
	return newName
}

//judge if a url is a link to an images
func isImgUrl(imgUrl string) bool {
	reg,_:= regexp.Compile(`[^"]*.(jpg|png|jpeg|gif|ico)$`) 
	imgUrl = strings.ToLower(imgUrl)
	return reg.MatchString(imgUrl) 
}

//record the size of download images 
func updateTotalSize(addBytes uint64){
	updataSizeMutex.Lock()
	totalImgbytes += addBytes
	updataSizeMutex.Unlock()
}

