# ROM Dynamics Multi-Robot Control — Go Web Server

A web-based multi-robot fleet management application rewritten from the original Qt6 C++ GUI. Communicates with ROS 2 robots via the **rosbridge** WebSocket protocol.

## Features

- **Multi-robot management** — Add, remove, and switch between robots
- **Real-time map display** — OccupancyGrid rendered on HTML5 Canvas with zoom/pan
- **Virtual joystick** — Touch and mouse support for manual robot control
- **Navigation system** — Waypoints, Service Points, Patrol Points, Path Points, Wall Obstacles
- **Mode switching** — Navigation, Mapping, Remapping, Map Editing, Settings
- **Map management** — List, open, and save maps via ROS services
- **Velocity graphs** — Real-time Chart.js plots for velocity and position
- **Speech-to-text** — Whisper integration for voice commands
- **Diagnostics** — TF/Odom/Map/Laser frequency monitoring

## Architecture

```
Browser  ←→  Go Server  ←→  rosbridge_server  ←→  ROS 2 (robot)
(WebSocket)    (HTTP)       (WebSocket:9090)
```

The Go server acts as middleware:
- Serves the web UI (Go Templates + HTMX)
- Maintains WebSocket connections to each robot's rosbridge
- Bridges real-time data to the browser via a server WebSocket

## Prerequisites

- Go 1.21+
- ROS 2 robot(s) running `rosbridge_server` (port 9090)
- (Optional) `whisper` CLI + model for speech-to-text
- (Optional) `ffmpeg` for audio format conversion

## Quick Start

```bash
# Clone and build
cd ~/Desktop/rom_go_app
go mod tidy
make build

# Run
make run
# or
make dev

# Open browser
# http://localhost:8080
```

## Configuration (Environment Variables)

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `ROSBRIDGE_PORT` | `9090` | Default rosbridge port |
| `WHISPER_BIN` | — | Path to whisper binary |
| `WHISPER_MODEL` | — | Path to whisper model file |
| `SPEECH_LOG_DIR` | `/tmp/rom_speech` | Directory for speech recordings |

## Project Structure

```
rom_go_app/
├── main.go                 # Entry point, HTTP router, embed FS
├── config/config.go        # Configuration from environment
├── rosbridge/
│   ├── types.go            # ROS message types (OccupancyGrid, Odom, TF, etc.)
│   ├── protocol.go         # Rosbridge JSON protocol helpers
│   └── client.go           # WebSocket client to rosbridge
├── robot/
│   ├── robot.go            # Robot model with all sensor state
│   ├── manager.go          # Thread-safe multi-robot registry + broadcast
│   └── navigation.go       # Navigation point CRUD & ROS service calls
├── handlers/
│   ├── pages.go            # Page rendering handlers
│   ├── robot_api.go        # Robot CRUD REST API + HTMX partials
│   ├── map_api.go          # Map list/save/open + mode switching
│   ├── nav_api.go          # Navigation point API
│   ├── ws_handler.go       # Browser WebSocket handler (bridge)
│   └── speech_api.go       # Speech recording & whisper transcription
├── templates/
│   ├── layout.html         # Base HTML layout (CDN: HTMX, Chart.js)
│   ├── index.html          # Main app UI
│   ├── partials/           # HTMX response fragments
│   └── dialogs/            # Modal dialog fragments
├── static/
│   ├── css/style.css       # Dark theme CSS
│   └── js/
│       ├── app.js          # Main application controller
│       ├── websocket.js    # Browser WebSocket client
│       ├── map_canvas.js   # Canvas-based map renderer
│       ├── joystick.js     # Virtual joystick (touch+mouse)
│       ├── graphs.js       # Chart.js velocity/position graphs
│       ├── speech.js       # MediaRecorder speech capture
│       └── notifications.js # Toast notification system
├── Makefile
└── README.md
```

## ROS Topics & Services

**Subscribed Topics:**
- `/{ns}/map` — OccupancyGrid
- `/{ns}/diff_controller/cmd_vel_unstamped` — Twist (velocity feedback)
- `/{ns}/tf` — TFMessage
- `/{ns}/odom` — Odometry
- `/{ns}/diff_controller/odom` — Controller Odometry
- `/{ns}/scan` — LaserScan
- `/{ns}/map_bfp_publisher` — Pose2D

**Published Topics:**
- `/{ns}/diff_controller/cmd_vel_unstamped` — Twist (at 20 Hz)

**Services Called:**
- `/{ns}/which_maps` — List/save/select maps, mode switching
- `/{ns}/which_tasks` — Task execution, settings, power management
- `/{ns}/construct_yaml_and_bt` — Navigation point CRUD

## Cross-Compilation

```bash
# For ARM64 (e.g., Raspberry Pi 4)
make build-arm

# The binary at build/rom_dynamics_web_arm64 is self-contained
# (templates & static files are embedded via go:embed)
```

## License

Internal use — ROM Dynamics
