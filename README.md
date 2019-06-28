# web-digger

----------

### 简介
这是一个用go编写的网络爬虫，主要功能包括自动爬取网页上的图片。使用方法：对config.conf中的参数进行设置，然后直接运行程序即可。


----------


### 状态码含义
0 - - - - - - 保存图片成功

1 - - - - - -  url无效

2 - - - - - -  无法访问

3 - - - - - -  读取响应主体失败

4 - - - - - -  响应主体长度为0

5 - - - - - -  图片体积小于设定的最小值 

6 - - - - - -  图片体积大于设定的最大值 

7 - - - - - -  在磁盘创建图像文件失败

8 - - - - - -  保存图片到磁盘失败

9 - - - - - -  保存图片的总空间超过了设定值


----------

### 配置案例  

    # 保存图片的目录 （必填）
    source_path = "D:\TempImg"
    
    # url 种子, （入口）（必填）
    url_seed = "https://tieba.baidu.com/f?kw=%E5%A5%B3%E5%9B%BE"
    
    # 同时下载图片的线程数 ,(选填,默认为 1)
    thread_numbers = 20
    
    # 忽略小于多少 kb 一下的图片, (选填,默认为 1)
    min_img_kb = 10
    
    # 忽略大于多少 mb 以上的图片, (选填,默认为 10)
    max_img_mb = 5
    
    # 下载图片占用磁盘的最大空间, (选填，默认100)
    max_occupy_mb = 1000
    

