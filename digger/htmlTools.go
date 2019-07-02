package digger

import(
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)
 

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
 
//find all <a/> in a html code of specifed url
func digAtags(url string)[]string{
	html, err := digHtml(url)
	res := make([]string, 0)
	if err!=nil {
		fmt.Println(err)
		return res
	}
	//extract  <a/> tag from html code
	aReg,_ := regexp.Compile(`<a [^>]*>`)
	res = aReg.FindAllString(html, -1)
	return res
}

//find and select some url from an html text 
//only right syntax url and frist_meet url would be returned
func digLinkUrls(url string) []string {
	//get a tag text from html code
	aTags := digAtags(url)
	newUrls := make([]string, 0)
	if len(aTags)==0 {
		return newUrls
	}
	//extract some right and first_times_used url from <a/>
	for _, a := range aTags {
		aurl := getHref(a, url)
		if !canUsed(aurl) { 	//synax not right or already check before or out of basehref
			//errLog.Println(aurl)	
			continue
		}
		newUrls = append(newUrls, aurl)
	}
	return newUrls
} 

//find all image url from an <img/>
func getImgUrls(imgTag string, basehref string)[]string{
	imgReg, _ := regexp.Compile(`="[^ ]*.(jpg|png|jpeg|gif){1}"`)
	urls := imgReg.FindAllString(imgTag, -1)
	if len(urls)==0 {
		imgTag = strings.Replace(imgTag, `'`, `"`, -1)
		urls = imgReg.FindAllString(imgTag, -1)
	}
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

//extract and correct the url from a tag
//called by digurl()
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

//=================== tools function place below =================================

//return if a url is checked by it function before
//if a url have a worng syntax will also return false
func canUsed(url string) bool{
	if !strings.HasPrefix(url, "http"){
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

//get a string to identified urls to same path 
//called by wasUsed
func getUrlPath(url string) string{
	tindex := strings.Index(url, ":")
	url = url[tindex+1:]
	url = strings.Trim(url, `/`)
	return url
}


func hasPageTag(atag string) bool {
	if page_tag == "" {
		return false
	}
	if strings.Index(atag, page_tag) < 0  {
		return false
	}
	return true
}

func hasTargetTag(atag string) bool {
	if target_tag == "" {
		return true
	}
	if strings.Index(atag, target_tag) < 0  {
		return false
	}
	return true
}
