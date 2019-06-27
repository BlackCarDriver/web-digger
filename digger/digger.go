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
)

const(
	config_path = "./config/"
)

var (
	target_url string
	source_path string
)

func init(){
	conf ,err := NewConfig(config_path)
	if err != nil {
		panic(err)
	} 
	conf.Register("source_path","",true)
	source_path, _ = conf.GetString("source_path")
}


func Test(){
	url := "https://imgsa.baidu.com/forum/pic/item/8a13632762d0f703c25d6d2306fa513d2797c566.jpg"
	err := downLoadImages(url)
	fmt.Println(err)
}

func downLoadImages(imgUrl string)error{
	if !isImgUrl(imgUrl) {
		return fmt.Errorf("%s is not match an imgUrl !", imgUrl)
	}
	tmp := strings.LastIndex(imgUrl, `/`)
	imgName := imgUrl[tmp:]
	resp, _ := http.Get(imgUrl)
	body, _ := ioutil.ReadAll(resp.Body)
	imgPath := fmt.Sprint(source_path, os.PathSeparator, imgName)
	out, _ := os.Create(imgPath)
	io.Copy(out, bytes.NewReader(body))
	return nil
}

//judge if a url is a link to an images
func isImgUrl(imgUrl string) bool {
	reg,_:= regexp.Compile(`^http[s]?.*.(jpg|png|jpeg|gif|ico)$`) 
	imgUrl = strings.ToLower(imgUrl)
	return reg.MatchString(imgUrl) 
}