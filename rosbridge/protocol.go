package rosbridge

import "encoding/json"

// ──────────────────────────── Rosbridge JSON protocol helpers

// SubscribeMsg creates a rosbridge subscribe message.
func SubscribeMsg(topic, msgType string) []byte {
	msg := map[string]interface{}{
		"op":    "subscribe",
		"topic": topic,
		"type":  msgType,
	}
	b, _ := json.Marshal(msg)
	return b
}

// UnsubscribeMsg creates a rosbridge unsubscribe message.
func UnsubscribeMsg(topic string) []byte {
	msg := map[string]interface{}{
		"op":    "unsubscribe",
		"topic": topic,
	}
	b, _ := json.Marshal(msg)
	return b
}

// PublishMsg creates a rosbridge publish message.
func PublishMsg(topic string, data interface{}) []byte {
	msg := map[string]interface{}{
		"op":    "publish",
		"topic": topic,
		"msg":   data,
	}
	b, _ := json.Marshal(msg)
	return b
}

// CallServiceMsg creates a rosbridge call_service message.
func CallServiceMsg(service string, args interface{}, id string) []byte {
	msg := map[string]interface{}{
		"op":      "call_service",
		"service": service,
		"args":    args,
		"id":      id,
	}
	b, _ := json.Marshal(msg)
	return b
}

// ──────────────────────────── Topic type constants

const (
	TypeOccupancyGrid = "nav_msgs/msg/OccupancyGrid"
	TypeOdometry      = "nav_msgs/msg/Odometry"
	TypeTFMessage     = "tf2_msgs/msg/TFMessage"
	TypeLaserScan     = "sensor_msgs/msg/LaserScan"
	TypeTwist         = "geometry_msgs/msg/Twist"
)

// ──────────────────────────── which_maps service args builder

func WhichMapsArgs(requestString, mapSave, mapSelect, token string) map[string]interface{} {
	return map[string]interface{}{
		"request_string":     requestString,
		"map_name_to_save":   mapSave,
		"map_name_to_select": mapSelect,
		"login_access_token": token,
	}
}

// ──────────────────────────── which_tasks service args builder

func WhichTaskArgs(taskName, settingsData string) map[string]interface{} {
	return map[string]interface{}{
		"task_name":     taskName,
		"settings_data": settingsData,
	}
}

// ──────────────────────────── construct_yaml_and_bt navigation point builders

func WaypointToJSON(pts []NavigationPoint) []map[string]interface{} {
	result := make([]map[string]interface{}, len(pts))
	for i, p := range pts {
		result[i] = map[string]interface{}{
			"name":            p.Name,
			"image_x_px":      p.ImageXPx,
			"image_y_px":      p.ImageYPx,
			"image_theta_deg": p.ImageThetaDeg,
			"world_x_m":       p.WorldXM,
			"world_y_m":       p.WorldYM,
			"world_theta_rad": p.WorldThetaRad,
		}
	}
	return result
}

func WallObstaclesToJSON(walls []WallObstacle) []map[string]interface{} {
	result := make([]map[string]interface{}, len(walls))
	for i, w := range walls {
		result[i] = map[string]interface{}{
			"image_x_px_start": w.ImageXPxStart,
			"image_y_px_start": w.ImageYPxStart,
			"image_x_px_end":   w.ImageXPxEnd,
			"image_y_px_end":   w.ImageYPxEnd,
			"world_x_m_start":  w.WorldXMStart,
			"world_y_m_start":  w.WorldYMStart,
			"world_x_m_end":    w.WorldXMEnd,
			"world_y_m_end":    w.WorldYMEnd,
		}
	}
	return result
}
