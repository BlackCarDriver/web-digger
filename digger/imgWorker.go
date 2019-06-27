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
	"math/rand"
	"time"
)

//download an image from imgurl
func downLoadImages(imgUrl string)error{
	if !isImgUrl(imgUrl) {
		return fmt.Errorf("%s is not match an imgUrl !", imgUrl)
	}
	tmp := strings.LastIndex(imgUrl, `/`)
	imgName := imgUrl[tmp:]
	imgName = strings.Trim(imgName, `/`)
	if !nameIsOk(imgName) {
		imgName =  reName(imgName)
	}
	resp, err := http.Get(imgUrl)
	if err != nil{
		return fmt.Errorf(" http.Get(imgUrl) error: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll(resp.Body) error: %v", err)
	}
	imgPath := fmt.Sprint(source_path, string(os.PathSeparator), imgName)
	out, err := os.Create(imgPath)
	defer out.Close()
	if err != nil {
		return fmt.Errorf("os.Create(imgPath) error: %v", err)
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("bytes.NewReader(body) error: %v", err)
	}
	return nil
}

//find all image url from an img tag
func getImgSlice(imgTag string)[]string{
	imgReg, _ := regexp.Compile(`="[^ ]*.(jpg|png|jpeg|gif){1}"`)
	urls := imgReg.FindAllString(imgTag, -1)
	for i:=0; i< len(urls); i++ {
		urls[i] = urls[i][2 : len(urls[i])-1]
		if strings.HasPrefix(urls[i], `//`) {
			urls[i] = "http:" + urls[i]
		}
	}
	return urls
}

//judge if a string can be used to name a file
func nameIsOk(name string) bool {
	nameReg,_ := regexp.Compile(`^[a-zA-Z0-9_.!@#$%^&()]{4,100}$`)
	return nameReg.MatchString(name)
}

//change an file name
func reName(name string) string{
	tmp := strings.LastIndex(name, ".")
	suffix := name[tmp:]
	return GetRandomString(20)+suffix
}

//create an random string that with length l
func  GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}


//judge if a url is a link to an images
func isImgUrl(imgUrl string) bool {
	reg,_:= regexp.Compile(`[^"]*.(jpg|png|jpeg|gif|ico)$`) 
	imgUrl = strings.ToLower(imgUrl)
	return reg.MatchString(imgUrl) 
}
