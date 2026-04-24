package biz

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tbaehler/gin-keycloak/pkg/ginkeycloak"
	"golang.org/x/oauth2"

	"fdi_data_board/internal/conf"
)

const (
	ConnTypeNone       = "Unknown"
	ConnTypeHelloWorld = "hello_world"
)

const (
	ReporterTypeNone       = "Unknown"
	ReporterTypeHelloWorld = "hello_world"
)

const (
	WriteWait  = 5 * time.Second
	PongWait   = 60 * time.Second
	PingPeriod = PongWait * 5 / 10
)

type WsClient struct {
	id      string
	enterAt time.Time

	cd *conf.Data

	connType  string
	conn      *websocket.Conn
	sessionId string
	wsHelper  *ServerWsClientHelper

	sendChan chan []byte
}

type WebsocketReqData struct {
	SessionId string `json:"session_id"`
}

type WebsocketReq struct {
	Request string            `json:"request"`
	Token   string            `json:"token"`
	Data    *WebsocketReqData `json:"data"`
}

func NewWsClient(cd *conf.Data, conn *websocket.Conn, helper *ServerWsClientHelper) *WsClient {
	return &WsClient{
		id:       uuid.NewString(),
		enterAt:  time.Now(),
		cd:       cd,
		conn:     conn,
		connType: ConnTypeNone,
		wsHelper: helper,
		sendChan: make(chan []byte),
	}
}

func (wst *WsClient) auth(token string) bool {
	oauthToken := &oauth2.Token{AccessToken: token, TokenType: "Bearer"}
	if !oauthToken.Valid() {
		log.Warn("Invalid Token - nil or expired")
		return false
	}

	tc, err := ginkeycloak.GetTokenContainer(oauthToken, ginkeycloak.KeycloakConfig{
		Url:           wst.cd.GetKeycloak().GetUrl(),
		Realm:         wst.cd.GetKeycloak().GetRealm(),
		FullCertsPath: nil,
	})
	if err != nil {
		log.Warnf("Can not extract TokenContainer, caused by: %s", err)
		return false
	}

	isExpired := func(token *ginkeycloak.KeyCloakToken) bool {
		if token.Exp == 0 {
			return false
		}
		now := time.Now()
		fromUnixTimestamp := time.Unix(token.Exp, 0)
		return now.After(fromUnixTimestamp)
	}
	if isExpired(tc.KeyCloakToken) {
		log.Warn(" Keycloak Token has expired")
		return false
	}

	if !tc.Valid() {
		log.Warn("Invalid Token")
		return false
	}

	return true
}

func (wst *WsClient) ReceiveData() {
	defer func() {
		wst.wsHelper.UnregisterChan() <- wst
	}()

	wst.wsHelper.QueuedChan() <- wst

	var req WebsocketReq
	if err := wst.conn.ReadJSON(&req); err != nil {
		log.Error("Unable to read websocket handshake request")
		return
	}

	//验证token
	if ok := wst.auth(req.Token); !ok {
		wst.conn.WriteMessage(websocket.CloseAbnormalClosure, []byte("unauthenticated"))
		log.Error("Unable to authenticate websocket client")
		return
	}

	switch req.Request {
	case ConnTypeHelloWorld:
		wst.connType = req.Request
	default:
		log.Warnf("Unknown websocket connection type: %v", req.Request)
		return
	}

	if req.Data != nil {
		wst.sessionId = req.Data.SessionId
	}

	//握手成功
	wst.wsHelper.RegisterChan() <- wst

	//响应pong
	wst.conn.SetReadDeadline(time.Now().Add(PongWait))
	wst.conn.SetPongHandler(func(appData string) error {
		wst.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	//客户端关闭
	for {
		tp, _, err := wst.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warnf("%v websocket connection read type:%v, error: %v", wst.sessionId+":"+wst.connType, tp, err)
			}
			return
		}
	}
}

func (wst *WsClient) SendData() {
	timer := time.NewTicker(PingPeriod)
	defer func() {
		timer.Stop()
		wst.wsHelper.UnregisterChan() <- wst
	}()

	for {
		select {
		case msg, ok := <-wst.sendChan:
			wst.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				wst.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}

			if err := wst.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-timer.C:
			wst.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			//定时ping client探活
			if err := wst.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (wst *WsClient) Id() string {
	return wst.id
}

func (wst *WsClient) ConnType() string {
	return wst.connType
}

func (wst *WsClient) SendChan() chan<- []byte {
	return wst.sendChan
}

func (wst *WsClient) IsHandshakeTimeout() bool {
	return wst.enterAt.Add(3*time.Second).Before(time.Now()) && wst.connType == ConnTypeNone
}

func (wst *WsClient) CloseWsClient() {
	wst.conn.Close()

	if wst.sendChan != nil {
		close(wst.sendChan)
		wst.sendChan = nil
	}
}
