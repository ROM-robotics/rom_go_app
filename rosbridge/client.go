package rosbridge

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client manages a WebSocket connection to a rosbridge_server.
type Client struct {
	mu   sync.Mutex
	conn *websocket.Conn
	ns   string // robot namespace prefix
	host string
	port int

	connected    bool
	reconnecting bool
	stopCh       chan struct{}

	// Subscribed topic names (full, with namespace)
	topicMap      string
	topicCmdVel   string
	topicTF       string
	topicOdom     string
	topicCtrlOdom string
	topicLaser    string
	topicMapBfp   string

	// cmd_vel publishing
	cmdVelEnabled bool
	desiredTwist  TwistData
	lastTwist     TwistData
	cmdVelTicker  *time.Ticker

	// Stored TF for map→odom
	globalMapOdom TransformStamped

	// Callbacks — set by the robot layer
	OnMap          func(MapData)
	OnTwist        func(TwistData)
	OnTF           func(TFData)
	OnOdom         func(OdomData)
	OnCtrlOdom     func(OdomData)
	OnLaser        func(LaserData)
	OnMapBfp       func(Pose2D)
	OnConnected    func()
	OnDisconnected func()

	// Service response channels
	svcMu      sync.Mutex
	svcPending map[string]chan json.RawMessage
}

// NewClient creates a new rosbridge client.
func NewClient(ns, host string, port int) *Client {
	c := &Client{
		ns:         ns,
		host:       host,
		port:       port,
		stopCh:     make(chan struct{}),
		svcPending: make(map[string]chan json.RawMessage),
	}
	return c
}

// Connect dials the rosbridge WebSocket server.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	url := fmt.Sprintf("ws://%s:%d", c.host, c.port)
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		go c.scheduleReconnect()
		return fmt.Errorf("dial %s: %w", url, err)
	}

	c.conn = conn
	c.connected = true
	go c.readLoop()
	c.startCmdVelPublisher()

	if c.OnConnected != nil {
		go c.OnConnected()
	}
	log.Printf("[rosbridge] Connected to %s (ns=%s)", url, c.ns)
	return nil
}

// Disconnect closes the connection.
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disconnect()
}

func (c *Client) disconnect() {
	if !c.connected {
		return
	}
	c.connected = false

	if c.cmdVelTicker != nil {
		c.cmdVelTicker.Stop()
	}
	select {
	case c.stopCh <- struct{}{}:
	default:
	}

	if c.conn != nil {
		c.conn.Close()
	}

	if c.OnDisconnected != nil {
		go c.OnDisconnected()
	}
	log.Printf("[rosbridge] Disconnected (ns=%s)", c.ns)
}

// IsConnected returns connection state.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *Client) send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// ──────────────────────────── Topic subscriptions

func (c *Client) SubscribeMap(topic string) {
	if topic == "" {
		topic = "/map"
	}
	c.topicMap = c.ns + topic
	c.send(SubscribeMsg(c.topicMap, TypeOccupancyGrid))
}

func (c *Client) SubscribeCmdVel(topic string) {
	if topic == "" {
		topic = "/diff_controller/cmd_vel_unstamped"
	}
	c.topicCmdVel = c.ns + topic
	c.send(SubscribeMsg(c.topicCmdVel, TypeTwist))
}

func (c *Client) SubscribeTF(topic string) {
	if topic == "" {
		topic = "/tf"
	}
	c.topicTF = c.ns + topic
	c.send(SubscribeMsg(c.topicTF, TypeTFMessage))
}

func (c *Client) SubscribeOdom(topic string) {
	if topic == "" {
		topic = "/odom"
	}
	c.topicOdom = c.ns + topic
	c.send(SubscribeMsg(c.topicOdom, TypeOdometry))
}

func (c *Client) SubscribeControllerOdom(topic string) {
	if topic == "" {
		topic = "/diff_controller/odom"
	}
	c.topicCtrlOdom = c.ns + topic
	c.send(SubscribeMsg(c.topicCtrlOdom, TypeOdometry))
}

func (c *Client) SubscribeLaser(topic string) {
	if topic == "" {
		topic = "/scan"
	}
	c.topicLaser = c.ns + topic
	c.send(SubscribeMsg(c.topicLaser, TypeLaserScan))
}

func (c *Client) SubscribeMapBfp(topic string) {
	if topic == "" {
		topic = "/map_bfp_publisher"
	}
	c.topicMapBfp = c.ns + topic
	c.send(SubscribeMsg(c.topicMapBfp, ""))
}

// SubscribeAllTopics subscribes to all standard topics.
func (c *Client) SubscribeAllTopics() {
	c.SubscribeMap("")
	c.SubscribeTF("")
	c.SubscribeOdom("")
	c.SubscribeControllerOdom("")
	c.SubscribeLaser("")
	c.SubscribeMapBfp("")
	c.SubscribeCmdVel("")
}

func (c *Client) UnsubscribeAll() {
	topics := []string{c.topicMap, c.topicCmdVel, c.topicTF, c.topicOdom, c.topicCtrlOdom, c.topicLaser, c.topicMapBfp}
	for _, t := range topics {
		if t != "" {
			c.send(UnsubscribeMsg(t))
		}
	}
}

// ──────────────────────────── cmd_vel publishing

func (c *Client) SetDesiredCmdVel(twist TwistData) {
	c.mu.Lock()
	c.desiredTwist = twist
	c.mu.Unlock()
}

func (c *Client) SetCmdVelEnabled(enabled bool) {
	c.mu.Lock()
	c.cmdVelEnabled = enabled
	c.mu.Unlock()
}

func (c *Client) SetCmdVelTopic(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topicCmdVel = c.ns + topic
}

func (c *Client) startCmdVelPublisher() {
	c.cmdVelTicker = time.NewTicker(50 * time.Millisecond) // 20 Hz
	go func() {
		for {
			select {
			case <-c.stopCh:
				return
			case <-c.cmdVelTicker.C:
				c.publishCmdVelTick()
			}
		}
	}()
}

func (c *Client) publishCmdVelTick() {
	c.mu.Lock()
	if !c.connected || !c.cmdVelEnabled {
		c.mu.Unlock()
		return
	}

	desired := c.desiredTwist
	last := c.lastTwist
	topic := c.topicCmdVel
	c.mu.Unlock()

	if topic == "" {
		return
	}

	// Only publish on change
	if desired.LinearX == last.LinearX && desired.AngularZ == last.AngularZ &&
		desired.LinearY == last.LinearY {
		return
	}

	msg := map[string]interface{}{
		"linear":  map[string]float64{"x": desired.LinearX, "y": desired.LinearY, "z": 0},
		"angular": map[string]float64{"x": 0, "y": 0, "z": desired.AngularZ},
	}
	c.send(PublishMsg(topic, msg))

	c.mu.Lock()
	c.lastTwist = desired
	c.mu.Unlock()
}

// ──────────────────────────── Service calls

// CallService sends a service call and waits for response (with timeout).
func (c *Client) CallService(service string, args interface{}, timeout time.Duration) (json.RawMessage, error) {
	id := fmt.Sprintf("svc_%s_%d", service, time.Now().UnixMilli())
	fullService := c.ns + service

	ch := make(chan json.RawMessage, 1)
	c.svcMu.Lock()
	c.svcPending[id] = ch
	c.svcMu.Unlock()

	defer func() {
		c.svcMu.Lock()
		delete(c.svcPending, id)
		c.svcMu.Unlock()
	}()

	if err := c.send(CallServiceMsg(fullService, args, id)); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("service call %s timed out", service)
	}
}

// Handshake calls /which_name and returns robot namespace + status.
func (c *Client) Handshake() (*HandshakeResponse, error) {
	args := WhichMapsArgs("handshake", "", "", "*#5447972162718281828459#")
	raw, err := c.CallService("/which_name", args, 10*time.Second)
	if err != nil {
		return nil, err
	}

	// Try "values" wrapper first (rosbridge convention)
	var resp struct {
		Values HandshakeResponse `json:"values"`
	}
	if err := json.Unmarshal(raw, &resp); err == nil && resp.Values.RobotNamespace != "" {
		return &resp.Values, nil
	}

	// Fallback: direct parse
	var hs HandshakeResponse
	if err := json.Unmarshal(raw, &hs); err != nil {
		return nil, fmt.Errorf("parse handshake: %w", err)
	}
	return &hs, nil
}

// RequestNavigationMode calls which_maps service with "navi" request.
func (c *Client) RequestNavigationMode() (json.RawMessage, error) {
	args := WhichMapsArgs("navi", "", "", "")
	return c.CallService("/which_maps", args, 10*time.Second)
}

// RequestMappingMode calls which_maps service with "mapping" request.
func (c *Client) RequestMappingMode() (json.RawMessage, error) {
	args := WhichMapsArgs("mapping", "", "", "")
	return c.CallService("/which_maps", args, 10*time.Second)
}

// RequestRemappingMode calls which_maps service with "remapping" request.
func (c *Client) RequestRemappingMode() (json.RawMessage, error) {
	args := WhichMapsArgs("remapping", "", "", "")
	return c.CallService("/which_maps", args, 10*time.Second)
}

// RequestWhichMaps asks the robot what maps it has.
func (c *Client) RequestWhichMaps() (*WhichMapsResponse, error) {
	args := WhichMapsArgs("which_maps", "", "", "")
	raw, err := c.CallService("/which_maps", args, 10*time.Second)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values WhichMapsResponse `json:"values"`
	}
	if err := json.Unmarshal(raw, &resp); err == nil && resp.Values.TotalMaps > 0 {
		return &resp.Values, nil
	}

	var wm WhichMapsResponse
	json.Unmarshal(raw, &wm)
	return &wm, nil
}

// SaveMap saves the current map with the given name.
func (c *Client) SaveMap(name string) (json.RawMessage, error) {
	args := WhichMapsArgs("save_map", name, "", "")
	return c.CallService("/which_maps", args, 30*time.Second)
}

// SelectMap selects/opens a map by name.
func (c *Client) SelectMap(name string) (json.RawMessage, error) {
	args := WhichMapsArgs("select_map", "", name, "")
	return c.CallService("/which_maps", args, 30*time.Second)
}

// ──────────────────────────── construct_yaml_and_bt service calls

func (c *Client) sendNavPoints(requestString string, pointsKey string, points interface{}) (json.RawMessage, error) {
	args := map[string]interface{}{
		"request_string": requestString,
		pointsKey:        points,
	}
	return c.CallService("/construct_yaml_and_bt", args, 15*time.Second)
}

func (c *Client) AddWaypoints(pts []NavigationPoint) (json.RawMessage, error) {
	return c.sendNavPoints("add_waypoints", "waypoints", WaypointToJSON(pts))
}

func (c *Client) AddServicePoints(pts []NavigationPoint) (json.RawMessage, error) {
	return c.sendNavPoints("add_servicepoints", "servicepoints", WaypointToJSON(pts))
}

func (c *Client) AddPatrolPoints(pts []NavigationPoint) (json.RawMessage, error) {
	return c.sendNavPoints("add_patrolpoints", "patrolpoints", WaypointToJSON(pts))
}

func (c *Client) AddPathPoints(pts []NavigationPoint) (json.RawMessage, error) {
	return c.sendNavPoints("add_pathpoints", "pathpoints", WaypointToJSON(pts))
}

func (c *Client) SaveWallObstacles(walls []WallObstacle) (json.RawMessage, error) {
	return c.sendNavPoints("save_obstacles", "obstacles", WallObstaclesToJSON(walls))
}

func (c *Client) ClearWallObstacles() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "clear_obstacles"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GetWaypoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "get_waypoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GetServicePoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "get_servicepoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GetPatrolPoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "get_patrolpoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GetPathPoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "get_pathpoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GoAllWaypoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "go_all_waypoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GoAllServicePoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "go_all_servicepoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GoAllPatrolPoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "go_all_patrolpoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

func (c *Client) GoAllPathPoints() (json.RawMessage, error) {
	args := map[string]interface{}{"request_string": "go_all_pathpoints"}
	return c.CallService("/construct_yaml_and_bt", args, 10*time.Second)
}

// ──────────────────────────── which_tasks service calls

func (c *Client) RequestTask(taskName, settings string) (*WhichTaskResponse, error) {
	args := WhichTaskArgs(taskName, settings)
	raw, err := c.CallService("/which_tasks", args, 30*time.Second)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values WhichTaskResponse `json:"values"`
	}
	if err := json.Unmarshal(raw, &resp); err == nil && resp.Values.TaskName != "" {
		return &resp.Values, nil
	}

	var wt WhichTaskResponse
	json.Unmarshal(raw, &wt)
	return &wt, nil
}

func (c *Client) RequestSettingsRead() (*WhichTaskResponse, error) {
	return c.RequestTask("settings_read", "")
}

func (c *Client) RequestSettingsSave(yaml string) (*WhichTaskResponse, error) {
	return c.RequestTask("settings_save", yaml)
}

func (c *Client) RequestReboot() (*WhichTaskResponse, error) {
	return c.RequestTask("reboot", "")
}

func (c *Client) RequestPowerOff() (*WhichTaskResponse, error) {
	return c.RequestTask("poweroff", "")
}

func (c *Client) SendVoiceCommand(cmd string) (*WhichTaskResponse, error) {
	return c.RequestTask("voice_command", cmd)
}

// RequestWhichMapsNames returns just the map names as a string slice.
func (c *Client) RequestWhichMapsNames() ([]string, error) {
	resp, err := c.RequestWhichMaps()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return []string{}, nil
	}
	return resp.MapNames, nil
}

// ──────────────────────────── Read loop — parse incoming messages

func (c *Client) readLoop() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			wasConnected := c.connected
			c.connected = false
			c.mu.Unlock()

			if wasConnected {
				if c.OnDisconnected != nil {
					go c.OnDisconnected()
				}
				go c.scheduleReconnect()
			}
			return
		}
		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(raw []byte) {
	var envelope struct {
		Op    string          `json:"op"`
		Topic string          `json:"topic"`
		ID    string          `json:"id"`
		Msg   json.RawMessage `json:"msg"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return
	}

	switch envelope.Op {
	case "publish":
		c.handlePublish(envelope.Topic, envelope.Msg)
	case "service_response":
		c.handleServiceResponse(envelope.ID, raw)
	}
}

func (c *Client) handlePublish(topic string, msg json.RawMessage) {
	switch topic {
	case c.topicMap:
		c.parseMap(msg)
	case c.topicCmdVel:
		c.parseTwist(msg)
	case c.topicTF:
		c.parseTF(msg)
	case c.topicOdom:
		c.parseOdom(msg, false)
	case c.topicCtrlOdom:
		c.parseOdom(msg, true)
	case c.topicLaser:
		c.parseLaser(msg)
	case c.topicMapBfp:
		c.parseMapBfp(msg)
	}
}

func (c *Client) handleServiceResponse(id string, raw []byte) {
	c.svcMu.Lock()
	ch, ok := c.svcPending[id]
	c.svcMu.Unlock()
	if ok {
		// Extract the full response object
		ch <- json.RawMessage(raw)
	}
}

// ──────────────────────────── Message parsers

func (c *Client) parseMap(msg json.RawMessage) {
	if c.OnMap == nil {
		return
	}

	var grid struct {
		Info struct {
			Width      int     `json:"width"`
			Height     int     `json:"height"`
			Resolution float64 `json:"resolution"`
			Origin     struct {
				Position struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"position"`
			} `json:"origin"`
		} `json:"info"`
		Data []int `json:"data"`
	}
	if err := json.Unmarshal(msg, &grid); err != nil {
		return
	}

	data := make([]int8, len(grid.Data))
	for i, v := range grid.Data {
		if v > 127 {
			v -= 256
		}
		data[i] = int8(v)
	}

	c.OnMap(MapData{
		Width:      grid.Info.Width,
		Height:     grid.Info.Height,
		Resolution: grid.Info.Resolution,
		OriginX:    grid.Info.Origin.Position.X,
		OriginY:    grid.Info.Origin.Position.Y,
		Data:       data,
	})
}

func (c *Client) parseTwist(msg json.RawMessage) {
	if c.OnTwist == nil {
		return
	}
	var m struct {
		Linear  Vector3 `json:"linear"`
		Angular Vector3 `json:"angular"`
	}
	if err := json.Unmarshal(msg, &m); err != nil {
		return
	}
	c.OnTwist(TwistData{
		LinearX: m.Linear.X, LinearY: m.Linear.Y, LinearZ: m.Linear.Z,
		AngularX: m.Angular.X, AngularY: m.Angular.Y, AngularZ: m.Angular.Z,
	})
}

func (c *Client) parseTF(msg json.RawMessage) {
	if c.OnTF == nil {
		return
	}

	var tfMsg struct {
		Transforms []struct {
			Header struct {
				FrameID string `json:"frame_id"`
			} `json:"header"`
			ChildFrameID string `json:"child_frame_id"`
			Transform    struct {
				Translation Vector3    `json:"translation"`
				Rotation    Quaternion `json:"rotation"`
			} `json:"transform"`
		} `json:"transforms"`
	}
	if err := json.Unmarshal(msg, &tfMsg); err != nil {
		return
	}

	var tfData TFData
	emitTF := false

	for _, t := range tfMsg.Transforms {
		parent := t.Header.FrameID
		child := t.ChildFrameID

		if parent == "map" && child == "odom" {
			c.mu.Lock()
			c.globalMapOdom = TransformStamped{}
			c.globalMapOdom.Transform.Translation = t.Transform.Translation
			c.globalMapOdom.Transform.Rotation = t.Transform.Rotation
			c.mu.Unlock()

			tfData.MapOdomTx = t.Transform.Translation.X
			tfData.MapOdomTy = t.Transform.Translation.Y
			tfData.MapOdomTz = t.Transform.Translation.Z
			tfData.MapOdomRx = t.Transform.Rotation.X
			tfData.MapOdomRy = t.Transform.Rotation.Y
			tfData.MapOdomRz = t.Transform.Rotation.Z
			tfData.MapOdomRw = t.Transform.Rotation.W
		} else if parent == "odom" && child == "base_footprint" {
			c.mu.Lock()
			mo := c.globalMapOdom
			c.mu.Unlock()

			tfData.MapOdomTx = mo.Transform.Translation.X
			tfData.MapOdomTy = mo.Transform.Translation.Y
			tfData.MapOdomTz = mo.Transform.Translation.Z
			tfData.MapOdomRx = mo.Transform.Rotation.X
			tfData.MapOdomRy = mo.Transform.Rotation.Y
			tfData.MapOdomRz = mo.Transform.Rotation.Z
			tfData.MapOdomRw = mo.Transform.Rotation.W

			tfData.BfpTx = t.Transform.Translation.X
			tfData.BfpTy = t.Transform.Translation.Y
			tfData.BfpTz = t.Transform.Translation.Z
			tfData.BfpRx = t.Transform.Rotation.X
			tfData.BfpRy = t.Transform.Rotation.Y
			tfData.BfpRz = t.Transform.Rotation.Z
			tfData.BfpRw = t.Transform.Rotation.W
			tfData.BfpYaw = math.Atan2(
				2.0*(t.Transform.Rotation.W*t.Transform.Rotation.Z+t.Transform.Rotation.X*t.Transform.Rotation.Y),
				1.0-2.0*(t.Transform.Rotation.Y*t.Transform.Rotation.Y+t.Transform.Rotation.Z*t.Transform.Rotation.Z),
			)
			emitTF = true
		}
	}

	if emitTF {
		c.OnTF(tfData)
	}
}

func (c *Client) parseOdom(msg json.RawMessage, isController bool) {
	var odom Odometry
	if err := json.Unmarshal(msg, &odom); err != nil {
		return
	}
	data := OdomFromMsg(odom)

	if isController {
		if c.OnCtrlOdom != nil {
			c.OnCtrlOdom(data)
		}
	} else {
		if c.OnOdom != nil {
			c.OnOdom(data)
		}
	}
}

func (c *Client) parseLaser(msg json.RawMessage) {
	if c.OnLaser == nil {
		return
	}
	var scan LaserScan
	if err := json.Unmarshal(msg, &scan); err != nil {
		return
	}
	c.OnLaser(LaserData{
		FrameID:        scan.Header.FrameID,
		AngleMin:       scan.AngleMin,
		AngleMax:       scan.AngleMax,
		AngleIncrement: scan.AngleIncrement,
		RangeMin:       scan.RangeMin,
		RangeMax:       scan.RangeMax,
		Ranges:         scan.Ranges,
	})
}

func (c *Client) parseMapBfp(msg json.RawMessage) {
	if c.OnMapBfp == nil {
		return
	}
	var p Pose2D
	if err := json.Unmarshal(msg, &p); err != nil {
		return
	}
	c.OnMapBfp(p)
}

// ──────────────────────────── Reconnect logic

func (c *Client) scheduleReconnect() {
	time.Sleep(3 * time.Second)
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	log.Printf("[rosbridge] Reconnecting to %s:%d ...", c.host, c.port)
	c.Connect()
}
