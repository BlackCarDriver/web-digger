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
		if uint64(max_occupy_mb * 1048576) < totalImgbytes || goingToStop == true {
			signal := syscall.Signal(2)
			shutdownsign <- signal
			resChan <- 9
			continue
		}
		resChan <- downLoadImages(url)	
	}
}

//download an image from imgurl
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

//extract and refix the url from a tag
func getHref(aTag string, basehref string)string{
	hrefReg,_ := regexp.Compile(`href="[^"]*`)
	url := hrefReg.FindString(aTag)
	if len(url)<7 {		
		return ""
	}
	url = url[6:]		//erase 'href="'
	if len(url) < 2 {
		return ""
	}
	if strings.HasPrefix(url, "http"){	
		return strings.TrimRight(url, `/`)
	}
	if strings.HasPrefix(url, `//`) {				// "//aa" -> http：//aa
		url = `http:` + url
	}else {											//such as "/aa" or "aa"  such append to basehref
		tindex := strings.Index(basehref, "?")		// www.baidu.com/asdfad?index=...? 
		if tindex > 0 {
			basehref = basehref[:tindex]			// www.baidu.com/aadsfd
		}
		tindex = strings.LastIndex(basehref,`/`)	
		if tindex > 0 {
			basehref = basehref[:tindex]			// www.baudu.com/
			basehref = strings.TrimRight(basehref, `/`)
		}
		if url[0] != '/' {
			url = "/" + url
		}
		url = basehref + url
	}
	url = strings.TrimRight(url, `/`)
	return url
}

//can use mean the url format is right, have specified prefix, and not yet read before
func canUse(url string) bool{
	if !strings.HasPrefix(url, "http") {
		return false
	}
	if url_prefix != "" && !strings.HasPrefix(url, url_prefix){
		return false
	}
	identi := getUrlPath(url)
	if url_map[identi] {
		return false
	}else{
		url_map[identi] = true
	}
	return true
}

//judge if an url contain url_must_contain according to config
func hasKey(rightUrl string) bool {
	if url_must_contain != "" {
		return true
	}
	return strings.Index(rightUrl, url_must_contain) >= 0
}

//if the url can be used and get true after it function, it should be a change page link
func isPageUrl(rightUrl string) bool {
	if url_nextpage == "" {
		return false
	}
	return strings.Index(rightUrl, url_nextpage) >= 0
}

//avoid visit a same path with different url
func getUrlPath(url string) string{
	tindex := strings.Index(url, ":")
	url = url[tindex+1:]
	url = strings.Trim(url, `/`)
	return url
}


