package digger

import(
	"fmt"
	"reflect"
	"strconv"
	"os"
	"io"
	
	"errors"
	"strings"
	"bufio"
	"regexp"
)

//recorde the filename that alread read, each file can only read once
var readHistory = make(map[string]bool)

/*
explain of Config struct:
	configPath is the root path of config files, all files with suffix ".conf" will be parse into rawConf
when the struct is init. 
	rawConf is the map tmpely saving the string that read from config files, those string will not be
used until you register then.
	ripeConf is the map saving config value, those config is read from rawConf through Register()
*/
type Config struct{
	configPath string
	rawConf map[string]string 
	ripeConf map[string]interface{}
}

type ConfigMachine interface {
	InitWithFilesPath(filesPath string) error
	Display() 
	Register(keyName string , dfValue interface{}, isImportant bool)
	getInterface(keyName string) (value interface{}, err error)
	GetInt(keyName string) (value int, err error)
	GetInts(keyName string) (value []int, err error)
	GetString(keyName string) (value string, err error)
	GetStrings(keyName string) (value []string, err error)
	GetBool(keyName string) (value bool, err error)
}

//the mainly way of obtain a Config
func NewConfig(confPath string)(ConfigMachine, error) {
	newMachine := new(Config)
	err := newMachine.InitWithFilesPath(confPath)
	return newMachine, err
}

//=========== method in interface ===============
func (c *Config) InitWithFilesPath(Configpath string) error{
	if c.configPath != "" {
		return errors.New("You can't init the Confi twice!")
	}
	if !strings.HasSuffix(c.configPath, "/") {
		c.configPath += "/"
	}
	c.rawConf = make(map[string]string)
	c.ripeConf = make(map[string]interface{}) 
	c.configPath = Configpath;
	errList := c.readAllConfig()
	return errList
}

func (c *Config) Register(confName string, dfValue interface{}, isStrict bool){
	rawStr, ok := c.rawConf[confName]
	tyName := reflect.TypeOf(dfValue).String()
	var err error
	if !ok && isStrict {	
		err = fmt.Errorf("Config %v don't exit !", confName)
		goto tail
	}
	if !ok && !isStrict {
		c.ripeConf[confName] = dfValue
		goto tail
	}
	switch tyName {
	case "int":
		tmpInt,err := strconv.Atoi(rawStr)
		if err != nil {
			goto tail
		}
		c.ripeConf[confName] = tmpInt
		break;

	case "string":
		c.ripeConf[confName] = rawStr
		break;

	case "float64": 
		tmpFloat,err := strconv.ParseFloat(rawStr, 64)
		if err != nil {
			goto tail
		}
		c.ripeConf[confName] = tmpFloat
		break;

	case "bool":  
		tmpBool, err := strconv.ParseBool(rawStr)
		if err != nil {
			goto tail 
		}
		c.ripeConf[confName] = tmpBool
		break;

	case "[]string":
		tmpStr := strings.Trim(rawStr,`"`)
		c.ripeConf[confName] = strings.Split(tmpStr, `","`)
		break;

	case "[]int":
		tmpArry := strings.Split(rawStr,",")
		tmpIntArry := make([]int, 0)
		for _,strInt := range tmpArry {
			tmpInt, err := strconv.Atoi(strInt)
			if err!=nil {
				goto tail
			}
			tmpIntArry = append(tmpIntArry, tmpInt)
		}
		c.ripeConf[confName] = tmpIntArry
		break
	default:
		err = fmt.Errorf("Unsupport type : %v", tyName)
	}
	tail:
	//handle your errors
	if err!=nil{
		errMsg := fmt.Sprintf("Error happen when register config %s , msg: %v", confName, err)
		panic(errMsg)
	}
}

//display the key name and value name in rawMap and ripeMap
func (c *Config) Display(){
	fmt.Println("============= rawConf ======== ")
	for k,v := range c.rawConf {
	 	fmt.Printf(" %v -->  %v \n", k,v)
	}
	fmt.Println( "============ ripefMap ========" )
	for k,v := range c.ripeConf {
		fmt.Printf(" %v -->  %v \n", k,v)
	}
}

//called by other GetXXX functions
func (c *Config) getInterface(keyName string)(value interface{}, err error){
	if !isLegalName(keyName) {
		err = errors.New("keyName is not right!")
		return
	}
	value, ok := c.ripeConf[keyName]
	if !ok {
		err = fmt.Errorf("KeyName %v not found in config list!", keyName)
		return
	}
	return value,nil
}

func (c *Config)GetInt(keyName string) (value int, err error) {
	any , err := c.getInterface(keyName)
	return any.(int), err 
}

func (c *Config)GetInts(keyName string) (value []int, err error) {
	any , err := c.getInterface(keyName)
	return any.([]int), err 
}

func (c *Config)GetString(keyName string) (value string, err error) {
	any , err := c.getInterface(keyName)
	return any.(string), err 
}

func (c *Config)GetStrings(keyName string) (value []string, err error) {
	any , err := c.getInterface(keyName)
	return any.([]string), err
}

func (c *Config)GetBool(keyName string) (value bool, err error) {
	any , err := c.getInterface(keyName)
	return any.(bool), err 
}


//=============== tools function ==========

//read all files with .conf suffix in configPath
func (c *Config)readAllConfig() error {
	filesPath := c.configPath
	file ,err := os.Open( filesPath )
	if err != nil {
		return err
	}
	defer file.Close()
	fi, err := file.Readdir(0)
	if err != nil {
		return err
	}
	errReport := ""
	for _, info := range fi {
		//only read files that name like *.conf
		if strings.HasSuffix(info.Name(), ".conf") == false {
			continue
		}
		//guarante each file only read one times
		if readHistory[info.Name()] {
			errReport += fmt.Sprintf("can not read %v, already read before...", info.Name())
			continue;
		}
		readHistory[info.Name()] = true
		tmpPath := filesPath + info.Name()
		err := c.readConfig(tmpPath)
		if err != nil {
			errReport += fmt.Sprintf("\n %v", err)
		}
	}
	if errReport == ""{
		return nil
	}
	return errors.New(errReport)
}

//read a config file and save message into Conf.rawMap
func (c *Config)readConfig(path string) error {
	file,err := os.Open(path)
	if handleErr("os.Open(path) ", err, false) {
		return err
	}
	defer file.Close()
	buf := bufio.NewReader(file)
	for{
		lineByte, _, err := buf.ReadLine()
		line := strings.TrimSpace( string(lineByte) )
		if err == io.EOF {	//end of file
			break
		}
		if err != nil {		//other error
			fmt.Println(err)
			return err
		}
		if line == "" {		//ignore empty line
			continue
		}
		if strings.HasPrefix(line, "#") {	//ignore cmment
			continue
		}
		index := strings.Index(line, "=")
		if index <= 0 {						//unknow format
			return errors.New("Reading config was interupt because unexpect fomat of config (index <= 0): " +  string(lineByte) )
		}
		confName := strings.TrimSpace(line[:index])
		confValue := strings.TrimSpace(line[index+1:])
		if len(confName) == 0 || len(confValue) == 0 {	//unknow format
			return errors.New("Reading config was interupt because unexpect fomat of config (len==0): " +  string(lineByte) )
		}

		if isLegalName(confName) == false {				//config name not legal
			return errors.New("Config Name not legal at line : " + string(lineByte) )
		}

		if isStringType(confValue) {	//match string type
			confValue = strings.Trim(confValue, `"`)
			goto saveConf
		}

		if isNumberType(confValue) {	//match int or float type
			goto saveConf
		}

		if confValue=="true" || confValue == "false" {	//match bool type
			goto saveConf
		}
		//read an multi line string to rawMap, dont
		if confValue == `{`	 {		
			tmpStr := ""
			for {
				tmplineByte, _, tmpErr := buf.ReadLine()
				if tmpErr != nil { 	
					return fmt.Errorf("Readding worng by mistack after ‘%v’ , error: %v ", string(lineByte), tmpErr)
				}
				tmpline := string(tmplineByte)
				if strings.HasPrefix(strings.TrimSpace(tmpline), `}`) {
					break
				}
				tmpStr += tmpline 
				tmpStr += "\n"
			}
			confValue = tmpStr
			goto saveConf
		}	

		if confValue == "[" {		//mathch an array
			tmpStr := ""
			for {
				tmplineByte, _, tmpErr := buf.ReadLine()
				if tmpErr != nil { 	
					return fmt.Errorf("Unexpect error when reading array type config in or near : ‘%v’, error: %v ", string(lineByte), tmpErr)
				}
				tmpline := string(tmplineByte)
				tmpline = strings.TrimSpace(tmpline)
				if tmpline == "]" {
					break
				}
				if strings.HasSuffix(tmpline, ",") {
					tmpline = strings.TrimRight(tmpline, ",")
				}
				tmpStr += tmpline
				tmpStr += ","
			}
			confValue = strings.TrimRight(tmpStr, ",")
			goto saveConf
		}

	saveConf:
		c.rawConf[confName] = confValue
	}
	return nil
}

func handleErr(prefix string ,err error, isSeriou bool) ( errNotNull bool) {
	if err == nil {
		return false
	}
	fmt.Println(prefix , err)
	if isSeriou {
		os.Exit(2)
	}
	return true	 
}

//judge if a name of config is legal
func isLegalName(confName string) bool {
	legalNameReg, _ := regexp.Compile(`^[a-zA-Z0-9_]+$`) 
	isLegal := legalNameReg.MatchString(confName)
	return isLegal
}

//judege if a config value match a string type, scuh as `"Is is config value"`
func isStringType(confValue string) bool {
	tmpStr := confValue
	counter := strings.Count(tmpStr, `"`)
	if counter != 2 {
		return false
	}
	tmpStr = strings.Trim(tmpStr, `"`)
	return (strings.Count(tmpStr, `"`) == 0)
}

//judege if a config value match a integer or float type
func isNumberType(confValue string) bool {
	_, isInt := strconv.Atoi(confValue)
	_, isFlo := strconv.ParseFloat(confValue, 64)
	return (isInt==nil || isFlo==nil)
}

//==========================================================

func Main(){
	example()
}

func example(){
	//create an config object by giving config path
	tc,err := NewConfig("./config/conf/")
	if err!=nil {
		fmt.Println("the following is the errors during reading config file :")
		fmt.Println(err)
	}

	//registe config by giving default value
	tc.Register("t_string", "test", true)
	tc.Register("t_string2", "test", true)
	tc.Register("t_muti_string", "test", true)
	tc.Register("t_int", 0, true)
	tc.Register("t_float", 0.1, true)
	tc.Register("t_bool", false, true)
	tc.Register("t_str_arry", make([]string,1), true)
	tc.Register("t_int_array", make([]int, 1), true)
	//register a new config key that don't exist in config file by setting isStrict = false
	tc.Register("newCOnfig", "t_muti_string", false)
	
	//display the config in map
	tc.Display()

	//get a config value by configName
	paragraph, err := tc.GetString("t_muti_string")
	if err!=nil {
		fmt.Println(err)
	}else{
		fmt.Println(paragraph)
	}
}