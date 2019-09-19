package tuisong

//第三方推送集成(个推)
import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gogf/gf/g"
	"github.com/gogf/gf/g/encoding/gjson"
	"github.com/gogf/gf/g/net/ghttp"
	"github.com/gogf/gf/g/os/gcache"
	"github.com/gogf/gf/g/os/glog"
	"github.com/gogf/gf/g/os/gmutex"
	"github.com/gogf/gf/g/os/gtime"
	"github.com/gogf/gf/g/util/gconv"
	"github.com/gogf/gf/g/util/grand"
)

//填写对应客户端应用配置
var (
	AppID        = "irEknDDWA17A10zkrRdQe2"
	AppKey       = "1CpCTSeNTtAFtg77ruhoQ8"
	AppSecret    = "oN9juj3oa27uJBLwz6j054"
	MasterSecret = "rm79LNVvDyAXUVlJbLJJB3"
)

/*
完整API由 (TS_SERVER+参数(appid)+TS_API_*)组成
*/
const (
	TS_SERVER   = "https://restapi.getui.com/v1/"
	TS_API_PUSH = "/push_single" //推送API末尾部分
	TS_API_SIGN = "/auth_sign"   //鉴权API末尾部分
)

const (
	// 通知栏消息布局样式
	T_SYSTEM   = 0 //0:系统样式
	T_GETUI    = 1 //1:个推样式
	T_JUSTPIC  = 4 //4:纯图样式(背景图样式)
	T_FULLOPEN = 6 //6:展开通知样式
	//消息类型

	NOTYPOPLOAD  = "notypopload"  //下载模板
	NOTIFICATION = "notification" //普通消息模板
	LINK         = "link"         //链接模板

)

//模板结构参考第三方推送(个推)开发文档;http://docs.getui.com/getui/server/rest/template/
type TsTemplate struct {
	Message      `json:"message,omitempty"` //消息配置
	Notification interface{}                `json:"notification,omitempty"` //普通通知模板
	Link         interface{}                `json:"link,omitempty"`         //链接模板
	Notypopload  interface{}                `json:"notypopload,omitempty"`  //文件下载模板
	Cid          string                     `json:"cid,omitemtpy"`          //客户端ID,与alias二选其一
	Alias        string                     `json:"alias,omitempty"`        //客户端别名与cid二选其一
	Requestid    string                     `json:"requestid,omitempty"`    //请求唯一标识号
}

//通知栏消息布局样式
type Style struct {
	Type         int    `json:"type"`                   // 通知栏消息布局样式
	Text         string `json:"text,omitempty"`         //通知内容
	Title        string `json:"title,omitempty"`        //通知标题
	Logo         string `json:"logo,omitempty"`         //通知的图标名称,包含后缀名（需要在客户端开发时嵌入），如“push.png”
	Logourl      string `json:"logourl,omitempty"`      //通知图标URL地址
	Is_ring      bool   `json:"is_ring,omitempty"`      //是否响铃,默认响铃
	Is_vibrate   bool   `json:"is_vibrate,omitempty"`   //是否震动,默认振动
	Is_clearable bool   `json:"is_clearable,omitempty"` //是否可清除,默认可清除
	Banner_url   string `json:"banner_url,omitempty"`   //通过url方式指定动态banner图片作为通知背景图,纯图类型通知
	Big_style    string `json:"big_style,omitempty"`    //通知展示样式,枚举值包括 1,2,3
	/*
		big_style	必传属性					展开样式说明
		1		big_image_url				通知展示大图样式，参数是大图的URL地址
		2		big_text					通知展示文本+长文本样式，参数是长文本
		3		big_image_url,banner_url	通知展示大图+小图样式，参数是大图URL和小图URL
	*/
	Big_image_url string `json:"big_image_url,omitempty"` //通知大图URL地址
	Big_text      string `json:"big_text,omitempty"`      //通知展示文本+长文本样式，参数是长文本

}

//简单通知模板
type Notification struct {
	Style                interface{} `json:"style,omitempty"`                //通知栏消息布局样式
	Transmission_type    bool        `json:"transmission_type,omitempty"`    //收到消息是否立即启动应用，true为立即启动，false则广播等待启动，默认是否
	Transmission_content string      `json:"transmission_content,omitempty"` //透传内容
	Duration_begin       string      `json:"duration_begin,omitempty"`       //设定展示开始时间，格式为yyyy-MM-dd HH:mm:ss
	Duration_end         string      `json:"duration_end,omitempty"`         //设定展示结束时间，格式为yyyy-MM-dd HH:mm:ss
}

//点开通知打开网页模板
type Link struct {
	Style          interface{} `json:"style,omitempty"`          //通知栏消息布局样式
	Duration_begin string      `json:"duration_begin,omitempty"` //设定展示开始时间，格式为yyyy-MM-dd HH:mm:ss
	Duration_end   string      `json:"duration_end,omitempty"`   //设定展示结束时间，格式为yyyy-MM-dd HH:mm:ss
	Url            string      `json:"url,omitempty"`            //打开网址;当使用link作为推送模板时，当客户收到通知时，在通知栏会下是一条含图标、标题等的通知，用户点击时，可以打开您指定的网页。
}

//点击通知弹窗下载模板
type Notypopload struct {
	Style          interface{} `json:"style,omitempty"`          //通知栏消息布局样式
	Notyicon       string      `json:"notyicon,omitempty"`       //是	通知栏图标
	Notytitle      string      `json:"notytitle,omitempty"`      //是	通知标题
	Notycontent    string      `json:"notycontent,omitempty"`    //是	通知内容
	Poptitle       string      `json:"poptitle,omitempty"`       //是	弹出框标题
	Popcontent     string      `json:"popcontent,omitempty"`     //是	弹出框内容
	Popimage       string      `json:"popimage,omitempty"`       //是	弹出框图标
	Popbutton1     string      `json:"popbutton1,omitempty"`     //是	弹出框左边按钮名称
	Popbutton2     string      `json:"popbutton2,omitempty"`     //是	弹出框右边按钮名称
	Loadicon       string      `json:"loadicon,omitempty"`       //否	现在图标
	Loadtitle      string      `json:"loadtitle,omitempty"`      //否	下载标题
	Loadurl        string      `json:"loadurl,omitempty"`        //是	下载文件地址
	Is_autoinstall bool        `json:"is_autoinstall,omitempty"` //否	是否自动安装，默认值false
	Is_actived     bool        `json:"is_actived,omitempty"`     //否	安装完成后是否自动启动应用程序，默认值false
	Androidmark    string      `json:"androidmark,omitempty"`    //否	安卓标识
	Symbianmark    string      `json:"symbianmark,omitempty"`    //否	塞班标识
	Iphonemark     string      `json:"iphonemark,omitempty"`     //否	苹果标志
	Duration_begin string      `json:"duration_begin,omitempty"` //否	设定展示开始时间，格式为yyyy-MM-dd HH:mm:ss
	Duration_end   string      `json:"duration_end,omitempty"`   //否	设定展示结束时间，格式为yyyy-MM-dd HH:mm:ss
}

//消息配置结构
type Message struct {
	Appkey              string `json:"appkey,omitempty"`              //注册应用时生成的appkey
	Is_offline          bool   `json:"is_offline,omitempty"`          //是否离线推送
	Offline_expire_time int    `json:"offline_expire_time,omitempty"` //消息离线存储有效期，单位：ms
	Push_network_type   int    `json:"push_network_type,omitempty"`   //选择推送消息使用网络类型，0：不限制，1：wifi
	Msgtype             string `json:"msgtype,omitempty"`             //消息应用类型，可选项：notification、link、notypopload、transmission
}

/*
	返回默认模板结构,可直接发送也可重构
*/
func New(cid, content string, msgtype ...string) (TsTemplate, error) {
	mtype := NOTIFICATION
	if len(msgtype) > 0 {
		mtype = msgtype[0]
	}
	ts_tmp := TsTemplate{
		Message: Message{
			Appkey:              AppKey,
			Is_offline:          true,
			Offline_expire_time: 10000000,
			Msgtype:             NOTIFICATION,
		},
		Cid:       cid,
		Requestid: grand.Digits(grand.N(15, 30)),
	}
	switch mtype {
	case NOTIFICATION:
		ts_tmp.Message.Msgtype = mtype
		ts_tmp.Notification = Notification{
			Style: Style{
				Type:         T_SYSTEM, //系统样式
				Text:         content,
				Title:        "分时住",
				Is_clearable: true, //可清除
				Is_ring:      true, //响铃
				Is_vibrate:   true, //震动
			},
			Transmission_type: true, //点击时默认打开应用
		}
	case LINK:
		ts_tmp.Message.Msgtype = mtype
		ts_tmp.Link = Link{
			Style: Style{
				Type:         T_SYSTEM, //系统样式
				Text:         content,
				Title:        "分时住",
				Is_clearable: true, //可清除
				Is_ring:      true, //响铃
				Is_vibrate:   true, //震动
			},
			Url: content,
		}
	case NOTYPOPLOAD:
		ts_tmp.Message.Msgtype = mtype
		ts_tmp.Notypopload = Notypopload{
			Style: Style{
				Type:         T_SYSTEM, //系统样式
				Text:         "通知内容",
				Title:        "分时住",
				Is_clearable: true, //可清除
				Is_ring:      true, //响铃
				Is_vibrate:   true, //震动
			},
			Notyicon:       "noty.png",
			Notytitle:      "请填写通知标题",
			Notycontent:    "请填写通知内容",
			Poptitle:       "请填写弹出框标题",
			Popcontent:     "请填写弹出框内容",
			Popimage:       "image.png",
			Popbutton1:     "leftButton",
			Popbutton2:     "rightButton",
			Loadicon:       "",
			Loadtitle:      "请填写下载标题",
			Loadurl:        "请填写下载文件地址",
			Is_autoinstall: false,
			Is_actived:     false,
		}
	default:
		glog.Error("未知的消息推送类型！:", mtype)
		return ts_tmp, errors.New("未知的消息推送类型！" + mtype)
	}
	return ts_tmp, nil
}

/*
	推送消息流程
	1.调用鉴权接口,获取auth_token
	2.调用推送接口
*/
func (tmp *TsTemplate) Send() {
	params, err := gjson.Encode(tmp)
	if err != nil {
		glog.Error("json解析失败！", err)
		return
	}
	auth_token := gcache.Get("auth_token")
	if auth_token == nil { //过期重新获取
		auth_token = getAuthToken()
		if auth_token == nil {
			glog.Error("发送失败！鉴权获取失败！")
			return
		}
	}
	//该URL为单推URL,如需群推查阅文档修改此处和对应消息模板内容即可
	url := TS_SERVER + AppID + TS_API_PUSH
	http_client := ghttp.NewClient()
	http_client.SetHeader("Content-Type", "application/json;charset=utf-8")
	http_client.SetHeader("authtoken", gconv.String(auth_token))
	r, err := http_client.Post(url, params)
	if err != nil {
		glog.Error(err)
		return
	}
	defer r.Close()
	resp := gjson.New(r.ReadAll()).ToMap()
	if resp["result"] == "ok" {
		glog.Println("推送发送成功", resp)
	} else {
		glog.Error("推送发送失败", resp)
	}

}

/*
	用途:推送鉴权
	功能:auth_token获取
	调用第三方推送API
	变量:url,params,resp

*/
func getAuthToken() interface{} {
	url := TS_SERVER + AppID + TS_API_SIGN
	http_client := ghttp.NewClient()
	http_client.SetHeader("Content-Type", "application/json;charset=utf-8")
	//sign = sha256(appkey+timestamp+mastersecret)
	sign_byte := sha256.Sum256([]byte(AppKey + gconv.String(gtime.Now().Millisecond()) + MasterSecret))
	sign := fmt.Sprintf("%x", sign_byte)
	//构建鉴权参数
	params := map[string]interface{}{
		"sign":      sign,
		"timestamp": gtime.Now().Millisecond(),
		"appkey":    AppKey,
	}
	//参数JSON序列化
	json_params, _ := gjson.New(params).ToJson()

	r, err := ghttp.NewClient().Post(url, json_params)
	if err != nil {
		glog.Error("鉴权请求失败", err)
		return nil
	}
	defer r.Close()
	//成功结果{"result":"ok","expire_time":"1568875773358","auth_token":"98b74e881a76dd5bbc98ec6cab8e650dfa33f24b1313507ba19c8947a320d7f5"}
	resp := gjson.New(r.ReadAll()).ToMap()
	if resp["result"] == "ok" { //存入缓存存到过期时间前1秒
		gcache.Set("auth_token", resp["auth_token"], gtime.D-1)
		return resp["auth_token"]
	}
	glog.Error(resp["result"])
	return nil

}

/*
	获取ClientID,该值由APP客户端通过集成第三方推送插件后获取客户机的唯一标识后传给后台与用户信息进行绑定并存储。
*/
func GetCid(uuid string) (string, error) {
	mu := gmutex.New()
	mu.Lock()
	defer mu.Unlock()
	dbres, err := g.DB().Table("device_bind").Where("user_uuid=?", uuid).One()
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if dbres == nil {
		return "", errors.New("未查询到该用户关联的设备id")
	}
	return dbres.ToMap()["cid"].(string), nil
}
