package digger

import(
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

)

const(
	config_path = "./config/"
)

var (
	target_url string
	source_path string
)

//init config values
func init(){
	conf ,err := NewConfig(config_path)
	if err != nil {
		panic(err)
	} 
	conf.Register("source_path","",true)
	source_path, _ = conf.GetString("source_path")
}


func Test(){
	url := `https://tieba.baidu.com/f?kw=%E6%9D%8E%E6%AF%85`;
	DigUrl(url)
}



//visit an url and do somthing through the html text
func DigUrl(targetUrl string) error{
	resp, err := http.Get(targetUrl)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	html := string(body)
	//get all img url from html code
	reg1, _ := regexp.Compile(`<img [^>]*>`) 
	imgTags := reg1.FindAllString(html, -1)
	imgSlice := make([]string,0)
	for _,j := range imgTags {
		imgSlice = append(imgSlice, getImgSlice(j)...) 
	}
	for _,j := range imgSlice {
		err := downLoadImages(j)
		if err != nil {
			fmt.Println(j , " : ", err)
		}else{
			fmt.Printf("0")
		}
	}
	return nil
}

