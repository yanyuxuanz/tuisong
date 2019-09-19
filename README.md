# tuisong
基于golang集成第三方推送（个推）实现服务端向客户端推送

#Quick Start

下载包到本地工作目录 或 mod直接拉取
```
go get github.com/yanyuxuanz/tuisong
```

使用

```
  import "github.com/yanyuxuanz/tuisong"
  cid = "xxxxxxxxxxxxxxx"//客户端ID，可根据实际业务情况获取和存储
  content ="填写对应推送内容。。。。"
  if ts,err := tuisong.New(cid,msg.Content);err == nil{
			ts.Send()
	}
  
```

#Thanks

![gf](https://gf.cdn.johng.cn/logo.png)
[gogf](https://github.com/gogf/gf "gogf")
