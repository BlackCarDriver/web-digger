package digger

import(
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

//find all <a/> in a html code of specifed url
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

//find all image url from an <img/>
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

//extract and refix the url from a tag
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

//avoid visit a same path with different url
func getUrlPath(url string) string{
	tindex := strings.Index(url, ":")
	url = url[tindex+1:]
	url = strings.Trim(url, `/`)
	return url
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
