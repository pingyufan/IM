package models

import (
	"IM/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"gopkg.in/fatih/set.v0"
	"gorm.io/gorm"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	gorm.Model
	UserID     int64  // 发送者
	TargetID   int64  // 接收者
	Type       int    // 发送类型 （1私聊  2群聊  3 心跳。。
	Media      int    // 消息类型
	Content    string // 消息内容
	CreateTime uint64 // 创建时间
	ReadTime   uint64 // 读取时间
	Pic        string
	Url        string
	Desc       string
	Amount     int //其他数字统计
}

func (m *Message) TableName() string {
	return "message"
}

// 用于管理 WebSocket 连接的状态和相关信息，便于在实时通信应用中维护客户端的连接和交互
type Node struct {
	Conn          *websocket.Conn //连接
	Addr          string          //客户端地址
	FirstTime     uint64          //首次连接时间
	HeartbeatTime uint64          //心跳时间
	LoginTime     uint64          //登录时间
	DataQueue     chan []byte     //消息
	GroupSets     set.Interface   //好友 / 群
}

// 映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

// 读写锁
var rwLocker sync.RWMutex

// 需要 ：发送者ID ，接受者ID ，消息类型，发送的内容，发送类型
func Chat(writer http.ResponseWriter, request *http.Request) {
	// 获取token
	query := request.URL.Query()
	Id := query.Get("userId")
	userId, _ := strconv.ParseInt(Id, 10, 64)

	isvalida := true
	conn, err := (&websocket.Upgrader{
		//token 校验
		CheckOrigin: func(r *http.Request) bool {
			return isvalida
		},
	}).Upgrade(writer, request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 获取conn
	currentTime := uint64(time.Now().Unix())
	node := &Node{
		Conn:          conn,
		Addr:          conn.RemoteAddr().String(), //获取网络连接的远程地址
		HeartbeatTime: currentTime,
		LoginTime:     currentTime,
		DataQueue:     make(chan []byte, 50),
		GroupSets:     set.New(set.ThreadSafe),
	}
	// userid 跟 node绑定 并加锁
	rwLocker.Lock()
	clientMap[userId] = node
	rwLocker.Unlock()
	// 完成发送逻辑
	go sendProc(node)
	go recvProc(node)
	// 加入在线用户到缓存
	SetUserOnlineInfo("online_"+Id, []byte(node.Addr), time.Duration(viper.GetInt("timeout.RedisOnlineTime"))*time.Hour)
}

func sendProc(node *Node) {
	for {
		select {
		case data := <-node.DataQueue:
			fmt.Println("[ws]sendProc >>> msg :", string(data))
			// 向 WebSocket 连接发送消息
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func recvProc(node *Node) {
	for {
		_, data, err := node.Conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		msg := Message{}
		err = json.Unmarshal(data, &msg)
		if err != nil {
			fmt.Println(err)
		}
		// 心跳检测 msg.Media == -1 || msg.Type == 3
		if msg.Type == 3 {
			currentTime := uint64(time.Now().Unix())
			node.Heartbeat(currentTime)
		} else {
			dispatch(data)
			broadMsg(data) //todo 将消息广播到局域网
			fmt.Println("[ws] recvProc <<<< ", string(data))
		}
	}
}

var udpsendChan chan []byte = make(chan []byte, 1024)

func broadMsg(data []byte) {
	udpsendChan <- data
}

func init() {
	go udpSendProc()
	go udpRecvProc()
	fmt.Println("init goroutine ")
}

// 创建并维护一个 UDP 连接，持续监听通道中的数据，并将其发送到指定的目标地址。
func udpSendProc() {
	con, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(192, 168, 0, 255), // 路由网关地址
		Port: viper.GetInt("port.udp"),
	})
	defer con.Close()
	if err != nil {
		fmt.Println(err)
	}
	for {
		select {
		case data := <-udpsendChan:
			fmt.Println("[ws] udpSendProc >>> ", string(data))
			_, err := con.Write(data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func udpRecvProc() {
	con, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: viper.GetInt("port.udp"),
	})
	if err != nil {
		fmt.Println(err)
	}
	defer con.Close()
	for {
		var buf [512]byte
		n, err := con.Read(buf[0:])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("udpRecvProc data :", string(buf[0:n]))
		dispatch(buf[0:n])
	}
}

// 后端调度逻辑处理
func dispatch(data []byte) {
	msg := Message{}
	msg.CreateTime = uint64(time.Now().Unix())
	err := json.Unmarshal(data, &msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	switch msg.Type {
	case 1: //私信
		fmt.Println("dispatch data :", string(data))
		sendMsg(msg.TargetID, data)

	case 2: // 群发
		sendGroupMsg(msg.TargetID, data)

	}
}

func sendGroupMsg(targetId int64, msg []byte) {
	fmt.Println("开始发送群消息")
	userIds := SearchUserByGroupId(uint(targetId))
	for i := 0; i < len(userIds); i++ {
		if targetId != int64(userIds[i]) {
			sendMsg(int64(userIds[i]), msg)
		}
	}
}

func JoinGroup(userId uint, comId string) (int, string) {
	contact := Contact{}
	contact.OwnerId = userId

	contact.Type = 2
	community := Community{}

	utils.DB.Where("id=? or name=?", comId, comId).Find(&community)
	if community.Name == "" {
		return -1, "没有找到群"
	}
	utils.DB.Where("owner_idd=? and target_id=? and type=2", userId, comId).Find(&contact)
	if !contact.CreatedAt.IsZero() {
		return -1, "已加过该群"
	} else {
		contact.TargetId = community.ID
		utils.DB.Create(&contact)
		return 0, "加群成功"
	}
}

// 处理消息的发送逻辑
func sendMsg(userId int64, msg []byte) {
	rwLocker.RLock()
	node, ok := clientMap[userId]
	rwLocker.RUnlock()

	// 解析消息
	jsonMsg := Message{}
	json.Unmarshal(msg, &jsonMsg)

	ctx := context.Background()
	targetIdStr := strconv.Itoa(int(userId))
	userIdStr := strconv.Itoa(int(jsonMsg.UserID))
	jsonMsg.CreateTime = uint64(time.Now().Unix())

	// 从 Redis 获取该用户的在线状态
	r, err := utils.Red.Get(ctx, "online_"+userIdStr).Result()
	if err != nil {
		fmt.Println(err)
	}
	// 如果用户在线， 发送消息
	if r != "" {
		if ok {
			fmt.Println("sendMsg >>> userId: ", userId, "msg:", string(msg))
			node.DataQueue <- msg // 消息放入消息队列
		}
	}

	// 生成Redis键  存储历史消息,确保了相同用户对之间的消息可以统一存储
	var Key string
	if userId > jsonMsg.UserID {
		Key = "msg_" + userIdStr + "_" + targetIdStr
	} else {
		Key = "msg_" + targetIdStr + "_" + userIdStr
	}

	// 从Redis获取历史消息
	res, err := utils.Red.ZRevRange(ctx, Key, 0, -1).Result()
	if err != nil {
		fmt.Println(err)
	}

	// 计算新的信息数 (容量+1)
	score := float64(cap(res)) + 1
	// 将消息添加到Redis的有序集合中
	ress, e := utils.Red.ZAdd(ctx, Key, &redis.Z{score, string(msg)}).Result()
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(ress)
}

// 获取缓存里面的消息
func RedisMsg(userIdA int64, userIdB int64, start int64, end int64, isRev bool) []string {
	rwLocker.RLock()
	//node, ok := clientMap[userIdA]
	rwLocker.RUnlock()
	//jsonMsg := Message{}
	//json.Unmarshal(msg, &jsonMsg)

	ctx := context.Background()
	userIdStr := strconv.Itoa(int(userIdA))
	targetIdStr := strconv.Itoa(int(userIdB))

	var Key string
	if userIdA > userIdB {
		Key = "msg_" + targetIdStr + "_" + userIdStr
	} else {

		Key = "msg_" + userIdStr + "_" + targetIdStr
	}

	//根据排序需求，从Redis中检索消息列表，正序/倒序
	var rels []string
	var err error
	if isRev {
		rels, err = utils.Red.ZRange(ctx, Key, start, end).Result()
	} else {
		rels, err = utils.Red.ZRevRange(ctx, Key, start, end).Result()
	}
	if err != nil {
		fmt.Println(err)
	}

	return rels
}

// 需要重写此方法才能完整的msg转byte[]
func (msg Message) MarshalBinary() ([]byte, error) {
	return json.Marshal(msg)
}

// 更新用户心跳
func (node *Node) Heartbeat(currentTime uint64) {
	node.HeartbeatTime = currentTime
	return
}

// 清理超时连接
func CleanConnection(param interface{}) (result bool) {
	result = true
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("cleanconnection err", r)
		}
	}()

	currentTIme := uint64(time.Now().Unix())
	for i := range clientMap {
		node := clientMap[i]
		if node.IsHearbeatTImeOut(currentTIme) {
			fmt.Println("心跳超时。。。关闭连接", node)
			node.Conn.Close()
		}
	}
	return result
}

func (node *Node) IsHearbeatTImeOut(currentTime uint64) (timeout bool) {
	if node.HeartbeatTime+viper.GetUint64("timeout.HearbeatMaxTimme") <= currentTime {
		fmt.Println("心跳超时。。。自动下线", node)
		timeout = true
	}
	return
}
