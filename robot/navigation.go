package robot

import (
	"fmt"
	"sync"

	"rom_go_app/rosbridge"
)

// NavigationManager handles navigation point operations across robots.
type NavigationManager struct {
	mu sync.RWMutex
}

// NewNavigationManager creates a NavigationManager.
func NewNavigationManager() *NavigationManager {
	return &NavigationManager{}
}

// ──────────────────────────── Add points

// AddWaypoint adds a waypoint to the robot, with validation.
func (nm *NavigationManager) AddWaypoint(rb *Robot, name string, x, y, theta float64) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	pt, err := nm.validateAndCreate(rb, "waypoint", name, x, y, theta)
	if err != nil {
		return err
	}
	rb.mu.Lock()
	rb.Waypoints = append(rb.Waypoints, pt)
	rb.mu.Unlock()
	return nil
}

// AddServicePoint adds a service point to the robot.
func (nm *NavigationManager) AddServicePoint(rb *Robot, name string, x, y, theta float64) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	pt, err := nm.validateAndCreate(rb, "servicepoint", name, x, y, theta)
	if err != nil {
		return err
	}
	rb.mu.Lock()
	rb.ServicePoints = append(rb.ServicePoints, pt)
	rb.mu.Unlock()
	return nil
}

// AddPatrolPoint adds a patrol point to the robot.
func (nm *NavigationManager) AddPatrolPoint(rb *Robot, name string, x, y, theta float64) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	pt, err := nm.validateAndCreate(rb, "patrolpoint", name, x, y, theta)
	if err != nil {
		return err
	}
	rb.mu.Lock()
	rb.PatrolPoints = append(rb.PatrolPoints, pt)
	rb.mu.Unlock()
	return nil
}

// AddPathPoint adds a path point to the robot.
func (nm *NavigationManager) AddPathPoint(rb *Robot, name string, x, y, theta float64) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	pt, err := nm.validateAndCreate(rb, "pathpoint", name, x, y, theta)
	if err != nil {
		return err
	}
	rb.mu.Lock()
	rb.PathPoints = append(rb.PathPoints, pt)
	rb.mu.Unlock()
	return nil
}

// AddWallObstacle adds a wall obstacle to the robot.
func (nm *NavigationManager) AddWallObstacle(rb *Robot, name string, x1, y1, x2, y2 float64) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if name == "" {
		return fmt.Errorf("wall obstacle name cannot be empty")
	}

	wall := rosbridge.WallObstacle{
		WorldXMStart: x1, WorldYMStart: y1,
		WorldXMEnd: x2, WorldYMEnd: y2,
	}
	rb.mu.Lock()
	rb.WallObstacles = append(rb.WallObstacles, wall)
	rb.mu.Unlock()
	return nil
}

// ──────────────────────────── Send points to robot via rosbridge

// SendWaypointsToRobot sends all waypoints to the robot's rosbridge.
func (nm *NavigationManager) SendWaypointsToRobot(rb *Robot) error {
	rb.mu.RLock()
	pts := make([]rosbridge.NavigationPoint, len(rb.Waypoints))
	copy(pts, rb.Waypoints)
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.AddWaypoints(pts)
	return err
}

// SendServicePointsToRobot sends all service points.
func (nm *NavigationManager) SendServicePointsToRobot(rb *Robot) error {
	rb.mu.RLock()
	pts := make([]rosbridge.NavigationPoint, len(rb.ServicePoints))
	copy(pts, rb.ServicePoints)
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.AddServicePoints(pts)
	return err
}

// SendPatrolPointsToRobot sends all patrol points.
func (nm *NavigationManager) SendPatrolPointsToRobot(rb *Robot) error {
	rb.mu.RLock()
	pts := make([]rosbridge.NavigationPoint, len(rb.PatrolPoints))
	copy(pts, rb.PatrolPoints)
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.AddPatrolPoints(pts)
	return err
}

// SendPathPointsToRobot sends all path points.
func (nm *NavigationManager) SendPathPointsToRobot(rb *Robot) error {
	rb.mu.RLock()
	pts := make([]rosbridge.NavigationPoint, len(rb.PathPoints))
	copy(pts, rb.PathPoints)
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.AddPathPoints(pts)
	return err
}

// SendWallObstaclesToRobot sends wall obstacles.
func (nm *NavigationManager) SendWallObstaclesToRobot(rb *Robot) error {
	rb.mu.RLock()
	walls := make([]rosbridge.WallObstacle, len(rb.WallObstacles))
	copy(walls, rb.WallObstacles)
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.SaveWallObstacles(walls)
	return err
}

// ──────────────────────────── Request points from robot

// RequestWaypoints fetches waypoints from the robot.
func (nm *NavigationManager) RequestWaypoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	// The response is handled via service response — the caller
	// would need to parse the result. For now, fire and forget.
	_, err := client.GetWaypoints()
	return err
}

// RequestServicePoints fetches service points from the robot.
func (nm *NavigationManager) RequestServicePoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GetServicePoints()
	return err
}

// RequestPatrolPoints fetches patrol points from the robot.
func (nm *NavigationManager) RequestPatrolPoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GetPatrolPoints()
	return err
}

// RequestPathPoints fetches path points from the robot.
func (nm *NavigationManager) RequestPathPoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GetPathPoints()
	return err
}

// ──────────────────────────── Go all points

// GoAllWaypoints triggers the robot to navigate all waypoints.
func (nm *NavigationManager) GoAllWaypoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GoAllWaypoints()
	return err
}

// GoAllServicePoints triggers navigation of all service points.
func (nm *NavigationManager) GoAllServicePoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GoAllServicePoints()
	return err
}

// GoAllPatrolPoints triggers navigation of all patrol points.
func (nm *NavigationManager) GoAllPatrolPoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GoAllPatrolPoints()
	return err
}

// GoAllPathPoints triggers navigation of all path points.
func (nm *NavigationManager) GoAllPathPoints(rb *Robot) error {
	rb.mu.RLock()
	client := rb.Client
	rb.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("robot not connected")
	}
	_, err := client.GoAllPathPoints()
	return err
}

// ──────────────────────────── Clear points

// ClearWaypoints removes all waypoints from the robot.
func (nm *NavigationManager) ClearWaypoints(rb *Robot) {
	rb.mu.Lock()
	rb.Waypoints = nil
	rb.mu.Unlock()
}

// ClearServicePoints removes all service points.
func (nm *NavigationManager) ClearServicePoints(rb *Robot) {
	rb.mu.Lock()
	rb.ServicePoints = nil
	rb.mu.Unlock()
}

// ClearPatrolPoints removes all patrol points.
func (nm *NavigationManager) ClearPatrolPoints(rb *Robot) {
	rb.mu.Lock()
	rb.PatrolPoints = nil
	rb.mu.Unlock()
}

// ClearPathPoints removes all path points.
func (nm *NavigationManager) ClearPathPoints(rb *Robot) {
	rb.mu.Lock()
	rb.PathPoints = nil
	rb.mu.Unlock()
}

// ClearWallObstacles removes all wall obstacles and notifies the robot.
func (nm *NavigationManager) ClearWallObstacles(rb *Robot) error {
	rb.mu.Lock()
	rb.WallObstacles = nil
	client := rb.Client
	rb.mu.Unlock()

	if client != nil && client.IsConnected() {
		_, err := client.ClearWallObstacles()
		return err
	}
	return nil
}

// ClearAllPoints removes all navigation points from the robot.
func (nm *NavigationManager) ClearAllPoints(rb *Robot) {
	rb.mu.Lock()
	rb.Waypoints = nil
	rb.ServicePoints = nil
	rb.PatrolPoints = nil
	rb.PathPoints = nil
	rb.WallObstacles = nil
	rb.mu.Unlock()
}

// DeletePoint removes a single navigation point by name and type.
func (nm *NavigationManager) DeletePoint(rb *Robot, pointType, name string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	switch pointType {
	case "waypoint":
		rb.Waypoints = removeByName(rb.Waypoints, name)
	case "service_point":
		rb.ServicePoints = removeByName(rb.ServicePoints, name)
	case "patrol_point":
		rb.PatrolPoints = removeByName(rb.PatrolPoints, name)
	case "path_point":
		rb.PathPoints = removeByName(rb.PathPoints, name)
	}
}

func removeByName(pts []rosbridge.NavigationPoint, name string) []rosbridge.NavigationPoint {
	result := make([]rosbridge.NavigationPoint, 0, len(pts))
	for _, p := range pts {
		if p.Name != name {
			result = append(result, p)
		}
	}
	return result
}

// GetCounts returns navigation point counts.
func (nm *NavigationManager) GetCounts(rb *Robot) (waypoints, service, patrol, path, walls int) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return len(rb.Waypoints), len(rb.ServicePoints), len(rb.PatrolPoints), len(rb.PathPoints), len(rb.WallObstacles)
}

// ──────────────────────────── Helpers

func (nm *NavigationManager) validateAndCreate(rb *Robot, pointType, name string, x, y, theta float64) (rosbridge.NavigationPoint, error) {
	if name == "" {
		return rosbridge.NavigationPoint{}, fmt.Errorf("%s name cannot be empty", pointType)
	}

	// Check for duplicate names within the same type
	rb.mu.RLock()
	var existing []rosbridge.NavigationPoint
	switch pointType {
	case "waypoint":
		existing = rb.Waypoints
	case "servicepoint":
		existing = rb.ServicePoints
	case "patrolpoint":
		existing = rb.PatrolPoints
	case "pathpoint":
		existing = rb.PathPoints
	}
	rb.mu.RUnlock()

	for _, pt := range existing {
		if pt.Name == name {
			return rosbridge.NavigationPoint{}, fmt.Errorf("duplicate %s name: %s", pointType, name)
		}
	}

	return rosbridge.NavigationPoint{
		Name:          name,
		WorldXM:       x,
		WorldYM:       y,
		WorldThetaRad: theta,
	}, nil
}
