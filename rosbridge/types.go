package rosbridge

import "math"

// ──────────────────────────── Geometry primitives

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Quaternion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

// Yaw extracts yaw (radians) from a quaternion.
func (q Quaternion) Yaw() float64 {
	siny := 2.0 * (q.W*q.Z + q.X*q.Y)
	cosy := 1.0 - 2.0*(q.Y*q.Y+q.Z*q.Z)
	return math.Atan2(siny, cosy)
}

type Pose struct {
	Position    Vector3    `json:"position"`
	Orientation Quaternion `json:"orientation"`
}

type Twist struct {
	Linear  Vector3 `json:"linear"`
	Angular Vector3 `json:"angular"`
}

type Stamp struct {
	Sec     int `json:"sec"`
	Nanosec int `json:"nanosec,omitempty"`
	Nsec    int `json:"nsec,omitempty"` // ROS1 compat
}

func (s Stamp) NanosecValue() int {
	if s.Nanosec != 0 {
		return s.Nanosec
	}
	return s.Nsec
}

type Header struct {
	Stamp   Stamp  `json:"stamp"`
	FrameID string `json:"frame_id"`
}

// ──────────────────────────── TwistData (cmd_vel)

type TwistData struct {
	LinearX  float64 `json:"linear_x"`
	LinearY  float64 `json:"linear_y"`
	LinearZ  float64 `json:"linear_z"`
	AngularX float64 `json:"angular_x"`
	AngularY float64 `json:"angular_y"`
	AngularZ float64 `json:"angular_z"`
}

// ──────────────────────────── OccupancyGrid (map)

type MapInfo struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Resolution float64 `json:"resolution"`
	Origin     Pose    `json:"origin"`
}

type OccupancyGrid struct {
	Header Header  `json:"header"`
	Info   MapInfo `json:"info"`
	Data   []int8  `json:"data"`
}

// MapData is the simplified map representation sent to the browser.
type MapData struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Resolution float64 `json:"resolution"`
	OriginX    float64 `json:"origin_x"`
	OriginY    float64 `json:"origin_y"`
	Data       []int8  `json:"data"`
}

// ──────────────────────────── Odometry

type PoseWithCovariance struct {
	Pose       Pose      `json:"pose"`
	Covariance []float64 `json:"covariance"`
}

type TwistWithCovariance struct {
	Twist      Twist     `json:"twist"`
	Covariance []float64 `json:"covariance"`
}

type Odometry struct {
	Header       Header              `json:"header"`
	ChildFrameID string              `json:"child_frame_id"`
	Pose         PoseWithCovariance  `json:"pose"`
	Twist        TwistWithCovariance `json:"twist"`
}

// OdomData is a simplified odometry for the browser.
type OdomData struct {
	FrameID      string  `json:"frame_id"`
	ChildFrameID string  `json:"child_frame_id"`
	PosX         float64 `json:"pos_x"`
	PosY         float64 `json:"pos_y"`
	PosZ         float64 `json:"pos_z"`
	OrientX      float64 `json:"orient_x"`
	OrientY      float64 `json:"orient_y"`
	OrientZ      float64 `json:"orient_z"`
	OrientW      float64 `json:"orient_w"`
	Yaw          float64 `json:"yaw"`
	LinearX      float64 `json:"linear_x"`
	LinearY      float64 `json:"linear_y"`
	AngularZ     float64 `json:"angular_z"`
}

func OdomFromMsg(o Odometry) OdomData {
	return OdomData{
		FrameID:      o.Header.FrameID,
		ChildFrameID: o.ChildFrameID,
		PosX:         o.Pose.Pose.Position.X,
		PosY:         o.Pose.Pose.Position.Y,
		PosZ:         o.Pose.Pose.Position.Z,
		OrientX:      o.Pose.Pose.Orientation.X,
		OrientY:      o.Pose.Pose.Orientation.Y,
		OrientZ:      o.Pose.Pose.Orientation.Z,
		OrientW:      o.Pose.Pose.Orientation.W,
		Yaw:          o.Pose.Pose.Orientation.Yaw(),
		LinearX:      o.Twist.Twist.Linear.X,
		LinearY:      o.Twist.Twist.Linear.Y,
		AngularZ:     o.Twist.Twist.Angular.Z,
	}
}

// ──────────────────────────── TF

type TransformStamped struct {
	Header       Header `json:"header"`
	ChildFrameID string `json:"child_frame_id"`
	Transform    struct {
		Translation Vector3    `json:"translation"`
		Rotation    Quaternion `json:"rotation"`
	} `json:"transform"`
}

type TFMessage struct {
	Transforms []TransformStamped `json:"transforms"`
}

// TFData holds the two transforms we care about.
type TFData struct {
	MapOdomTx float64 `json:"map_odom_tx"`
	MapOdomTy float64 `json:"map_odom_ty"`
	MapOdomTz float64 `json:"map_odom_tz"`
	MapOdomRx float64 `json:"map_odom_rx"`
	MapOdomRy float64 `json:"map_odom_ry"`
	MapOdomRz float64 `json:"map_odom_rz"`
	MapOdomRw float64 `json:"map_odom_rw"`
	BfpTx     float64 `json:"bfp_tx"`
	BfpTy     float64 `json:"bfp_ty"`
	BfpTz     float64 `json:"bfp_tz"`
	BfpRx     float64 `json:"bfp_rx"`
	BfpRy     float64 `json:"bfp_ry"`
	BfpRz     float64 `json:"bfp_rz"`
	BfpRw     float64 `json:"bfp_rw"`
	BfpYaw    float64 `json:"bfp_yaw"`
}

// ──────────────────────────── LaserScan

type LaserScan struct {
	Header         Header    `json:"header"`
	AngleMin       float64   `json:"angle_min"`
	AngleMax       float64   `json:"angle_max"`
	AngleIncrement float64   `json:"angle_increment"`
	RangeMin       float64   `json:"range_min"`
	RangeMax       float64   `json:"range_max"`
	Ranges         []float64 `json:"ranges"`
	Intensities    []float64 `json:"intensities"`
}

// LaserData is simplified for the browser.
type LaserData struct {
	FrameID        string    `json:"frame_id"`
	AngleMin       float64   `json:"angle_min"`
	AngleMax       float64   `json:"angle_max"`
	AngleIncrement float64   `json:"angle_increment"`
	RangeMin       float64   `json:"range_min"`
	RangeMax       float64   `json:"range_max"`
	Ranges         []float64 `json:"ranges"`
}

// ──────────────────────────── Pose2D (map→base_footprint shortcut)

type Pose2D struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Theta float64 `json:"theta"`
}

// ──────────────────────────── Navigation points

type NavigationPoint struct {
	Name          string  `json:"name"`
	ImageXPx      float64 `json:"image_x_px"`
	ImageYPx      float64 `json:"image_y_px"`
	ImageThetaDeg float64 `json:"image_theta_deg"`
	WorldXM       float64 `json:"world_x_m"`
	WorldYM       float64 `json:"world_y_m"`
	WorldThetaRad float64 `json:"world_theta_rad"`
}

type WallObstacle struct {
	ImageXPxStart float64 `json:"image_x_px_start"`
	ImageYPxStart float64 `json:"image_y_px_start"`
	ImageXPxEnd   float64 `json:"image_x_px_end"`
	ImageYPxEnd   float64 `json:"image_y_px_end"`
	WorldXMStart  float64 `json:"world_x_m_start"`
	WorldYMStart  float64 `json:"world_y_m_start"`
	WorldXMEnd    float64 `json:"world_x_m_end"`
	WorldYMEnd    float64 `json:"world_y_m_end"`
}

// ──────────────────────────── Service response types

type WhichMapsResponse struct {
	TotalMaps int      `json:"total_maps"`
	MapNames  []string `json:"map_names"`
}

type HandshakeResponse struct {
	RobotNamespace string  `json:"robot_namespace"`
	Status         int     `json:"status"`
	RobotDiameter  float64 `json:"robot_diameter"`
}

type WhichTaskResponse struct {
	TaskName         string `json:"task_name"`
	Status           int    `json:"status"`
	ResponseSettings string `json:"response_settings"`
}
