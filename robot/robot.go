package robot

import (
	"sync"
	"time"

	"rom_go_app/rosbridge"
)

// Mode represents the application mode.
type Mode string

const (
	ModeNavigation Mode = "navigation"
	ModeMapping    Mode = "mapping"
	ModeRemapping  Mode = "remapping"
	ModeMapEditing Mode = "mapediting"
	ModeSettings   Mode = "settings"
)

// Robot holds all state for a single robot.
type Robot struct {
	mu sync.RWMutex

	ID        string `json:"id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`

	Radius    float64 `json:"radius"`
	Connected bool    `json:"connected"`

	// ROS bridge client
	Client *rosbridge.Client `json:"-"`

	// Latest sensor data
	Map            rosbridge.MapData   `json:"-"`
	MapReceived    bool                `json:"-"`
	Odom           rosbridge.OdomData  `json:"odom"`
	ControllerOdom rosbridge.OdomData  `json:"controller_odom"`
	TF             rosbridge.TFData    `json:"tf"`
	TFReceived     bool                `json:"-"`
	Laser          rosbridge.LaserData `json:"-"`
	MapBfp         rosbridge.Pose2D    `json:"map_bfp"`

	// Velocity from subscribed cmd_vel
	Velocity rosbridge.TwistData `json:"velocity"`

	// Velocity history for graphs (last N samples)
	VelocityHistory []rosbridge.TwistData `json:"-"`
	MaxHistory      int                   `json:"-"`

	// Navigation points
	Waypoints     []rosbridge.NavigationPoint `json:"waypoints"`
	ServicePoints []rosbridge.NavigationPoint `json:"service_points"`
	PatrolPoints  []rosbridge.NavigationPoint `json:"patrol_points"`
	PathPoints    []rosbridge.NavigationPoint `json:"path_points"`
	WallObstacles []rosbridge.WallObstacle    `json:"wall_obstacles"`

	// Map list cache
	MapList []string `json:"map_list"`

	// User settings
	LinearVelRatio  float64 `json:"linear_vel_ratio"`
	AngularVelRatio float64 `json:"angular_vel_ratio"`

	// Frequency tracking
	lastMapTime   time.Time
	MapHz         int `json:"map_hz"`
	lastTFTime    time.Time
	TFHz          int `json:"tf_hz"`
	lastOdomTime  time.Time
	OdomHz        int `json:"odom_hz"`
	lastLaserTime time.Time
	LaserHz       int `json:"laser_hz"`
}

// NewRobot creates a new Robot and its rosbridge client.
func NewRobot(id, ns, name, ip string, port int) *Robot {
	r := &Robot{
		ID:              id,
		Namespace:       ns,
		Name:            name,
		IP:              ip,
		Port:            port,
		Radius:          0.30,
		MaxHistory:      100,
		VelocityHistory: make([]rosbridge.TwistData, 0, 100),
		LinearVelRatio:  1.0,
		AngularVelRatio: 1.0,
	}

	client := rosbridge.NewClient(ns, ip, port)

	// Wire up callbacks
	client.OnMap = func(m rosbridge.MapData) {
		r.mu.Lock()
		r.Map = m
		r.MapReceived = true
		r.MapHz = r.measureHz(&r.lastMapTime)
		r.mu.Unlock()
	}

	client.OnTwist = func(t rosbridge.TwistData) {
		r.mu.Lock()
		r.Velocity = t
		r.VelocityHistory = append(r.VelocityHistory, t)
		if len(r.VelocityHistory) > r.MaxHistory {
			r.VelocityHistory = r.VelocityHistory[1:]
		}
		r.mu.Unlock()
	}

	client.OnTF = func(tf rosbridge.TFData) {
		r.mu.Lock()
		r.TF = tf
		r.TFReceived = true
		r.TFHz = r.measureHz(&r.lastTFTime)
		r.mu.Unlock()
	}

	client.OnOdom = func(o rosbridge.OdomData) {
		r.mu.Lock()
		r.Odom = o
		r.OdomHz = r.measureHz(&r.lastOdomTime)
		r.mu.Unlock()
	}

	client.OnCtrlOdom = func(o rosbridge.OdomData) {
		r.mu.Lock()
		r.ControllerOdom = o
		r.mu.Unlock()
	}

	client.OnLaser = func(l rosbridge.LaserData) {
		r.mu.Lock()
		r.Laser = l
		r.LaserHz = r.measureHz(&r.lastLaserTime)
		r.mu.Unlock()
	}

	client.OnMapBfp = func(p rosbridge.Pose2D) {
		r.mu.Lock()
		r.MapBfp = p
		r.mu.Unlock()
	}

	client.OnConnected = func() {
		r.mu.Lock()
		r.Connected = true
		r.mu.Unlock()
		client.SubscribeAllTopics()
		client.SetCmdVelEnabled(true)
	}

	client.OnDisconnected = func() {
		r.mu.Lock()
		r.Connected = false
		r.mu.Unlock()
	}

	r.Client = client
	return r
}

func (r *Robot) measureHz(last *time.Time) int {
	now := time.Now()
	if last.IsZero() {
		*last = now
		return 0
	}
	elapsed := now.Sub(*last)
	*last = now
	if elapsed > 0 {
		return int(time.Second / elapsed)
	}
	return 0
}

// GetMap returns a thread-safe copy of the map data.
func (r *Robot) GetMap() rosbridge.MapData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Map
}

// GetVelocityHistory returns a copy of velocity history.
func (r *Robot) GetVelocityHistory() []rosbridge.TwistData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]rosbridge.TwistData, len(r.VelocityHistory))
	copy(cp, r.VelocityHistory)
	return cp
}

// GetSnapshot returns a safe snapshot of the robot state.
func (r *Robot) GetSnapshot() Robot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Robot{
		ID:              r.ID,
		Namespace:       r.Namespace,
		Name:            r.Name,
		IP:              r.IP,
		Port:            r.Port,
		Radius:          r.Radius,
		Connected:       r.Connected,
		MapReceived:     r.MapReceived,
		Odom:            r.Odom,
		ControllerOdom:  r.ControllerOdom,
		TF:              r.TF,
		TFReceived:      r.TFReceived,
		MapBfp:          r.MapBfp,
		Velocity:        r.Velocity,
		Waypoints:       r.Waypoints,
		ServicePoints:   r.ServicePoints,
		PatrolPoints:    r.PatrolPoints,
		PathPoints:      r.PathPoints,
		WallObstacles:   r.WallObstacles,
		MapList:         r.MapList,
		LinearVelRatio:  r.LinearVelRatio,
		AngularVelRatio: r.AngularVelRatio,
		MapHz:           r.MapHz,
		TFHz:            r.TFHz,
		OdomHz:          r.OdomHz,
		LaserHz:         r.LaserHz,
	}
}

// GetMapList returns a copy of the robot's map list.
func (r *Robot) GetMapList() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.MapList))
	copy(out, r.MapList)
	return out
}

// SetMapList sets the robot's map list.
func (r *Robot) SetMapList(maps []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.MapList = maps
}

// SetVelocity sets the desired velocity through the rosbridge client.
func (r *Robot) SetVelocity(linearX, angularZ float64) {
	r.mu.RLock()
	lr := r.LinearVelRatio
	ar := r.AngularVelRatio
	r.mu.RUnlock()

	r.Client.SetDesiredCmdVel(rosbridge.TwistData{
		LinearX:  linearX * lr,
		AngularZ: angularZ * ar,
	})
}

// StopConnection disconnects the robot.
func (r *Robot) StopConnection() {
	r.Client.UnsubscribeAll()
	r.Client.Disconnect()
}

// SetRadius sets the robot's radius in meters.
func (r *Robot) SetRadius(radius float64) {
	r.mu.Lock()
	r.Radius = radius
	r.mu.Unlock()
}

// ImportPoints bulk-imports navigation points by type.
func (r *Robot) ImportPoints(pointType string, points []rosbridge.NavigationPoint, walls []rosbridge.WallObstacle) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch pointType {
	case "waypoint":
		r.Waypoints = points
	case "service_point":
		r.ServicePoints = points
	case "patrol_point":
		r.PatrolPoints = points
	case "path_point":
		r.PathPoints = points
	case "wall":
		r.WallObstacles = walls
	}
}
