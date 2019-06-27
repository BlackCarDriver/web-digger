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
)



func imgDownLoader(no int, urlChan <-chan string , resChan chan<- int){
	for url := range urlChan {
		if uint64(max_occupy_mb * 1048576) < totalImgbytes {
			signal := syscall.Signal(2)
			shutdownsign <- signal
			resChan <- 9
		}
		resChan <- downLoadImages(url)	
	}
}

//download an image from imgurl
func downLoadImages(imgUrl string)int{
	if !isImgUrl(imgUrl) {
		return 1
	}
	imgName := getName(imgUrl)
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
	go updateTotalSize(uint64(imgSize))
	imgPath := fmt.Sprint(source_path, string(os.PathSeparator), imgName)
	out, err := os.Create(imgPath)
	defer out.Close()
	if err != nil {
		return 7
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return 8
	}
	return 0
}

//find all image url from an img tag
func getImgSlice(imgTag string, basehref string)[]string{
	imgReg, _ := regexp.Compile(`="[^ ]*.(jpg|png|jpeg|gif){1}"`)
	urls := imgReg.FindAllString(imgTag, -1)
	for i:=0; i< len(urls); i++ {
		urls[i] = urls[i][2 : len(urls[i])-1]
		if strings.HasPrefix(urls[i], `//`) {
			urls[i] = "http:" + urls[i]
		}else if strings.HasPrefix(urls[i], `/`){
			urls[i] = basehref + urls[i]
		}
	}
	return urls
}

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

